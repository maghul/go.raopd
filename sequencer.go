package raopd

import (
	"fmt"
	//	"os"
	"sort"
	"time"
)

var seqlog = getLogger("raopd.sequencer")
var debugSequenceLogFlag bool

type sequencer struct {
	// Control channel
	control chan int
	ref     string

	// Internally used
	low     seqno
	lowd    bool
	retries map[seqno]int
	packets map[seqno]*rtpPacket

	sl *sequencelog
}

type rerequest struct {
	first, count seqno
}

func (rr *rerequest) String() string {
	return fmt.Sprintf("ReRequest{first=%d, count=%d}", rr.first, rr.count)
}

// Restart the sequencer. Empty all internal caches
func (s *sequencer) flush() {
	s.control <- 0
}

// Close the sequencer completely.
func (s *sequencer) close() {
	s.control <- 1
}

// Internal functions

// Sequencer is in recovery mode.
func (s *sequencer) inRecovery() bool {
	return len(s.packets) > 0
}

// Internal function to clear all internal caches and
// start over. Called by sequence go-routine when flush has been called.
func (s *sequencer) restartSequencer() {
	s.lowd = false
	s.low = 0
	s.retries = make(map[seqno]int)
	s.packets = make(map[seqno]*rtpPacket)
}

// flush packet cache from seqno and onwards and set low to
// first gap in the cache.
func (s *sequencer) flushCached(sn seqno, outf func(pkt *rtpPacket)) {
	sn--
	for {
		sn++
		pkt, ok := s.packets[sn]
		if ok {
			s.sl.outputPacket(pkt)
			outf(pkt)
			delete(s.packets, sn)
		} else {
			break
		}
	}
	s.low = sn
}

// handle an incoming packet.
// If in sequence then just output it
// If too new (i.e. a gap exists) cache it.
// If too old just drop it.
func (s *sequencer) handle(pkt *rtpPacket, outf func(pkt *rtpPacket)) {
	sn := pkt.sn

	if !s.lowd {
		if pkt.recovery {
			// Ignore recovery packets if the sequencer has been restarted
			s.sl.inputPacket(pkt, "SEQUENCER RESTART")
			return
		} else {
			s.lowd = true
			s.low = sn
			s.sl.note(" sequencer::handle: Initial seqno=", sn)
		}
	}
	delete(s.retries, sn)
	if s.low == sn {
		s.sl.inputPacket(pkt, "")
		s.sl.outputPacket(pkt)
		outf(pkt)
		s.flushCached(sn+1, outf)
	} else if seqnoDelta(sn, s.low) < 0 {
		s.sl.inputPacket(pkt, "OLD DISCARDED")
	} else {
		s.sl.inputPacket(pkt, "RECOVER")
		s.packets[sn] = pkt
	}
}

// Scan for gaps and send rerequests for these packets. Increment
// the retry counter for each missing packet and check if it matches
// 10, 40 or 90 ms for resends or 150ms for delete.
// When 150ms has been reached for a gap it will just be skipped and
// deleted from the sequence cache
func (s *sequencer) sendReRequests(request chan rerequest) {
	entries := len(s.packets)
	start := seqno(0)
	count := seqno(0)
	retry := 0
	ii := s.low
	for entries > 0 {
		retry = 100000
		count = 0
		for ; entries > 0; ii++ {
			_, ok := s.packets[ii]
			if ok {
				entries--
			} else {
				start = ii
				break
			}
		}
		for ; entries > 0; ii++ {
			_, ok := s.packets[ii]
			if ok {
				entries--
				ii++
				break
			} else {
				s.retries[ii]++
				if retry > s.retries[ii] {
					retry = s.retries[ii]
				}
				count++
			}
		}
		ctl := count > 20000
		if ctl {
			s.printState()
		}
		if count > 0 {
			switch retry {
			case 3, 11, 23: // Send rerequest at 30ms, 110ms, and 230ms
				rr := &rerequest{start, count}
				s.sl.reRequest(rr, retry)
				request <- *rr
			case 37: // Well I don't think we'll get any packets after 370 ms
				s.remove(start, count)
			}
		}
	}
}

func (s *sequencer) printState() {
	prefix := "Sequence ReRequest fail: "
	entries := len(s.packets)
	s.sl.note(prefix, "low=", s.low, ", entries=", entries)
	ia := []int{}

	for ii, _ := range s.packets {
		ia = append(ia, int(ii))
	}
	sort.Ints(ia)
	for _, ii := range ia {
		s.sl.note(prefix, "packet index = ", ii)
	}

	start := seqno(0)
	count := seqno(0)
	retry := 0
	ii := s.low
	for entries > 0 {
		count = 0
		for ; entries > 0; ii++ {
			_, ok := s.packets[ii]
			if ok {
				entries--
			} else {
				start = ii
				break
			}
		}
		for ; entries > 0; ii++ {
			_, ok := s.packets[ii]
			if ok {
				entries--
				ii++
				break
			} else {
				s.retries[ii]++
				if retry > s.retries[ii] {
					retry = s.retries[ii]
				}
				count++
			}
		}
		s.sl.note(prefix, "start=", start, ", count=", count, ", ii=", ii)
	}
}

// Remove all retries and packets starting with start and count entries
// low will be set to the new start
func (s *sequencer) remove(start, count seqno) {
	s.sl.removePackets(start, count)
	for ii := count; ii > 0; ii-- {
		delete(s.packets, start)
		delete(s.retries, start)
		start++
	}
	s.low = start
}

// Start a sequence in a goroutine.
func startSequencer(ref string, data chan *rtpPacket, outf func(pkt *rtpPacket), request chan rerequest) *sequencer {

	s := &sequencer{}
	s.control = make(chan int, 0)
	s.restartSequencer()
	s.ref = ref
	s.sl = &sequencelog{}

	timeout := time.Duration(10 * time.Millisecond) // 10 mS
	timer := time.NewTimer(timeout)

	var cmd int

	go func() {
	normal:
		for {
			// Normal operation
			for !s.inRecovery() {
				if debugSequenceLogFlag != s.sl.traceing {
					s.modifyTrace()
				}
				select {
				case pkt := <-data:
					s.handle(pkt, outf)
				case cmd = <-s.control:
					goto command
				}
			}

			// Recovery
			for s.inRecovery() {
				if debugSequenceLogFlag != s.sl.traceing {
					s.modifyTrace()
				}
				timer.Reset(timeout)
				select {
				case pkt := <-data:
					s.handle(pkt, outf)
				case cmd = <-s.control:
					goto command
				case <-timer.C:
					s.sendReRequests(request)
					s.flushCached(s.low, outf)

				}
			}
			continue normal

		command:
			switch cmd {
			case 0:
				s.sl.note("Restarting Sequencer")
				s.restartSequencer()
			case 1:
				s.sl.note("Shutting down Sequencer")
				return
			}

		}
	}()

	return s
}

func (s *sequencer) modifyTrace() {
	if s.sl == nil || !s.sl.traceing {
		// Open a new trace
		s.sl.initTraceLog(s.ref, "seqnolog", false)
	} else {
		s.sl.closeTraceLog()
	}
}

func debugSequencer(flag bool) {
	debugSequenceLogFlag = flag
}
