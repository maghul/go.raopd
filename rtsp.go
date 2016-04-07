package raopd

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
)

/* It would have been nice to use the HTTP package since it is 99%
 * http. But the remaining 1% is important and impossible to tack on
 * to the http server.
 */
type rtspServer struct {
	i    *info
	raop *raop
}

type rtspSession struct {
	i    *info
	raop *raop
	c    net.Conn
}

type rtspResponseWriter struct {
	h       http.Header
	wr      *bufio.Writer
	written bool
}

func newRtspResponseWriter(wr *bufio.Writer) *rtspResponseWriter {
	rw := &rtspResponseWriter{}
	rw.wr = wr
	rw.written = false
	rw.h = make(map[string][]string)
	return rw
}

func (t *rtspResponseWriter) Header() http.Header {
	return t.h
}

func (t *rtspResponseWriter) Write(d []byte) (int, error) {
	if !t.written {
		t.WriteHeader(200)
	}
	return t.wr.Write(d)
}

func (t *rtspResponseWriter) finishResponse() {
	if !t.written {
		t.WriteHeader(200)
	}
	t.wr.Flush()
}

func makeRtspServer(i *info, raop *raop) *rtspServer {
	r := &rtspServer{}
	r.i = i
	r.raop = raop
	return r
}

func (r *rtspServer) Close() {
	// TODO: Close the RTSP server and possible the RTP servers as well.
}

func statusMap(code int) string {
	return "OK"
}

func (t *rtspResponseWriter) WriteHeader(statusCode int) {
	if !t.written {
		t.wr.WriteString(fmt.Sprintf("RTSP/1.0 %d %s\r\n", statusCode, statusMap(statusCode)))
		for key, values := range t.h {
			for _, value := range values {
				t.wr.WriteString(key)
				t.wr.WriteString(": ")
				t.wr.WriteString(value)
				t.wr.WriteString("\r\n")
			}
		}
		t.wr.WriteString("\r\n")
	}
}

func (r *rtspServer) Serve(l net.Listener) error {
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		rs := &rtspSession{r.i, r.raop, c}
		go rs.runRtspServerSession(c)
	}

}

func (rs *rtspSession) handleChallenge(challenge string) (string, error) {
	addr := rs.LocalAddr()
	tcpaddr := addr.(*net.TCPAddr)
	ip := tcpaddr.IP
	return rs.i.rsaKeySign(challenge, ip, rs.raop.hwaddr)
}

func readToString(r io.Reader) string {
	b := bytes.NewBufferString("")
	io.Copy(b, r)
	return b.String()
}

func scanf(s, f string, t ...interface{}) bool {
	n, _ := fmt.Sscanf(s, f, t...)
	return n > 0
}

func (rs *rtspSession) handle(rw http.ResponseWriter, req *http.Request) {
	h := rw.Header()
	h.Add("Cseq", req.Header["Cseq"][0])
	h.Add("Apple-Jack-Status", "connected; type=analog")

	switch req.Method {
	case "OPTIONS":
		h.Add("Public", "ANNOUNCE, SETUP, RECORD, PAUSE, FLUSH, TEARDOWN, OPTIONS, GET_PARAMETER, SET_PARAMETER")
		challenge := req.Header.Get("Apple-Challenge")
		if challenge != "" {
			response, err := rs.handleChallenge(challenge)
			if err != nil {
				rw.WriteHeader(400)
				return
			}

			h.Add("Apple-Response", response)
		}

	case "ANNOUNCE":

		var rdr io.Reader
		cl, ok := req.Header["Content-Length"]
		if ok {
			clen, err := strconv.ParseInt(cl[0], 10, 32)
			if err != nil {
				rw.WriteHeader(400)
				return
			}
			fmt.Println("Content-Length=", clen)
			rdr = io.LimitReader(req.Body, clen)
		} else {
			rdr = req.Body
		}
		sdp := makeSdpRecords(rdr)
		remote := sdp["c"]
		rtpmap := sdp["a=rtpmap"]
		fmtp := sdp["a=fmtp"]

		aeskey, err := rs.i.rsaKeyDecrypt(sdp["a=rsaaeskey"])
		if err != nil {
			rtsplog.Info.Println("Could not decrypt AES key, key=", sdp["a=rsaaeskey"], ", :", err)
			rw.WriteHeader(400)
			return
		}
		aesiv, err := rs.i.rsaKeyParseIv(sdp["a=aesiv"])
		if err != nil {
			rtsplog.Info.Println("Could not parse IV, aesiv=", sdp["a=aesiv"], ", :", err)
			rw.WriteHeader(400)
			return
		}

		fmt.Println("AESKEY=", aeskey)
		fmt.Println("AESIV=", aesiv)
		rs.raop.aeskey, err = aes.NewCipher(aeskey)
		rs.raop.aesiv = aesiv

		if err != nil {
			rw.WriteHeader(400)
			return
		}
		rtsplog.Debug.Println("RAOP AESKEY=", rs.raop.aeskey)
		rs.raop.startRtp()
		rs.raop.initAlac(remote, rtpmap, fmtp)

	case "SETUP":
		raop := rs.raop

		raop.dacpID = req.Header["Dacp-Id"][0]
		raop.activeRemote = req.Header["Active-Remote"][0]
		raop.clientUserAgent = req.Header["User-Agent"][0]

		session := "DEADBEEF"
		transport := fmt.Sprintf("RTP/AVP/UDP;unicast;mode=record;timing_port=%d;events;control_port=%d;server_port=%d\nSession: %s",
			raop.timing.Port(), raop.control.Port(), raop.data.Port(), session)
		h.Add("Transport", transport)

	case "GET_PARAMETER":
		raop := rs.raop

		raop.dacpID = req.Header["Dacp-Id"][0]
		raop.activeRemote = req.Header["Active-Remote"][0]
		raop.clientUserAgent = req.Header["User-Agent"][0]

		content := bytes.NewBufferString("")
		raop.getParameters(req.Body, content)
		h.Add("Content-Type", "text/parameters")
		h.Add("Content-Length", fmt.Sprintf("%d", content.Len()))
		io.Copy(rw, content)

	case "SET_PARAMETER":
		contentType := req.Header["Content-Type"][0]
		switch contentType {
		case "text/parameters":
			s := readToString(req.Body)
			s = strings.Trim(s, " \r\n")
			var vol float32
			var start, current, end int64
			switch {
			case scanf(s, "volume: %f", &vol):
				//				fmt.Println("volume:", vol)
				rs.raop.plc.SetVolume(vol)
			case scanf(s, "progress: %d/%d/%d", &start, &current, &end):
				//				fmt.Println("progress:", start, current, end)
				rs.raop.setProgress(start, current, end)
			}
		case "image/jpeg":
			rs.raop.plc.LoadCoverArt(contentType, req.Body)
		case "application/x-dmap-tagged":
			rs.raop.plc.LoadMetadata(req.Body)
		default:
			rtsplog.Info().Println("SET_PARAMETER: Unknown Content-Type=", contentType)

		}
	case "RECORD":
		rs.raop.plc.Play()
	case "PAUSE":
		fmt.Println("....................... PAUSE?")
		rs.raop.plc.Pause()
	case "FLUSH":
	case "TEARDOWN":
		rs.raop.teardown()
	default:
		rw.WriteHeader(404)
	}
}

func (rs *rtspSession) LocalAddr() net.Addr {
	return rs.c.LocalAddr()
}

func (rs *rtspSession) runRtspServerSession(c net.Conn) {
	//	rd := bufio.NewReader(c)
	wr := bufio.NewWriter(c)

	for {
		req, err := rs.readRequest(ioutil.NopCloser(c))
		if err != nil {
			fmt.Println("Ending RTSP session:", err)
			return
		}
		rw := newRtspResponseWriter(wr)
		rs.handle(rw, req)
		rw.finishResponse()
	}
}

func parseRTSPVersion(s string) (proto string, major int, minor int, err error) {
	parts := strings.SplitN(s, "/", 2)
	proto = parts[0]
	parts = strings.SplitN(parts[1], ".", 2)
	if major, err = strconv.Atoi(parts[0]); err != nil {
		return
	}
	if minor, err = strconv.Atoi(parts[0]); err != nil {
		return
	}
	return
}

func (rs *rtspSession) readRequest(rd io.ReadCloser) (req *http.Request, err error) {
	req = &http.Request{}
	req.Header = make(map[string][]string)

	var s string

	brd := bufio.NewReader(rd)
	tp := textproto.NewReader(brd)

	if s, err = tp.ReadLine(); err != nil {
		fmt.Println("READ REQUEST:H:", err)
		return
	}
	//	fmt.Println("RX:H:", s)
	parts := strings.SplitN(s, " ", 3)
	req.Method = parts[0]
	if req.URL, err = url.Parse(parts[1]); err != nil {
		return
	}

	req.Proto, req.ProtoMajor, req.ProtoMinor, err = parseRTSPVersion(parts[2])
	if err != nil {
		return
	}

	// read headers
	for {
		if s, err = tp.ReadLine(); err != nil {
			fmt.Println("READ REQUEST:C:", err)
			return
		}
		//		fmt.Println("RX:C:", s)
		if s = strings.TrimRight(s, "\r\n"); s == "" {
			break
		}

		parts := strings.SplitN(s, ":", 2)
		req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
	}

	req.ContentLength, _ = strconv.ParseInt(req.Header.Get("Content-Length"), 10, 0)
	if req.ContentLength > 0 {
		req.Body = ioutil.NopCloser(io.LimitReader(brd, req.ContentLength))
	} else {
		req.Body = ioutil.NopCloser(brd)
	}

	return req, nil
}
