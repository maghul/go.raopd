package raopd

import (
	"encoding/binary"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func checkSeqNo(t *testing.T, q chan *rtpPacket, expected int) {
	select {
	case p := <-q:
		assert.NotEqual(t, -1, expected, "Queue should be empty")
		assert.Equal(t, uint16(expected), p.seqno)
	case <-time.After(time.Millisecond * 1):
		assert.Equal(t, -1, expected, "Queue should not be empty")
	}
}

func checkSeqNos(t *testing.T, q chan *rtpPacket, from, to int) {
	for ii := from; ii <= to; ii++ {
		checkSeqNo(t, q, ii)
	}
	checkSeqNo(t, q, -1)
}

func checkReq(t *testing.T, q chan rerequest, expected, count int) {
	select {
	case r := <-q:
		assert.NotEqual(t, -1, expected, "Queue should be empty")
		assert.Equal(t, uint16(expected), r.first)
		assert.Equal(t, uint16(count), r.count)
	case <-time.After(time.Millisecond * 1):
		assert.Equal(t, expected, -1, "Queue should not be empty")
	}
}

func testPacket(seqno uint16, payloadType uint8) *rtpPacket {
	pkt := makeRtpPacket()
	pkt.seqno = seqno
	buf := pkt.buf[0:32]
	pkt.content = buf
	buf[1] = payloadType
	binary.BigEndian.PutUint16(buf[2:4], seqno)
	return pkt
}

func TestSequenceConsecutive(t *testing.T) {
	fmt.Println("TestSequenceConsecutive")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	startSequencer(in, of, request)

	in <- testPacket(4, 0)
	in <- testPacket(5, 0)
	in <- testPacket(6, 0)
	in <- testPacket(7, 0)

	checkSeqNo(t, out, 4)
	checkSeqNo(t, out, 5)
	checkSeqNo(t, out, 6)
	checkSeqNo(t, out, 7)
	checkSeqNo(t, out, -1)
	checkReq(t, request, -1, 0)

}

func TestSequenceSingleGap(t *testing.T) {
	fmt.Println("TestSequenceSingleGap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	startSequencer(in, of, request)

	in <- testPacket(4, 0)
	in <- testPacket(6, 0)
	in <- testPacket(5, 0)
	in <- testPacket(7, 0)

	checkSeqNo(t, out, 4)
	checkSeqNo(t, out, 5)
	checkSeqNo(t, out, 6)
	checkSeqNo(t, out, 7)
	checkSeqNo(t, out, -1)
	checkReq(t, request, -1, 0)

}

func TestSequenceDoubleGap(t *testing.T) {
	fmt.Println("TestSequenceDoublegap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	startSequencer(in, of, request)

	in <- testPacket(4, 0)
	in <- testPacket(7, 0)
	in <- testPacket(6, 0)
	in <- testPacket(5, 0)

	checkSeqNo(t, out, 4)
	checkSeqNo(t, out, 5)
	checkSeqNo(t, out, 6)
	checkSeqNo(t, out, 7)
	checkSeqNo(t, out, -1)
	checkReq(t, request, -1, 0)

}

func TestSequenceWideGap(t *testing.T) {
	fmt.Println("TestSequenceDoublegap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	startSequencer(in, of, request)

	in <- testPacket(46542, 0)
	in <- testPacket(46543, 0)
	in <- testPacket(46544, 0)

	in <- testPacket(46554, 0)
	in <- testPacket(46549, 0) //
	in <- testPacket(46555, 0)
	in <- testPacket(46556, 0)
	in <- testPacket(46557, 0)
	in <- testPacket(46558, 0)
	in <- testPacket(46559, 0)

	in <- testPacket(46569, 0)
	in <- testPacket(46570, 0)

	checkSeqNos(t, out, 46542, 46544)

	time.Sleep(1000)

	in <- testPacket(46545, 0)
	in <- testPacket(46547, 0)
	in <- testPacket(46548, 0)
	in <- testPacket(46549, 0)
	in <- testPacket(46550, 0)
	in <- testPacket(46551, 0)
	in <- testPacket(46552, 0)
	in <- testPacket(46552, 0)
	in <- testPacket(46553, 0)

	checkSeqNos(t, out, 46545, 46545)

	time.Sleep(1000)

	in <- testPacket(46546, 0)

	checkSeqNos(t, out, 46546, 46559)

	checkReq(t, request, 46545, 4)
	checkReq(t, request, 46550, 4)
	checkReq(t, request, 46560, 9)
	checkReq(t, request, 46546, 1)
	checkReq(t, request, -1, 0)

}
