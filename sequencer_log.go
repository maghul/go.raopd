package raopd

import (
	"fmt"
	"io"
)

type sequencelog struct {
	tracelog
	all bool // Set to true to log normal packets too.
}

func (sl *sequencelog) inputPacket(pkt *rtpPacket, state string) {
	if sl == nil {
		return
	}
	sn := pkt.sn
	sl.log(func(wr io.Writer) {
		fmt.Fprintln(wr, "INPUT PACKET ", sn, state)
	})
}

func (sl *sequencelog) outputPacket(pkt *rtpPacket) {
	if sl == nil {
		return
	}
	sn := pkt.sn
	sl.log(func(wr io.Writer) {
		fmt.Fprintln(wr, "OUTPUT PACKET ", sn)
	})
}

func (sl *sequencelog) reRequest(rr *rerequest, retry int) {
	if sl == nil {
		return
	}
	sl.log(func(wr io.Writer) {
		fmt.Fprintln(wr, "REREQUEST ", rr.first, "...", rr.first+rr.count-1, ", RETRY=", retry)
	})
}

func (sl *sequencelog) removePackets(start, count seqno) {
	if sl == nil {
		return
	}
	sl.log(func(wr io.Writer) {
		fmt.Fprintln(wr, "REMOVE ", start, "...", start+count-1)
	})
}

func (sl *sequencelog) note(msg ...interface{}) {
	if sl == nil {
		return
	}
	sl.log(func(wr io.Writer) {
		fmt.Fprintln(wr, msg...)
	})
}
