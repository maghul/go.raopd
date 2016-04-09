package raopd

import (
	"fmt"
)

type sequencer chan bool

type rerequest struct {
	first, count uint16
}

func (s sequencer) flush() {
	fmt.Println("SEQUENCE: flush request")
	s <- true
	fmt.Println("SEQUENCE: flush requested")
}

func sequencerDiagnostics(pkts map[uint16]*rtpPacket, recovery map[uint16]bool, start, end uint16) {
	s := "ok"
	b := start
	fmt.Println("\nSEQUENCER: FAILURE: start=", start, ", end=", end)
	for ii := start; ii != end+1; ii++ {
		pkt, _ := pkts[ii]
		rec, _ := recovery[ii]

		ns := "ok"
		if pkt == nil {
			if rec {
				ns = "in recovery"
			} else {
				ns = "unrecovered"
			}
		}
		if ns != s {
			fmt.Println("SEQUENCER: FAILURE: ", b, "...", ii, " state=", s)
			s = ns
			b = ii
		}
	}
	fmt.Println("SEQUENCER: FAILURE: ", b, "...", end, " state=", s)
}

func markUnrecoveredPackets(recovery map[uint16]int, pkts map[uint16]*rtpPacket, from,to uint16, defcon int) {
	for ; from<to; from++ {
		_, ok := pkts[from]
		//if ok {
		//	break
		//}
		level, _ := recovery[from]
		if !ok && level<defcon {
			fmt.Println( "MARKING PACKET ", from, " FOR REREQUEST AT DEFCON=", defcon,"!")
			recovery[from] = 0   // Don't delete here since it will be set immediately 
		}
	}
}

// This will take unordered data on the input channel and order
// it to the output channel. It will also send rerequests on the
// request channel if there are gaps in the indata
func startSequencer(data chan *rtpPacket, out func(pkt *rtpPacket), request chan rerequest) sequencer {
	s := sequencer(make(chan bool))

	go func() {
		pkts := make(map[uint16]*rtpPacket)
		recovery := make(map[uint16]int)
		lgp := uint16(0)
		recover := false
		defcon := 0 // The current maximum recovery level.

		pkt := <-data
		fmt.Println( "INITIAL", pkt.seqno)
		next := pkt.seqno + 1
		out(pkt)

		defer func() { panic("SEQUENCER: loop exit!") }()
		for {
			recover = false
		readloop:
			for {
				// Dump everything stored in the cache we can. That is
				// Everything from next until the first missing packet.
				for {
					pkt, ok := pkts[next]
					if !ok {
						break
					}
					delete(pkts, next)

					//					fmt.Println( "SEQUENCER:OUT: pkt=", pkt.seqno )
					out(pkt)
					next++
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

					case <-s: // Flush
						fmt.Println("SEQUENCER: Flushing Sequencer")
						pkts = make(map[uint16]*rtpPacket)
						recovery = make(map[uint16]int)
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
						fmt.Println("SEQUENCER: Restarting")
					}
				}

				span := int32(pkt.seqno) - int32(next)
				if span < -50000 {
					span += 0x10000
				}
				if span > 250 {
					fmt.Println("Large span!")
//					sequencerDiagnostics(pkts, recovery, next, pkt.seqno)
//					markUnrecoveredPackets(recovery, pkts, next, pkt.seqno)
				}

				//				fmt.Println( "SEQUENCER:IN: pkt=", pkt.seqno )

				switch {
				case pkt.seqno == next: // Packet in sequence: just output it
					fmt.Println("IN SYNC ", pkt.seqno)
					delete(recovery,next)
					//					fmt.Println( "SEQUENCER:OUT: pkt=", pkt.seqno )
					out(pkt)
					next++
					if (next==lgp) {
						if (defcon>0) {
							fmt.Println( "In sync again!" )
						}
						defcon = 0
					}
				case pkt.seqno > next: // Packet out of sequence. Stora and flag for recovery
					fmt.Println("OUT OF SYNC ", pkt.seqno)
					seqno := pkt.seqno
					pkts[pkt.seqno] = pkt
					if (seqno < 0x8000 && lgp > 0x8000) || seqno > lgp {
						// Check if seqno>lgp also handles if it has wrapped.
						// seqno has wrapped, top has not. i.e seqno is greater
						// lgp is the latest.
						lgp = seqno
						delete(recovery, pkt.seqno)
					} else {
						// If we get a packet less than top assume that we won't get
						// any packages prior to this package. If we are missing any
						// such packets then we can assume that any recovery sent has
						// failed and we need to request these packets again.
						// Only do the first gap since we will get here again if there
						// are more gaps later.
						// Set the defcon level to the recovered packet +1 
						defcon = recovery[seqno]+1
						delete(recovery, pkt.seqno)

						fmt.Println( "DEFCON=", defcon )
						markUnrecoveredPackets(recovery, pkts, next, seqno, defcon)
					}
					recover = true
				}
			}

			// Check gaps: Any missing packet not flagged as being in recovery should be
			// rerequested. These will be collected into blocks of rerequest for all consecutive
			// missing unrecovered packets.
			first := -1
			count := uint16(0)
			fmt.Println( "CHECK GAPS ", next, "...", lgp )
			for ii := next; ii != lgp+1; ii++ {
				_, ok := pkts[ii]
				rec, _ := recovery[ii]

				fmt.Println( ii," is ok=", ok, " recovery=", rec )
				ok = ok || rec>0 // Ignore if not OK when its in recovery
				if first < 0 {
					if !ok {
						first = int(ii) // Start of a new request
						count = 0
						recovery[ii] = defcon
					}
				} else {
					if ok {
						fmt.Println("REREQUEST: first=", first, ", count=", count)
						request <- rerequest{uint16(first), count}
						first = -1
					} else {
						recovery[ii] = defcon
					}
				}
				count++
			}
		}
	}()
	return s
}
