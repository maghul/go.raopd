package raopd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

const max_rtp_packet_size = 1800

type rtpHandler func(pkt *rtpPacket)
type rtpTransmitter func(conn *net.UDPConn, addrchan chan *net.UDPAddr)
type rtpFactory func() (rtpHandler, rtpTransmitter, string)

type rtpPacket struct {
	seqno   uint16
	content []byte
	buf     []byte
}

func makeRtpPacket() *rtpPacket {
	return &rtpPacket{0, nil, make([]byte, max_rtp_packet_size)}
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

func (r *raop) getDataHandler() (rtpHandler, rtpTransmitter, string) {
	return func(pkt *rtpPacket) {
		if pkt.payloadType() == 96 {
			r.seqchan <- pkt
		} else {
			fmt.Println("DATA CHANNEL: unknown payload type", pkt.payloadType())
			pkt.Reclaim()
		}
	}, nil, "DATA"
}

func (r *raop) getControlHandler() (rtpHandler, rtpTransmitter, string) {
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
			fmt.Println("CONTROL CHANNEL: Recovery Packet, seqno=", pkt.seqno)
			r.seqchan <- pkt

		default:
			fmt.Println("CONTROL CHANNEL: unknown payload type", pkt.payloadType())
			pkt.Reclaim()
		}
	}
	tx := func(conn *net.UDPConn, addrchan chan *net.UDPAddr) {
		buf := make([]byte, 32)
		seqno := uint16(1)
		//		timestamp := uint32(1)
		client := <-addrchan
		fmt.Println("CONTROL CHANNEL CLIENT:", client)

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
				fmt.Println("CONTROL CHANNEL: Recovery Request:", rr, "sent to", client)
			case client = <-addrchan:
				fmt.Println("CONTROL CHANNEL: client=", client)
			}
			seqno++
		}
	}
	return rx, tx, "CONTROL"
}

func (r *raop) getTimingHandler() (rtpHandler, rtpTransmitter, string) {
	return func(pkt *rtpPacket) {
		// Ignoring any incoming packets.
		pkt.Reclaim()
	}, nil, "TIMING"
}

func sameUDPAddr(a, b *net.UDPAddr) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	} else {
		return bytes.Compare(a.IP, b.IP) == 0 && a.Port == b.Port
	}
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
	var addrchan chan *net.UDPAddr
	if tx != nil {
		addrchan = make(chan *net.UDPAddr)
	}
	if handler != nil {
		go func() {
			defer func() { conn.Close() }()
			paddr := (*net.UDPAddr)(nil)
			for {
				pkt := makeRtpPacket()
				n, addr, err := conn.ReadFromUDP(pkt.buf)
				if addrchan != nil && !sameUDPAddr(addr, paddr) {
					addrchan <- addr
					paddr = addr
				}
				if err != nil {
					fmt.Println("Panic err=", err)
					return // Exit RTP server
				}
				pkt.content = pkt.buf[0:n]
				pkt.seqno = binary.BigEndian.Uint16(pkt.content[2:4])
				handler(pkt)

			}
		}()
	}
	if tx != nil {
		go tx(conn, addrchan)
		tx = nil
	}
	return (*rtp)(conn)
}
