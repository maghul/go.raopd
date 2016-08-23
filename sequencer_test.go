package raopd

import (
	"emh/logger"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const sequencerDebug = false

func inSeqRange(in chan *rtpPacket, from, to int) {
	for ii := from; ii != to+1; ii++ {
		in <- testPacket(seqno(ii), 0)
	}
}

func inSeqs(in chan *rtpPacket, va ...interface{}) {
	for _, v := range va {
		switch v := v.(type) {
		case int:
			in <- testPacket(seqno(v), 0)
		case []int:
			inSeqRange(in, v[0], v[1])
		default:
			panic("unknown type")
		}
	}
}

func checkSeqNo(t *testing.T, q chan *rtpPacket, expected int) {
	if sequencerDebug {
		seqlog.Debug.Println("CHECK SEQNO expected=", expected)
	}
	select {
	case p := <-q:
		if sequencerDebug {
			seqlog.Debug.Println("CHECK SEQNO received=", p.sn)
		}
		assert.NotEqual(t, -1, expected, fmt.Sprintf("Queue should be empty, contained packet with seqno=%d", p.sn))
		assert.Equal(t, seqno(expected), p.sn)
	case <-time.After(time.Millisecond * 1):
		if sequencerDebug {
			seqlog.Debug.Println("CHECK SEQNO  *empty*")
		}
		assert.Equal(t, -1, expected, fmt.Sprintf("Queue is empty, should contain packet with seqno=%d", expected))
	}
}

func checkSeqNos(t *testing.T, q chan *rtpPacket, from, to int) {
	seqlog.Debug.Println("CHECK SEQNO from=", from, ", to=", to)
	for ii := from; ii <= to; ii++ {
		checkSeqNo(t, q, ii)
	}
	checkSeqNo(t, q, -1)
}

func checkReq(t *testing.T, q chan rerequest, expected, count int) {
	if sequencerDebug {
		seqlog.Debug.Println("CHECK REREQUEST", expected, "...", expected+count-1)
	}
	select {
	case r := <-q:
		if sequencerDebug {
			seqlog.Debug.Println("CHECK REREQUEST: received", r)
		}
		assert.NotEqual(t, -1, expected, "Request Queue should be empty")
		if sequencerDebug {
			seqlog.Debug.Println("CHECK REREQUEST: ", expected, int(r.first))
		}
		assert.Equal(t, expected, int(r.first))
		assert.Equal(t, count, int(r.count))
	case <-time.After(time.Millisecond * 1):
		if sequencerDebug {
			seqlog.Debug.Println("CHECK REREQUEST: *empty")
		}
		msg := fmt.Sprintf("Request was empty, should contain [%d...%d]", expected, expected+count-1)
		assert.Equal(t, -1, expected, msg)
	}
}

func testPacket(sn seqno, payloadType uint8) *rtpPacket {
	pkt := makeRtpPacket()
	pkt.sn = sn
	buf := pkt.buf[0:32]
	pkt.content = buf
	buf[1] = payloadType
	sn.encode(buf[2:4])
	return pkt
}

func TestSequenceConsecutive(t *testing.T) {
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
	seqlog.Debug.Println("TestSequenceSingleGap")
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
	seqlog.Debug.Println("TestSequenceDoublegap")
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
	seqlog.Debug.Println("TestSequenceDoublegap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	s := startSequencer(in, of, request)

	inSeqs(in, []int{46542, 46544})               // 46542..46544
	inSeqs(in, 46554, 46549, []int{46555, 46559}) // 46542..46544 46549 46554..46559
	checkSeqNos(t, out, 46542, 46544)             // 46542..46544 <-- 46549 46554..46559

	inSeqs(in, 46545, []int{46547, 46553}) // 46545 465474..46559
	checkSeqNos(t, out, 46545, 46545)      // 46545 <-- 465474..46559

	time.Sleep(11 * time.Millisecond)
	checkReq(t, request, 46546, 1) // ??1 46546..46546
	checkReq(t, request, -1, 0)
	inSeqs(in, []int{46569, 46570}) // 46545 <-- 465474..46559 46569..46570

	time.Sleep(11 * time.Millisecond)
	checkReq(t, request, 46560, 9) // ??1 46560..46568
	checkReq(t, request, -1, 0)

	time.Sleep(22 * time.Millisecond)
	inSeqs(in, 46546)                 // 46546..46559 46569..46570
	checkSeqNos(t, out, 46546, 46559) // 46546..46559 <-- 46569..46570
	checkReq(t, request, 46546, 1)    // ??2 46546..46546
	checkReq(t, request, -1, 0)

	time.Sleep(11 * time.Millisecond)
	checkReq(t, request, 46560, 9) // ??2 46560..46568
	checkReq(t, request, -1, 0)

	s.close()
}

func TestSequenceReReRequest(t *testing.T) {
	seqlog.Debug.Println("TestSequenceDoublegap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 20)
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

	inSeqs(in, 46570)

	checkReq(t, request, 46545, 4)
	checkReq(t, request, 46550, 4)
	checkReq(t, request, 46560, 9)
	checkReq(t, request, -1, 0)

	time.Sleep(33 * time.Millisecond)

	inSeqs(in, 46546)
	checkSeqNos(t, out, 46546, 46559)

	// Second rerequests
	println("SECOND REREQUEST")
	checkReq(t, request, 46546, 1)
	checkReq(t, request, 46560, 9)
	checkReq(t, request, -1, 0)

	println("SECOND REREQUEST DONE")
	s.close()
	println("TEST DONE")

}

func NoTestSequenceSlowResponse(t *testing.T) {
	seqlog.Debug.Println("TestSequenceDoublegap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	s := startSequencer(in, of, request)

	inSeqs(in, []int{46542, 46544})
	inSeqs(in, []int{46547, 46554})

	checkSeqNos(t, out, 46542, 46544)
	time.Sleep(11 * time.Millisecond)

	// These are packets coming from the normal data RTP session and should
	// not retrigger a request which is already pending.
	inSeqs(in, []int{46555, 46556})
	time.Sleep(11 * time.Millisecond)

	checkReq(t, request, 46545, 2)
	// We should only have a single rerequest here
	checkReq(t, request, -1, 0)

	s.close()
}

func TestSequenceMissingRecoveryPacket(t *testing.T) {
	seqlog.Debug.Println("TestSequenceDoublegap")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	s := startSequencer(in, of, request)

	inSeqs(in, []int{46542, 46544})
	inSeqs(in, []int{46547, 46554})

	checkSeqNos(t, out, 46542, 46544)
	time.Sleep(11 * time.Millisecond)

	// This packet is a recovery but 46545 is still missing so we should
	// get a new request for that.
	inSeqs(in, 46546)
	time.Sleep(33 * time.Millisecond)

	checkReq(t, request, 46545, 2)
	checkReq(t, request, 46545, 1)
	checkReq(t, request, -1, 0)

	s.close()
}

func TestSequenceGiveUp(t *testing.T) {
	seqlog.Debug.Println("TestSequenceGiveUp")
	in := make(chan *rtpPacket, 10)
	out := make(chan *rtpPacket, 10)
	request := make(chan rerequest, 10)

	of := func(pkt *rtpPacket) {
		out <- pkt
	}
	s := startSequencer(in, of, request)

	inSeqs(in, []int{46542, 46544}) // 46542..46544
	inSeqs(in, []int{46547, 46554}) // 46542..46544  46547..46554

	checkSeqNos(t, out, 46542, 46544) // 46542..46544  <-- 46547..46554
	time.Sleep(11 * time.Millisecond)

	// This packet is a recovery but 46545 is still missing so we should
	// get a new request for that.
	inSeqs(in, 46546) // 46546..46554

	time.Sleep(11 * time.Millisecond)
	//	checkSeqNos(t, out, 46547, 46554)
	checkReq(t, request, 46545, 2) // ??1 46545..46546?
	checkReq(t, request, -1, 0)

	time.Sleep(11 * time.Millisecond)
	checkReq(t, request, -1, 0)

	time.Sleep(22 * time.Millisecond)
	checkReq(t, request, 46545, 1) // ??2 46545..46545?
	checkReq(t, request, -1, 0)

	time.Sleep(11 * time.Millisecond)
	checkReq(t, request, -1, 0)

	time.Sleep(44 * time.Millisecond)
	checkReq(t, request, 46545, 1) // ??3 46545..46545?
	checkReq(t, request, -1, 0)

	time.Sleep(11 * time.Millisecond)
	checkReq(t, request, -1, 0)

	time.Sleep(55 * time.Millisecond)
	println("7")
	checkReq(t, request, -1, 0)
	// We should not see any more requests for 46545
	//checkReq(t, request, -1, 0)

	s.close()
}

func TestSequencerRemove(t *testing.T) {
	s := &sequencer{}
	s.restartSequencer()
	Debug("log.debug/raopd.sequencer", 1)
	Debug("log.info/raopd.sequencer", 1)
	rrc := make(chan rerequest, 10)
	outf := func(rp *rtpPacket) {

	}
	s.lowd = true
	s.low = 117
	s.handle(testPacket(127, 0), outf)
	s.handle(testPacket(137, 0), outf)
	s.remove(117, 10)
	s.sendReRequests(rrc)
	s.sendReRequests(rrc)
	s.sendReRequests(rrc)

	//	s.checkReq(t, rrc, 117, 10)
	s.checkReq(t, rrc, 128, 9)
	s.checkReq(t, rrc, -1, 0)
}

func TestSequenceSeqNo(t *testing.T) {
	assert.Equal(t, 0, seqnoDelta(4711, 4711))
	assert.Equal(t, 1, seqnoDelta(4712, 4711))
	assert.Equal(t, -1, seqnoDelta(4711, 4712))
	assert.Equal(t, 16, seqnoDelta(10, 65530))
	assert.Equal(t, -16, seqnoDelta(65530, 10))
}
