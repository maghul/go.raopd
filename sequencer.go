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

// This will take unordered data on the input channel and order
// it to the output channel. It will also send rerequests on the
// request channel if there are gaps in the indata
func startSequencer(data chan *rtpPacket, out func(pkt *rtpPacket), request chan rerequest) sequencer {
	s := sequencer(make(chan bool))

	go func() {
		pkts := make(map[uint16]*rtpPacket)
		recovery := make(map[uint16]bool)
		lgp := uint16(0)
		recover := false

		pkt := <-data
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
						recovery = make(map[uint16]bool)
						lgp = uint16(0)
						recover = false

					flushPackets:
						for {
							select {
							case pkt = <-data:
								fmt.Println("SEQUENCER: flushing incoming packet")
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
					sequencerDiagnostics(pkts, recovery, next, pkt.seqno)
				}

				//				fmt.Println( "SEQUENCER:IN: pkt=", pkt.seqno )
				// Any packet received will not have to be recovered.
				delete(recovery, pkt.seqno)

				switch {
				case pkt.seqno == next: // Packet in sequence: just output it
					recovery[next] = false
					//					fmt.Println( "SEQUENCER:OUT: pkt=", pkt.seqno )
					out(pkt)
					next++
				case pkt.seqno > next: // Packet out of sequence. Stora and flag for recovery
					seqno := pkt.seqno
					pkts[pkt.seqno] = pkt
					if (seqno < 0x8000 && lgp > 0x8000) || seqno > lgp {
						// seqno has wrapped, top has not. i.e seqno is greater
						lgp = seqno
					}
					// If we get a packet assume that we won't get any packages
					// prior to this package. If we are missing any such package
					// then we can assume that any recovery sent has failed and
					// we need to send it again.
					for {
						if seqno == next {
							break
						}
						seqno--
						_, ok := pkts[seqno]

						if ok {
							break
						}
						recovery[seqno] = false

					}
					recover = true
				}
			}

			// Check gaps: Any missing packet not flagged as being in recovery should be
			// rerequested. These will be collected into blocks of rerequest for all consecutive
			// missing unrecovered packets.
			first := -1
			count := uint16(0)
			for ii := next; ii != lgp+1; ii++ {
				_, ok := pkts[ii]
				rec, _ := recovery[ii]

				ok = ok || rec // Ignore if not OK when its in recovery

				if first < 0 {
					if !ok {
						first = int(ii) // Start of a new request
						count = 0
						recovery[ii] = true
					}
				} else {
					if ok {
						fmt.Println("REREQUEST: first=", first, ", count=", count)
						request <- rerequest{uint16(first), count}
						first = -1
					} else {
						recovery[ii] = true
					}
				}
				count++
			}
		}
	}()
	return s
}
