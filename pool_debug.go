// +build debug_pool

package raopd

import (
	"fmt"
	"sync"
	"time"
)

type rtpPacket struct {
	seqno   uint16
	content []byte
	buf     []byte
	allocno int
}

var rtpPacketPoolCounter = 0
var rtpPacketPool = &sync.Pool{New: func() interface{} {
	rtpPacketPoolCounter++
	log.Debug().Println("RTP PACKET POOL: Created ", rtpPacketPoolCounter, " RTP packets in pool")
	return &rtpPacket{0, nil, make([]byte, max_rtp_packet_size), rtpPacketPoolCounter - 1}
}}

var allocs = make(map[int]int)
var allocmutex = new(sync.Mutex)

func (pkt *rtpPacket) debug(ref string) {
	log.Debug().Println(ref, ":", pkt.allocno, ", seq=", pkt.seqno)
}

func makeRtpPacket() *rtpPacket {
	pkt := rtpPacketPool.Get().(*rtpPacket)
	allocmutex.Lock()
	defer allocmutex.Unlock()
	allocs[pkt.allocno] = 1
	return pkt
}

func (pkt *rtpPacket) Reclaim() {
	pkt.debug("RECLAIM")
	rtpPacketPool.Put(pkt)

	allocmutex.Lock()
	defer allocmutex.Unlock()
	allocs[pkt.allocno] = 0
}

func getAllocs() []int {
	rv := make([]int, rtpPacketPoolCounter)
	allocmutex.Lock()
	defer allocmutex.Unlock()

	for ii := 0; ii < rtpPacketPoolCounter; ii++ {
		rv[ii] = allocs[ii]
	}
	return rv
}

func init() {
	go func() {
		log.Debug().Println("ALLOCS STARTED")
		for {
			time.Sleep(10 * time.Second)
			for ii, v := range getAllocs() {
				log.Debug().Println("ALLOCS: ", ii, "=", v)
			}
		}
	}()
}
