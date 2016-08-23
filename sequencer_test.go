package raopd

import (
	"encoding/binary"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func inSeqRange(in chan *rtpPacket, from, to int) {
	for ii := from; ii != to+1; ii++ {
		in <- testPacket(uint16(ii), 0)
	}
}

func inSeqs(in chan *rtpPacket, va ...interface{}) {
	for _, v := range va {
		switch v := v.(type) {
		case int:
			in <- testPacket(uint16(v), 0)
		case []int:
			inSeqRange(in, v[0], v[1])
		default:
			panic("unknown type")
		}
	}
}

func checkSeqNo(t *testing.T, q chan *rtpPacket, expected int) {
	debug("CHECK SEQNO expected=", expected)
	select {
	case p := <-q:
		debug("CHECK SEQNO received=", p.seqno)
		assert.NotEqual(t, -1, expected, fmt.Sprintf("Queue should be empty, contained packet with seqno=%d", p.seqno))
		assert.Equal(t, uint16(expected), p.seqno)
	case <-time.After(time.Millisecond * 1):
		debug("CHECK SEQNO  *empty*")
		assert.Equal(t, -1, expected, fmt.Sprintf("Queue is empty, should contain packet with seqno=%d", expected))
	}
}

func checkSeqNos(t *testing.T, q chan *rtpPacket, from, to int) {
	debug("CHECK SEQNO from=", from, ", to=", to)
	for ii := from; ii <= to; ii++ {
		checkSeqNo(t, q, ii)
	}
	checkSeqNo(t, q, -1)
}

func checkReq(t *testing.T, q chan rerequest, expected, count int) {
	debug("CHECK REREQUEST", expected, "...", expected+count-1)
	select {
	case r := <-q:
		debug("CHECK REREQUEST: received", r)
		assert.NotEqual(t, -1, expected, "Request Queue should be empty")
		debug("CHECK REREQUEST: ", expected, int(r.first))
		assert.Equal(t, expected, int(r.first))
		assert.Equal(t, count, int(r.count))
	case <-time.After(time.Millisecond * 1):
		debug("CHECK REREQUEST: *empty")
		msg := fmt.Sprintf("Request was empty, should contain [%d...%d]", expected, expected+count-1)
		assert.Equal(t, -1, expected, msg)
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
	debug("TestSequenceConsecutive")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	s := startSequencer(in, of, request)

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

	s.close()
}

func TestSequenceSingleGap(t *testing.T) {
	debug("TestSequenceSingleGap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	s := startSequencer(in, of, request)

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

	s.close()
}

func TestSequenceDoubleGap(t *testing.T) {
	debug("TestSequenceDoublegap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	s := startSequencer(in, of, request)

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

	s.close()
}

func TestSequenceWideGap(t *testing.T) {
	debug("TestSequenceDoublegap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	s := startSequencer(in, of, request)

	inSeqs(in, []int{46542, 46544})
	// gap 46545..46553
	inSeqs(in, 46554, 46549, []int{46555, 46559})
	// gap 46560..46568
	inSeqs(in, []int{46569, 46570})

	checkSeqNos(t, out, 46542, 46544)

	time.Sleep(11 * time.Millisecond)

	inSeqs(in, 46545, []int{46547, 46553})
	checkSeqNos(t, out, 46545, 46545)

	time.Sleep(11 * time.Millisecond)

	in <- testPacket(46546, 0)

	time.Sleep(11 * time.Millisecond)

	checkSeqNos(t, out, 46546, 46559)

	checkReq(t, request, 46545, 4)
	checkReq(t, request, 46550, 4)
	checkReq(t, request, 46560, 9)
	checkReq(t, request, 46546, 1)
	checkReq(t, request, -1, 0)

	s.close()
}

func TestSequenceReReRequest(t *testing.T) {
	debug("TestSequenceDoublegap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		debug("       OUT PACKET=", pkt.seqno)
		out <- pkt
	}
	s := startSequencer(in, of, request)

	inSeqs(in, []int{46542, 46544})
	// gap 46545..46553
	inSeqs(in, 46554, 46549, []int{46555, 46559})
	// gap 46560..46568
	inSeqs(in, []int{46569, 46570})

	checkSeqNos(t, out, 46542, 46544)

	time.Sleep(11 * time.Millisecond)

	inSeqs(in, 46545, []int{46547, 46553})
	checkSeqNos(t, out, 46545, 46545)

	time.Sleep(11 * time.Millisecond)

	inSeqs(in, 46570)

	time.Sleep(11 * time.Millisecond)

	inSeqs(in, 46546)

	time.Sleep(11 * time.Millisecond)

	checkSeqNos(t, out, 46546, 46559)

	checkReq(t, request, 46545, 4)
	checkReq(t, request, 46550, 4)
	checkReq(t, request, 46560, 9)
	checkReq(t, request, 46546, 1)
	checkReq(t, request, 46546, 1)
	checkReq(t, request, 46560, 9)
	checkReq(t, request, -1, 0)

	s.close()
}

func TestSequenceSeqNo(t *testing.T) {
	assert.Equal(t, 0, seqnoDelta(4711, 4711))
	assert.Equal(t, 1, seqnoDelta(4712, 4711))
	assert.Equal(t, -1, seqnoDelta(4711, 4712))
	assert.Equal(t, 16, seqnoDelta(10, 65530))
	assert.Equal(t, -16, seqnoDelta(65530, 10))
}
