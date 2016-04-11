package raopd

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

const max_rtp_packet_size = 1800

type rtpHandler func(pkt *rtpPacket)
type rtpTransmitter func(conn *net.UDPConn)
type rtpFactory func() (rtpHandler, rtpTransmitter, string)

type rtpPacket struct {
	seqno   uint16
	content []byte
	buf     []byte
}

var rtpPacketPoolCounter = 0
var rtpPacketPool = &sync.Pool{New: func() interface{} {
	rtpPacketPoolCounter++
	fmt.Println("RTP PACKET POOL: Created ", rtpPacketPoolCounter, " RTP packets in pool")
	return &rtpPacket{0, nil, make([]byte, max_rtp_packet_size)}
}}

func makeRtpPacket() *rtpPacket {
	return rtpPacketPool.Get().(*rtpPacket)
}

func (pkt *rtpPacket) Reclaim() {
	rtpPacketPool.Put(pkt)
}

func (pkt *rtpPacket) payloadType() uint8 {
	return pkt.content[1] & 0x7f
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
	tx := func(conn *net.UDPConn) {
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

				conn.Write(buf[0:8])
				fmt.Println("CONTROL CHANNEL: Recovery Request:", rr, "sent to", conn.RemoteAddr())
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

func startRtp(f rtpFactory, raddr *net.UDPAddr) *rtp {
	/*
	   	as := fmt.Sprintf( ":%d", randomPort )
	   	randomPort++

	   	caddr, err := net.ResolveUDPAddr("udp", as)
	   	if err != nil {
	   		panic("Could not resolve UDP address ':0'")
	   	}

	   	conn, err := net.ListenUDP("udp", caddr)
	   	if err != nil {
	   		panic(fmt.Sprintf("Could not listenl UDP local=%v: %v", caddr, err))
	   	}
	   //	claddr := conn.LocalAddr()

	   	laddr, err := net.ResolveUDPAddr("udp",":48123")
	   	if err != nil {
	   		panic(fmt.Sprintf("Could not dial UDP  remote=%v: %v", raddr, err))
	   	}
	*/

	var conn *net.UDPConn
	var err error

	if raddr == nil {
		fmt.Println("LISTENING")
		conn, err = net.ListenUDP("udp", nil)
	} else {
		fmt.Println("DIALING raddr=", raddr)
		conn, err = net.DialUDP("udp", nil, raddr)
	}
	if err != nil {
		panic(fmt.Sprintf("Could not dial UDP  remote=%v: %v", raddr, err))
	}

	handler, tx, name := f()
	fmt.Println("Starting RTP server ", name, "at conn l:", conn.LocalAddr(), ", r:", conn.RemoteAddr())
	/*
		var addrchan chan *net.UDPAddr
		if tx != nil {
			addrchan = make(chan *net.UDPAddr)
		}
	*/
	if handler != nil {
		go func() {
			defer func() { conn.Close() }()
			for {
				pkt := makeRtpPacket()
				var n int
				var err error
				n, err = conn.Read(pkt.buf)
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
		go tx(conn)
		tx = nil
	}
	return (*rtp)(conn)
}
