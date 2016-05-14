package raopd

import (
	"fmt"
	"os"
	"time"
)

func emptyDebug(v ...interface{}) {
}

var debug = emptyDebug

type sequencer struct {
	seq chan int
}

type rerequest struct {
	first, count uint16
}

func (s sequencer) flush() {
	s.seq <- 0
}

func (s sequencer) stop() {
	s.seq <- 1
}

func (s sequencer) close() {
	s.seq <- 1
}

func seqnoDelta(a, b uint16) int {
	switch {
	case a == b:
		return 0
	case a > b:
		return int(a - b)
	case (a < 8192 && b > 65536-8192):
		return 65536 - int(b-a)
	case a < b:
		return -int(b - a)
	}
	panic(fmt.Sprint("seqnoDelta, no case for a=", a, ", b=", b))
}

// This will take unordered data on the input channel and order
// it to the output channel. It will also send rerequests on the
// request channel if there are gaps in the indata
func startSequencer(data chan *rtpPacket, outf func(pkt *rtpPacket), request chan rerequest) sequencer {
	s := sequencer{make(chan int)}

	go func() {
		pkts := make(map[uint16]*rtpPacket)

		long := time.Duration(365 * 24 * time.Hour)
		short := time.Duration(10 * time.Millisecond) // 10 mS
		current := long
		timer := time.NewTimer(current)

		c := 0

		transmit := func(pkt *rtpPacket) {
			debug("TX: ", pkt.seqno)
			delete(pkts, pkt.seqno)
			outf(pkt)
		}

		initial := true
		next := uint16(0)

		sendReRequests := func() {
			entries := len(pkts)
			gap := false
			start := uint16(0)
			count := uint16(0)
			debug("REREQ: next=", next, ", entries=", entries)
			for ii := next; entries > 0; ii++ {
				pkt, ok := pkts[ii]
				debug("REREQ: ii=", ii, " --> ok=", ok, ", ", pkt != nil)
				if ok {
					if gap {
						debug("TX:", start, ", ", count)
						request <- rerequest{start, count}
						gap = false
					}
					entries--
				} else {
					pkts[ii] = nil // Mark as being in recovery
					if !gap {
						gap = true
						start = ii
						count = 0
					}
				}
				count++

			}
		}

		defer func() {
			if c == 0 {
				panic("SEQUENCER: unexpected loop exit!")
			}
		}()
		for {
			// Reset the timer to short (10ms) if the state has changed
			newTime := long
			if len(pkts) > 0 {
				newTime = short
			}
			if current != newTime {
				current = newTime
				timer.Reset(current)
				debug("Rearming timer ", timer, " for ", current)
			}

			// Dump all packets possible from next onwards
			for {
				pkt, ok := pkts[next]
				if !ok || pkt == nil {
					break
				}
				transmit(pkt)
				next++
			}

			select {
			case pkt := <-data:
				seqno := pkt.seqno
				delta := seqnoDelta(seqno, next)

				debug("RX:", seqno)
				switch {
				case initial:
					initial = false
					fallthrough

				case delta == 0:
					next = pkt.seqno + 1
					transmit(pkt)

				case delta > 200:
					fmt.Fprintln(os.Stderr, "SEQUENCER: Delta too large, ", delta, " flushing sequencer")
					s.flush()

				case delta > 0:
					pkts[seqno] = pkt
					for ii := next; ii < seqno; ii++ {
						p, ok := pkts[ii]
						if ok && p == nil { // Supposedly being recovered but not yet received, mark for recovery again
							delete(pkts, ii)
						}
					}

				default:
					pkt.Reclaim()
				}

			case <-timer.C:
				debug("TIMER TRIGGERED:", timer)
				sendReRequests()
				current = 0 // Force rearming the timer

			case c = <-s.seq: // Flush
				debug("SEQUENCER: Flushing Sequencer")
				for _, pkt := range pkts {
					if pkt != nil {
						pkt.Reclaim()
					}
				}
				pkts = make(map[uint16]*rtpPacket)
				initial = true
				if c == 1 {
					return
				}
			}

		}
	}()
	return s
}
