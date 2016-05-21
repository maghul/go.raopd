package raopd

import (
	"bytes"
	"emh/logger"
	"encoding/binary"
	"fmt"
	"net"
)

var rtplog = logger.GetLogger("raopd.rtp")

const max_rtp_packet_size = 1800

type rtpHandler func(pkt *rtpPacket)
type rtpTransmitter func(conn *net.UDPConn)
type rtpFactory func(raddr *net.UDPAddr) (rtpHandler, rtpTransmitter, string)

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

func (r *raop) getDataHandler(raddr *net.UDPAddr) (rtpHandler, rtpTransmitter, string) {
	prefix := fmt.Sprint("DATA:", raddr, ": ")
	return func(pkt *rtpPacket) {
		if pkt.payloadType() == 96 {
			r.seqchan <- pkt
		} else {
			rtplog.Debug.Println(prefix, " unknown payload type ", pkt.payloadType())
			pkt.Reclaim()
		}
	}, nil, "DATA"
}

func (r *raop) getControlHandler(raddr *net.UDPAddr) (rtpHandler, rtpTransmitter, string) {
	prefix := fmt.Sprint("CONTROL:", raddr, ": ")
	rx := func(pkt *rtpPacket) {
		switch pkt.payloadType() {
		case 84:
			// Not doing time sync yet...
			// binary.BigEndian.Uint64(pkt.content[8:16])   = NTP Time
			// binary.BigEndian.Uint32(pkt.content[16:20])  = RTP Timestamp for next packet
			pkt.Reclaim()

		case 85:
			rtplog.Debug.Println(prefix, "A retransmit request should not be received here")
			pkt.Reclaim()

		case 86:
			base := pkt.content
			status := pkt.seqno // Seqno is actuall some kind of status.
			if status == 1 {
				// It seems that status==1 means that the retransmission won't happen
				// We could keep track of the rerequests and zap them but it is easier to
				// flush the sequencer.
				failedSeqno := binary.BigEndian.Uint16(pkt.content[4:6])
				zero := binary.BigEndian.Uint16(pkt.content[6:8])
				rtplog.Debug.Println(prefix, "NO Resend for ", failedSeqno)
				if zero != 0 {
					rtplog.Info.Println(prefix, "Resend fail assertion error, zero=", zero)
				}
				pkt.Reclaim()
				r.sequencer.flush()
			} else {
				pkt.content = pkt.content[4:]
				pkt.seqno = binary.BigEndian.Uint16(pkt.content[2:4])
				rtplog.Debug.Printf("%sRecovery Packet, status=%d, seqno=%d", prefix, status, pkt.seqno)
				if base[4] != 0x80 && base[5] != 0x60 {
					l := len(base)
					if l > 20 {
						l = 20
					}
				}
				r.seqchan <- pkt
			}

		default:
			rtplog.Debug.Println(prefix, "Unknown payload type ", pkt.payloadType())
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
				rtplog.Debug.Println(prefix, "ReRequest:", seqno, ", rr=", rr)
				buf[0] = 0x80
				buf[1] = 85 + 0x80
				binary.BigEndian.PutUint16(buf[2:4], seqno)
				binary.BigEndian.PutUint16(buf[4:6], rr.first)
				binary.BigEndian.PutUint16(buf[6:8], rr.count)

				conn.Write(buf[0:8])
				rtplog.Debug.Println(prefix, "Recovery Request:", rr, " sent to ", conn.RemoteAddr())
			}
			seqno++
		}
	}
	return rx, tx, "CONTROL"
}

func (r *raop) getTimingHandler(raddr *net.UDPAddr) (rtpHandler, rtpTransmitter, string) {
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

func startRtp(f rtpFactory, raddr *net.UDPAddr) (*rtp, error) {
	var conn *net.UDPConn
	var err error

	if raddr == nil {
		rtplog.Debug.Println("LISTENING")
		conn, err = net.ListenUDP("udp", nil)
	} else {
		rtplog.Debug.Println("DIALING raddr=", raddr)
		conn, err = net.DialUDP("udp", nil, raddr)
	}
	if err != nil {
		return nil, err
	}

	handler, tx, name := f(raddr)
	rtplog.Debug.Println("Starting RTP server ", name, " at conn local=", conn.LocalAddr(), ", remote=", conn.RemoteAddr())
	if handler != nil {
		go func() {
			defer func() { conn.Close() }()
			for {
				pkt := makeRtpPacket()
				pkt.debug(name)
				var n int
				var err error
				n, err = conn.Read(pkt.buf)
				if err != nil {
					rtplog.Info.Println("Panic err=", err)
					return // Exit RTP server
				}
				pkt.content = pkt.buf[0:n]
				pkt.seqno = binary.BigEndian.Uint16(pkt.content[2:4])
				pkt.debug(name)
				handler(pkt)

			}
		}()
	}
	if tx != nil {
		go tx(conn)
		tx = nil
	}
	return (*rtp)(conn), nil
}
