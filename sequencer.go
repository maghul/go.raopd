package raopd

import (
	"emh/logger"
	"fmt"
	"os"
	"time"
)

var seqlog = logger.GetLogger("raopd.sequencer")

const sequencerDebug = false

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

			for ii := next; entries > 0; ii++ {
				pkt, ok := pkts[ii]
				if sequencerDebug {
					state := "GAP"
					if ok {
						if pkt == nil {
							state = "REQUESTED"
						} else {
							state = "OK"
						}
					}
					seqlog.Debug.Println("gap ", ii, ", state=", state)
				}
				if ok {
					if gap {
						seqlog.Debug.Println("TX: REREQUEST", start, ", ", count)
						if sequencerDebug {
							fmt.Println("TX: REREQUEST", start, ", ", count)
						}
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

		flush := func() {
			seqlog.Debug.Println("SEQUENCER: Flushing Sequencer")
			// Empty everything we have stored
			for _, pkt := range pkts {
				if pkt != nil {
					pkt.Reclaim()
				}
			}
			pkts = make(map[uint16]*rtpPacket)

			// Empty the input channels
		flushloop:
			for {
				select {
				case pkt := <-data:
					pkt.Reclaim()
				default:
					break flushloop
				}
			}

			initial = true
			seqlog.Debug.Println("SEQUENCER: Flushed Sequencer")
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

			if initial {
				if sequencerDebug {
					seqlog.Debug.Println("SEQUENCER: Waiting for data to start")
				}
			}
			select {
			case pkt := <-data:
				seqno := pkt.seqno
				delta := seqnoDelta(seqno, next)

				switch {
				case initial:
					initial = false
					fallthrough

				case delta == 0:
					next = pkt.seqno + 1
					transmit(pkt)

				case delta > 2000:
					fmt.Fprintln(os.Stderr, "SEQUENCER: Delta too large, ", delta, " flushing sequencer")
					flush()

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
				sendReRequests()
				current = 0 // Force rearming the timer

			case c = <-s.seq: // Flush
				flush()
			}

		}
	}()
	return s
}
