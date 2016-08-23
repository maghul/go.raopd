package raopd

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func readRaopResponse(cr *bufio.Reader) string {
	b := bytes.NewBufferString("")
	for {
		line, err := cr.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading socket: %v", err)
		}

		b.WriteString(line)
		if line == "\n" || line == "\r\n" {
			return b.String()
		}
	}
}

func raopTxRx(cw *bufio.Writer, cr *bufio.Reader, msg string) string {
	raoplog.Debug.Println("------------------------------------------------------------------------------------------------------------------")
	cw.WriteString(msg)
	cw.Flush()
	response := readRaopResponse(cr)
	raoplog.Debug.Println("RX:", response)
	raoplog.Debug.Println("------------------------------------------------------------------------------------------------------------------")
	return response
}

func TestRaopSetup(t *testing.T) {
	rf, err := NewSinkCollection("testdata/airport.key")
	if err != nil {
		panic(err)
	}

	source, err := rf.Register(makeTestClient())
	if err != nil {
		panic(err)
	}
	raoplog.Debug.Println("RAOP session started...")
	assert.Equal(t, "RAOP: hw=11:22:33:13:37:17", source.raop.String())
	assert.NotNil(t, source)


	conn, err := net.Dial("tcp", "127.0.0.1:15100")
	if err != nil {
		panic(err)
	}

	cr := bufio.NewReader(conn)
	cw := bufio.NewWriter(conn)

	raopTxRx(cw, cr, `OPTIONS * RTSP/1.0
Apple-Challenge: zrCksUWuXk5RqijsFIRXDw==
CSeq: 0
DACP-ID: 19050F2FE0FD618D
Active-Remote: 84694584
User-Agent: AirPlay/267.3

`)

	raopTxRx(cw, cr, `ANNOUNCE rtsp://fe80::461e:a1ff:fece:f4a9/9953613529495192746 RTSP/1.0
Content-Length: 657
Content-Type: application/sdp
CSeq: 1
DACP-ID: 19050F2FE0FD618D
Active-Remote: 84694584
User-Agent: AirPlay/267.3

v=0
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
`)

	raopTxRx(cw, cr, `SETUP rtsp://fe80::461e:a1ff:fece:f4a9/9953613529495192746 RTSP/1.0
Transport: RTP/AVP/UDP;unicast;mode=record;timing_port=53595;control_port=54411
CSeq: 2
DACP-ID: 19050F2FE0FD618D
Active-Remote: 84694584
User-Agent: AirPlay/267.3

`)

}
