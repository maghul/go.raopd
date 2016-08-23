// +build !debug_pool

package raopd

import (
	"sync"
)

type rtpPacket struct {
	sn      seqno
	content []byte
	buf     []byte
}

var rtpPacketPool = &sync.Pool{New: func() interface{} {
	return &rtpPacket{0, nil, make([]byte, max_rtp_packet_size)}
}}

func makeRtpPacket() *rtpPacket {
	return rtpPacketPool.Get().(*rtpPacket)
}

func (pkt *rtpPacket) Reclaim() {
	rtpPacketPool.Put(pkt)
}

func (pkt *rtpPacket) debug(ref string) {
}
