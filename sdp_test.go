package raopd

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSdp(t *testing.T) {
	sdpdata := `v=0
o=AirTunes 9953613529495192746 0 IN IP6 fe80::495:eb8c:ffb8:8083
s=AirTunes
i=Magnus iPhone
c=IN IP6 fe80::495:eb8c:ffb8:8083
t=0 0
m=audio 0 RTP/AVP 96
a=rtpmap:96 AppleLossless
a=fmtp:96 352 0 16 40 10 14 2 255 0 0 44100
a=rsaaeskey:SOcIgAMprqG1ET7Hd6ndqWsb4UzoQ+337gSxLQ0lYsheKvwF2VvVC8n8Cn90GB8BTA0iPmVFInHgBZlIcBmqVf6MmczfJMgEyPoBaHBhx2Qk1fP+6nhDFKGzPpMP88F6edaF956+5bevtGkhX/8Xv7p4oqhipZgpV9y4IZMmFFyp3vAowUPDtYVqv7Gvhvavq2JMQC5vFi+yHZ5H5NLhRmiOiGAihd5tDFYO+1XY4E1A3MJjn+O4s/yyYrT1sne/ZKw4ssckCwFvyYR4bZ2Isu9pkLo+njnlTtyE6o8rFjr6tP5yt1NMqARD1cReA3vWG6YF2Hl+2lq6DvwiBuVlbA==
a=aesiv:ts4b86KgrpXPdjvEkPOQdg==
a=min-latency:11025
a=max-latency:88200
`

	sdp := makeSdpRecords(bytes.NewBufferString(sdpdata))

	assert.NotNil(t, sdp)
	assert.Equal(t, "AirTunes", sdp["s"])
}
