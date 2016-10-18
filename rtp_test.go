package raopd

import (
	"net"
	"testing"
)

func TestRtpDataReceive(t *testing.T) {
	r := &Raop{}
	r.seqchan = make(chan *rtpPacket, 16)

	handler, _, _ := r.getDataHandler()

	pkt := testPacket(66, 96)
	handler(pkt)

	checkSeqNo(t, r.seqchan, 66)
	checkSeqNo(t, r.seqchan, -1)
}

func startRtpMock(r *Raop, f rtpFactory) *net.UDPConn {
	r.seqchan = make(chan *rtpPacket, 16)

	rtp := startRtp(f)

	addr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: int(rtp.Port()), Zone: ""}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}
	return conn
}

func TestRtpDataReceive2(t *testing.T) {
	r := &Raop{}
	conn := startRtpMock(r, r.getDataHandler)

	conn.Write(testPacket(66, 96).content)

	checkSeqNo(t, r.seqchan, 66)
	checkSeqNo(t, r.seqchan, -1)
}

func TestRtpControlReceive(t *testing.T) {
	r := &Raop{}
	conn := startRtpMock(r, r.getControlHandler)

	conn.Write(testPacket(68, 86).content)

	checkSeqNo(t, r.seqchan, 68)
	checkSeqNo(t, r.seqchan, -1)
}
