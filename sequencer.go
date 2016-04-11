package raopd

import (
	"fmt"
	"time"
)

const debugSequencer = false

type sequencer struct {
	seq chan bool
	req chan bool
}

type rerequest struct {
	first, count uint16
}

func (s sequencer) flush() {
	fmt.Println("SEQUENCE: flush request")
	s.seq <- true
	s.req <- true
	fmt.Println("SEQUENCE: flush requested")
}

// This will take unordered data on the input channel and order
// it to the output channel. It will also send rerequests on the
// request channel if there are gaps in the indata
func startSequencer(data chan *rtpPacket, out func(pkt *rtpPacket), request chan rerequest) sequencer {
	s := sequencer{make(chan bool), make(chan bool)}

	gap := make(chan uint16)
	nextSeqno := make(chan uint16)

	lgp := uint16(0) // TODO: We read this in the recovery thread, may have to be atomic
	go func() {
		pkts := make(map[uint16]*rtpPacket)

		pkt := <-data
		fmt.Println("INITIAL", pkt.seqno)
		next := pkt.seqno + 1
		nextSeqno <- next
		out(pkt)

		defer func() { panic("SEQUENCER: loop exit!") }()
		for {
		readloop:
			for {
				// Dump everything stored in the cache we can. That is
				// Everything from next until the first missing packet.
				recover := len(pkts) > 0
				if recover {
					for {
						pkt, ok := pkts[next]
						if !ok {
							break
						}
						delete(pkts, next)

						//					fmt.Println( "SEQUENCER:OUT: pkt=", pkt.seqno )
						next++
						nextSeqno <- next
						out(pkt)
					}
				}

				// Read a packet. Block for incoming packets in normal mode. If recover
				// is set then we will empty the data queue and exit to recovery (exit the readloop)
				if recover {
					select {
					case pkt = <-data:
					default:
						break readloop
					}
				} else {
					select {
					case pkt = <-data:

					case <-s.seq: // Flush
						fmt.Println("SEQUENCER: Flushing Sequencer")
						pkts = make(map[uint16]*rtpPacket)
						lgp = uint16(0)
						recover = false

					flushPackets:
						for {
							select {
							case pkt = <-data:
								fmt.Println("SEQUENCER: flushing incoming packet", pkt.seqno)
							default:
								break flushPackets
							}

						}
						fmt.Println("SEQUENCER: Waiting for restart")
						pkt = <-data
						next = pkt.seqno + 1
						out(pkt)
						nextSeqno <- next
						fmt.Println("SEQUENCER: Restarting")
					}
				}

				switch {
				case pkt.seqno == next: // Packet in sequence: just output it
					if debugSequencer {
						fmt.Println("IN SYNC ", pkt.seqno)
					}
					next = pkt.seqno + 1
					nextSeqno <- next
					out(pkt)

				case pkt.seqno > next: // Packet out of sequence. Stora and flag for recovery
					if debugSequencer {
						fmt.Println("OUT OF SYNC ", pkt.seqno)
					}
					seqno := pkt.seqno
					pkts[pkt.seqno] = pkt
					gap <- seqno
					if (seqno < 0x8000 && lgp > 0x8000) || seqno > lgp {
						// Check if seqno>lgp also handles if it has wrapped.
						// seqno has wrapped, top has not. i.e seqno is greater
						// lgp is the latest.
						lgp = seqno
					}

				}
			}
		}
	}()
	// Recovery transmit process
	go func() {
		recovery := make(map[uint16]int)
		// 0 - needs recovery
		// 1 - in recovery
		// 2 - recovered

		next := uint16(0)
		long := time.Duration(365 * 24 * time.Hour)
		short := time.Duration(10 * time.Millisecond) // 10 mS
		timer := time.NewTimer(long)

		for {
		gather:
			for {
				if len(recovery) > 0 {
					timer.Reset(short)
				} else {
					timer.Reset(long)
				}
				select {
				case gapseqno := <-gap:
					if debugSequencer {
						fmt.Println("RECOVERY: recovered ", gapseqno)
					}
					recovery[gapseqno] = 2 // Flag as recovered

					// TODO: This should probably not be done on every packet out of sync
					if debugSequencer {
						fmt.Println("REMARKING FROM ", next, " to ", gapseqno)
					}
					for ii := next; ii != gapseqno; ii++ {
						needsRecovery, ok := recovery[ii]
						if !ok || needsRecovery == 1 {
							if debugSequencer {
								fmt.Println("RECOVERY: FLAG ", ii, " for recovery.")
							}
							recovery[ii] = 0 // Should be recovered.
						} else {
							if debugSequencer {
								fmt.Println("RECOVERY: in recovery ", ii, " needsRecovery=", needsRecovery)
							}

						}
					}

					//				case recseqno := <- recovered:
					//					recovery[recseqno]  // Flag as recovered

				case next = <-nextSeqno:
					if debugSequencer {
						fmt.Println("RECOVERY: bottom ", next)
					}
					delete(recovery, next-1) // No need to check this anymore

				case <-s.req: // Flush
					recovery = make(map[uint16]int)
					next = 0

				case <-timer.C:
					break gather
				}

			}

			// Inactivity, check gaps
			first := -1
			count := uint16(0)
			if debugSequencer {
				fmt.Println("RECOVERY:", "CHECK GAPS ", next, "...", lgp)
			}
			for ii := next; ii != lgp+1; ii++ {
				needsRecovery, _ := recovery[ii]

				if debugSequencer {
					fmt.Println("RECOVERY:", ii, " needs recovery=", needsRecovery)
				}
				if first < 0 {
					if needsRecovery == 0 {
						first = int(ii) // Start of a new request
						count = 0
						recovery[ii] = 1
					}
				} else {
					if needsRecovery == 0 {
						recovery[ii] = 1
					} else {
						if debugSequencer {
							fmt.Println("RECOVERY:", "REREQUEST: first=", first, ", count=", count)
						}
						request <- rerequest{uint16(first), count}
						first = -1
					}
				}
				count++
			}
		}
	}()
	return s
}
