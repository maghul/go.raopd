package raopd

import (
	"encoding/binary"
	"fmt"
	"net"
)

const MAX_RTP_PACKET_SIZE = 1500

type rtpHandler func(pkt *rtpPacket)
type rtpTransmitter func(conn *net.UDPConn, client *net.UDPAddr)
type rtpFactory func() (rtpHandler, rtpTransmitter, string)

type rtpPacket struct {
	seqno   uint16
	content []byte
	buf     []byte
}

func makeRtpPacket() *rtpPacket {
	return &rtpPacket{0, nil, make([]byte, MAX_RTP_PACKET_SIZE)}
}

func (pkt *rtpPacket) payloadType() uint8 {
	return pkt.content[1] & 0x7f
}

func (pkt *rtpPacket) Reclaim() {
	// NYI: TODO: This is intended to be used for pooling rtp packets
	// instead of recreating them.
}

type rtp net.UDPConn

func (r *rtp) Port() int {
	claddr := r.LocalAddr()
	culaddr := claddr.(*net.UDPAddr)
	return culaddr.Port
}

func (r *rtp) Teardown() {
	r.Close()
}

func (r *Raop) getDataHandler() (rtpHandler, rtpTransmitter, string) {
	return func(pkt *rtpPacket) {
		if pkt.payloadType() == 96 {
			r.seqchan <- pkt
		} else {
			fmt.Println("DATA CHANNEL: unknown payload type", pkt.payloadType())
			pkt.Reclaim()
		}
	}, nil, "DATA"
}

func (r *Raop) getControlHandler() (rtpHandler, rtpTransmitter, string) {
	rx := func(pkt *rtpPacket) {
		switch pkt.payloadType() {
		case 84:
			// Not doing time sync yet...
			// binary.BigEndian.Uint64(pkt.content[8:16])   = NTP Time
			// binary.BigEndian.Uint32(pkt.content[16:20])  = RTP Timestamp for next packet
			pkt.Reclaim()

		case 85:
			fmt.Println("CONTROL CHANNEL: A retransmit request should not be received here")
			pkt.Reclaim()

		case 86:
			pkt.content = pkt.content[4:]
			pkt.seqno = binary.BigEndian.Uint16(pkt.content[2:4])
			r.seqchan <- pkt

		default:
			fmt.Println("CONTROL CHANNEL: unknown payload type", pkt.payloadType())
			pkt.Reclaim()
		}
	}
	tx := func(conn *net.UDPConn, client *net.UDPAddr) {
		buf := make([]byte, 32)
		seqno := uint16(1)
		//		timestamp := uint32(1)

		for {
			select {
			case rr := <-r.rrchan:
				fmt.Println("ReRequest:", rr)
				buf[0] = 0x80
				buf[1] = 85 + 0x80
				binary.BigEndian.PutUint16(buf[2:4], seqno)
				binary.BigEndian.PutUint16(buf[4:6], rr.first)
				binary.BigEndian.PutUint16(buf[6:8], rr.count)
				conn.WriteToUDP(buf[0:8], client)
			}
			seqno++
		}
	}
	return rx, tx, "CONTROL"
}

func (r *Raop) getTimingHandler() (rtpHandler, rtpTransmitter, string) {
	return func(pkt *rtpPacket) {
		// Ignoring any incoming packets.
		pkt.Reclaim()
	}, nil, "TIMING"
}

func startRtp(f rtpFactory) *rtp {

	caddr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		panic("Could not resolve UDP address ':0'")
	}
	conn, err := net.ListenUDP("udp", caddr)
	if err != nil {
		panic("Could not listen to UDP address ':0'")
	}

	claddr := conn.LocalAddr()

	handler, tx, name := f()
	fmt.Println("Starting RTP server ", name, "at", claddr)
	if handler != nil {
		go func() {
			for {
				pkt := makeRtpPacket()
				n, addr, err := conn.ReadFromUDP(pkt.buf)
				if tx != nil {
					go tx(conn, addr)
					tx = nil
				}
				if err != nil {
					panic(err)
				}
				pkt.content = pkt.buf[0:n]
				pkt.seqno = binary.BigEndian.Uint16(pkt.content[2:4])
				handler(pkt)

			}
		}()
	}
	return (*rtp)(conn)
}
