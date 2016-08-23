package raopd

import (
	"encoding/binary"
	"fmt"
)

type seqno uint16

const seqnoguard = 8192

func seqnoDelta(a, b seqno) int {
	switch {
	case a == b:
		return 0
	case (a < seqnoguard && b > 65536-seqnoguard):
		return 65536 - int(b-a)
	case (b < seqnoguard && a > 65536-seqnoguard):
		return int(a-b) - 65536
	case a > b:
		return int(a - b)
	case a < b:
		return -int(b - a)
	}
	panic(fmt.Sprint("seqnoDelta, no case for a=", a, ", b=", b))
}

func (sn seqno) encode(buf []byte) {
	binary.BigEndian.PutUint16(buf, uint16(sn))
}

func decodeSeqno(buf []byte) seqno {
	return seqno(binary.BigEndian.Uint16(buf))
}
