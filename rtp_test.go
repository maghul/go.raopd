package raopd

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func checkSeqNo(t *testing.T, q chan *rtpPacket, expected int) {
	select {
	case p := <-q:
		assert.NotEqual(t, -1, expected, fmt.Sprintf("Queue should be empty, contained packet with seqno=%d", p.sn))
		assert.Equal(t, seqno(expected), p.sn)
	case <-time.After(time.Millisecond * 1):
		assert.Equal(t, -1, expected, fmt.Sprintf("Queue is empty, should contain packet with seqno=%d", expected))
	}
}

func TestRtpDataReceive(t *testing.T) {
	r := &raop{}
	r.seqchan = make(chan *rtpPacket, 16)

	handler, _, _ := r.getDataHandler()

	pkt := testPacket(66, 96)
	handler(pkt)

	checkSeqNo(t, r.seqchan, 66)
	checkSeqNo(t, r.seqchan, -1)
}

func startRtpMock(r *raop, f rtpFactory) *net.UDPConn {
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
	r := &raop{}
	conn := startRtpMock(r, r.getDataHandler)

	conn.Write(testPacket(66, 96).content)

	checkSeqNo(t, r.seqchan, 66)
	checkSeqNo(t, r.seqchan, -1)
}

func TestRtpControlReceive(t *testing.T) {
	r := &raop{}
	conn := startRtpMock(r, r.getControlHandler)

	conn.Write(testPacket(68, 86).content)

	checkSeqNo(t, r.seqchan, 0) // Will rewrite the packet and therefore the sequence number
	checkSeqNo(t, r.seqchan, -1)
}
