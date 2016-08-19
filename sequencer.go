package raopd

import (
	"emh/logger"
	"fmt"
	//	"os"
	"time"
)

var seqlog = logger.GetLogger("raopd.sequencer")

type sequencer struct {
	// Control channel
	control chan int
	ref     string

	// Internally used
	low     seqno
	lowd    bool
	retries map[seqno]int
	packets map[seqno]*rtpPacket
}

type rerequest struct {
	first, count seqno
}

func (rr *rerequest) String() string {
	return fmt.Sprintf("ReRequest{first=%d, count=%d}", rr.first, rr.count)
}

// Restart the sequencer. Empty all internal caches
func (s sequencer) flush() {
	s.control <- 0
}

// Close the sequencer completely.
func (s sequencer) close() {
	s.control <- 1
}

// Internal functions

// Sequencer is in recovery mode.
func (m *sequencer) inRecovery() bool {
	return len(m.packets) > 0
}

// Internal function to clear all internal caches and
// start over. Called by sequence go-routine when flush has been called.
func (m *sequencer) restartSequencer() {
	m.lowd = false
	m.low = 0
	m.retries = make(map[seqno]int)
	m.packets = make(map[seqno]*rtpPacket)
}

// flush packet cache from seqno and onwards and set low to
// first gap in the cache.
func (m *sequencer) flushCached(sn seqno, outf func(pkt *rtpPacket)) {
	sn--
	for {
		sn++
		pkt, ok := m.packets[sn]
		//seqlog.Debug.Println("sequencer::handle: FLUSH seqno=", seqno, " exists=", ok)
		if ok {
			m.logOut(sn)
			outf(pkt)
			delete(m.packets, sn)
		} else {
			break
		}
	}
	m.low = sn
}

func (m *sequencer) logOut(sn seqno) {
	if sn%1000 == 0 {
		seqlog.Debug.Println(m.ref, " sequencer::logOut: output sequence number=", sn)
	}
}

// handle an incoming packet.
// If in sequence then just output it
// If too new (i.e. a gap exists) cache it.
// If too old just drop it.
func (m *sequencer) handle(pkt *rtpPacket, outf func(pkt *rtpPacket)) {
	sn := pkt.sn

	if !m.lowd {
		if pkt.recovery {
			// Ignore recovery packets if the sequencer has been restarted
			return
		} else {
			m.lowd = true
			m.low = sn
			seqlog.Debug.Println(m.ref, " sequencer::handle: Initial seqno=", sn)
		}
	}
	if m.retries[sn] > 0 {
		seqlog.Debug.Println(m.ref, " sequencer::handle: Filled gap", sn)
	}
	delete(m.retries, sn)
	if m.low == sn {
		m.logOut(sn)
		outf(pkt)
		m.flushCached(sn+1, outf)
	} else if seqnoDelta(sn, m.low) < 0 {
		seqlog.Debug.Println(m.ref, " sequencer::handle: too old packet", sn)
	} else {
		if !m.inRecovery() {
			seqlog.Debug.Println(m.ref, " sequencer::handle: starting recovery on packet=", sn, ", low=", m.low)
		}
		m.packets[sn] = pkt
	}
}

// Scan for gaps and send rerequests for these packets. Increment
// the retry counter for each missing packet and check if it matches
// 10, 40 or 90 ms for resends or 150ms for delete.
// When 150ms has been reached for a gap it will just be skipped and
// deleted from the sequence cache
func (m *sequencer) sendReRequests(request chan rerequest) {
	entries := len(m.packets)
	start := seqno(0)
	count := seqno(0)
	retry := 0
	ii := m.low
	for entries > 0 {
		retry = 100000
		count = 0
		for ; entries > 0; ii++ {
			_, ok := m.packets[ii]
			if ok {
				entries--
			} else {
				start = ii
				break
			}
		}
		for ; entries > 0; ii++ {
			_, ok := m.packets[ii]
			if ok {
				entries--
				ii++
				break
			} else {
				m.retries[ii]++
				if retry > m.retries[ii] {
					retry = m.retries[ii]
				}
				count++
			}
		}
		if count > 0 {
			switch retry {
			case 1, 4, 9: // Send rerequest at 10ms, 40ms, and 90ms
				rr := &rerequest{start, count}
				seqlog.Debug.Println(m.ref, " sequencer::sendReRequest: rerequest=", rr)
				request <- *rr
			case 15: // Well I don't think we'll get any packets after 150 ms
				m.remove(start, count)
			}
		}
	}
}

// Remove all retries and packets starting with start and count entries
// low will be set to the new start
func (m *sequencer) remove(start, count seqno) {
	for ii := count; ii > 0; ii-- {
		delete(m.packets, start)
		delete(m.retries, start)
		start++
	}
	m.low = start
}

// Start a sequence in a goroutine.
func startSequencer(ref string, data chan *rtpPacket, outf func(pkt *rtpPacket), request chan rerequest) *sequencer {

	m := &sequencer{}
	m.control = make(chan int, 0)
	m.restartSequencer()
	m.ref = ref

	timeout := time.Duration(10 * time.Millisecond) // 10 mS
	timer := time.NewTimer(timeout)

	var cmd int

	go func() {
	normal:
		for {
			// Normal operation
			for !m.inRecovery() {
				select {
				case pkt := <-data:
					m.handle(pkt, outf)
				case cmd = <-m.control:
					goto command
				}
			}

			// Recovery
			for m.inRecovery() {
				timer.Reset(timeout)
				select {
				case pkt := <-data:
					m.handle(pkt, outf)
				case cmd = <-m.control:
					goto command
				case <-timer.C:
					m.sendReRequests(request)
					m.flushCached(m.low, outf)

				}
			}
			continue normal

		command:
			switch cmd {
			case 0:
				seqlog.Debug.Println(m.ref, " Restarting Sequencer")
				m.restartSequencer()
			case 1:
				seqlog.Debug.Println(m.ref, " Shutting down Sequencer")
				return
			}

		}
	}()

	return m
}
