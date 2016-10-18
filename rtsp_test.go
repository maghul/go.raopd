package raopd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testClient struct {
	volume   float32
	pos, end int
}

func (tc *testClient) LoadCoverArt(mimetype string, content io.Reader) {
}

func (tc *testClient) LoadMetadata(content io.Reader) {
}

func (tc *testClient) SetVolume(volume float32) {
	tc.volume = volume
}

func (tc *testClient) SetProgress(pos, end int) {
	tc.pos = pos
	tc.end = end
}

func (tc *testClient) Play() {
	fmt.Println("TEST CLIENT:", "Play...")
}

func (tc *testClient) Pause() {
	fmt.Println("TEST CLIENT:", "Pause...")
}

func (tc *testClient) AudioWriter() io.Writer {
	fmt.Println("TEST CLIENT:", "AudioWrite")
	panic("I'm sorry Dave, I can't allow you to do that.")
}

func makeTestRtspSession() *rtspSession {
	i, err := makeInfo(getKeyfile())
	if err != nil {
		panic(err)
	}
	r := &Raop{}
	r.samplingRate = 44100
	r.plc = &testClient{}
	return &rtspSession{i, r, nil}
}

func TestParseRequest1(t *testing.T) {
	rs := `OPTIONS * RTSP/1.0
CSeq: 3
User-Agent: iTunes/10.6 (Macintosh; Intel Mac OS X 10.7.3) AppleWebKit/535.18.5
Client-Instance: 56B29BB6CB904862
DACP-ID: 56B29BB6CB904862
Active-Remote: 1986535575

`
	r := makeTestRtspSession()

	req, err := r.readRequest(ioutil.NopCloser(bytes.NewBufferString(rs)))

	assert.Nil(t, err)
	assert.Equal(t, "OPTIONS", req.Method)
}

type TestResponse struct {
	resp    *http.Response
	content *bytes.Buffer
}

func MakeTestResponse() *TestResponse {
	t := &TestResponse{&http.Response{}, bytes.NewBufferString("")}
	t.resp.Proto = "RTSP/1.0"
	t.resp.ProtoMajor = 1
	t.resp.ProtoMinor = 0
	t.resp.Status = ""
	t.resp.Header = http.Header(make(map[string][]string))
	t.content = bytes.NewBufferString("")
	t.resp.Body = ioutil.NopCloser(t.content)
	return t
}

func (t *TestResponse) Header() http.Header {
	return t.resp.Header
}

func (t *TestResponse) defaultHeader() {
	if t.resp.Status == "" {
		t.WriteHeader(200)
	}
}

func (t *TestResponse) Write(d []byte) (int, error) {
	t.defaultHeader()
	return t.content.Write(d)
}

func (t *TestResponse) WriteHeader(statusCode int) {
	t.resp.StatusCode = statusCode
	t.resp.Status = fmt.Sprintf("%d", statusCode)
}

func request(r *rtspSession, req string) (resp *http.Response, err error) {
	tr := MakeTestResponse()
	rr, err := r.readRequest(ioutil.NopCloser(bytes.NewBufferString(req)))
	if err != nil {
		return nil, err
	}
	r.handle(tr, rr)
	tr.defaultHeader()
	return tr.resp, nil
}

type headerAsserter struct {
	t      *testing.T
	header http.Header
}

func (ha *headerAsserter) assert(expected string, name string) {
	exp := []string{expected}
	assert.Equal(ha.t, exp, ha.header[name], name)
}

func TestChallengeResponse(t *testing.T) {
	req := `OPTIONS * RTSP/1.0
Apple-Challenge: zrCksUWuXk5RqijsFIRXDw==
CSeq: 0
DACP-ID: 19050F2FE0FD618D
Active-Remote: 84694584
User-Agent: AirPlay/267.3

`
	r := makeTestRtspSession()
	resp, err := request(r, req)

	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode, "StatusCode")
	ha := headerAsserter{t, resp.Header}
	ha.assert("0", "Cseq")
	ha.assert("connected; type=analog", "Apple-Jack-Status")
	ha.assert("VwIP3Ghly/c8X7nnvOowSq38HMKRx9XYys+yOMqAhQShmBT2tdQtjU71zlPsJGLHMV3mJyelvIGJEjituU9SFPQehk04gOsu3WYT1aWXfg6nkSNvvlyWThOm92FNY5oLEnGpfotInqC1ZyR11PkPbZnHFJqzcq1qdUeQ4QSdTkIAu2tknl0eH0t4srEVF1ITacG4yBtx0vf4QA9oe3ltcjIbpKueIXi6D5TE9LxW55m5Y0suI2KGUZlwYYWIMWgTXOK0Fgh+rPK98RT0SxZTUtnK/dZN2HGqcpHqNRy6MriXiuaOctRotdhUtfqTMsQQ/x+MMDcMx3IjY1Y0ER7cCw", "Apple-Response")
	ha.assert("ANNOUNCE, SETUP, RECORD, PAUSE, FLUSH, TEARDOWN, OPTIONS, GET_PARAMETER, SET_PARAMETER", "Public")
}

func TestAnnounce(t *testing.T) {
	req := `ANNOUNCE rtsp://fe80::461e:a1ff:fece:f4a9/9953613529495192746 RTSP/1.0
Content-Length: 670
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
`
	r := makeTestRtspSession()
	resp, err := request(r, req)

	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode, "StatusCode")
	ha := headerAsserter{t, resp.Header}
	ha.assert("1", "Cseq")
	ha.assert("connected; type=analog", "Apple-Jack-Status")

	// Check the RTP setup and encryption key
}

func TestSetup(t *testing.T) {
	req := `SETUP rtsp://fe80::461e:a1ff:fece:f4a9/9953613529495192746 RTSP/1.0
Transport: RTP/AVP/UDP;unicast;mode=record;timing_port=53595;control_port=54411
CSeq: 2
DACP-ID: 19050F2FE0FD618D
Active-Remote: 84694584
User-Agent: AirPlay/267.3

`
	r := makeTestRtspSession()
	resp, err := request(r, req)

	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode, "StatusCode")
	ha := headerAsserter{t, resp.Header}
	ha.assert("2", "Cseq")
	ha.assert("connected; type=analog", "Apple-Jack-Status")
	ha.assert("RTP/AVP/UDP;unicast;mode=record;timing_port=47021;events;control_port=42365;server_port=42148\nSession: DEADBEEF", "Transport")

	assert.Equal(t, "19050F2FE0FD618D", r.raop.dacpID)
	assert.Equal(t, "84694584", r.raop.activeRemote)
	assert.Equal(t, "AirPlay/267.3", r.raop.clientUserAgent)
}

func TestGetParameter(t *testing.T) {
	req := `GET_PARAMETER rtsp://fe80::461e:a1ff:fece:f4a9/9953613529495192746 RTSP/1.0
Content-Length: 8
Content-Type: text/parameters
CSeq: 3
DACP-ID: 19050F2FE0FD618D
Active-Remote: 84694584
User-Agent: AirPlay/267.3

volume
`
	r := makeTestRtspSession()
	resp, err := request(r, req)

	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode, "StatusCode")
	ha := headerAsserter{t, resp.Header}
	ha.assert("3", "Cseq")
	ha.assert("connected; type=analog", "Apple-Jack-Status")

	b := readToString(resp.Body)
	assert.Equal(t, "volume: 0.000000\n", b)
}

func TestSetParameterVolume(t *testing.T) {
	req := `SET_PARAMETER rtsp://fe80::461e:a1ff:fece:f4a9/9953613529495192746 RTSP/1.0
Content-Length: 8
Content-Type: text/parameters
CSeq: 4
DACP-ID: 19050F2FE0FD618D
Active-Remote: 84694584
User-Agent: AirPlay/267.3

volume: 3.1415
`
	r := makeTestRtspSession()
	resp, err := request(r, req)

	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode, "StatusCode")
	ha := headerAsserter{t, resp.Header}
	ha.assert("4", "Cseq")
	ha.assert("connected; type=analog", "Apple-Jack-Status")

	tc := r.raop.plc.(*testClient)
	assert.Equal(t, float32(3.1415), tc.volume)

}

func TestSetParameterProgress(t *testing.T) {
	req := `SET_PARAMETER rtsp://fe80::461e:a1ff:fece:f4a9/9953613529495192746 RTSP/1.0
Content-Length: 40
Content-Type: text/parameters
CSeq: 42
DACP-ID: 19050F2FE0FD618D
Active-Remote: 84694584
User-Agent: AirPlay/267.3

progress: 866155144/880664705/900835976
`
	r := makeTestRtspSession()
	resp, err := request(r, req)

	assert.Nil(t, err)
	assert.Equal(t, 200, resp.StatusCode, "StatusCode")
	ha := headerAsserter{t, resp.Header}
	ha.assert("42", "Cseq")
	ha.assert("connected; type=analog", "Apple-Jack-Status")

	tc := r.raop.plc.(*testClient)
	assert.Equal(t, 329014, tc.pos)
	assert.Equal(t, 786413, tc.end)

}
