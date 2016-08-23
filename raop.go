package raopd

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"emh/audio/alac"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

type raop struct {
	rf *ServiceRegistry
	l  net.Listener

	plc         Service
	hwaddr      net.HardwareAddr
	audioBuffer []byte
	alac        alac.AlacFile

	// TODO: This should be considered session data. There is a 1-1 relationship
	//       between an Raop instance and a session instance but cover different
	//       functionality
	dacpID          string
	activeRemote    string
	clientUserAgent string
	samplingRate    int64
	br              *bonjourRecord

	data, control, timing *rtp

	seqchan chan *rtpPacket
	rrchan  chan rerequest

	aeskey cipher.Block
	aesiv  []byte
}

func (r *raop) String() string {
	return fmt.Sprint("RAOP: hw=", r.hwaddr)
}

func (r *raop) port() uint16 {
	a := r.l.Addr()
	ta := a.(*net.TCPAddr)
	return (uint16)(ta.Port)
}

type rtspHandler struct {
	r *raop
}

func (r *rtspHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("RTSP REQUEST: ", req)
	rw.WriteHeader(http.StatusOK)
}

func (r *raop) startRtp(remote, rtpmap, fmtpstr string, aeskey []byte, aesiv []byte) {
	fmt.Println("startRtp...")
	r.seqchan = make(chan *rtpPacket, 256)
	r.rrchan = make(chan rerequest, 128)
	startSequencer(r.seqchan, r.handleAudioPacket, r.rrchan)
	if r.control == nil {
		r.control = startRtp(r.getControlHandler)
		r.data = startRtp(r.getDataHandler)
		r.timing = startRtp(r.getTimingHandler)
	}

	af := alac.NewAlacConfFromFmtp(fmtpstr)
	r.alac = alac.MakeAlacFile(af)
}

func (r *raop) teardown() {
	fmt.Println("What do I need to teardown actually?")
}

func (r *raop) getParameter(name string) string {
	switch name {
	case "volume":
		return "0.000000"
	default:
		return ""
	}
}

func (r *raop) getParameters(req io.Reader, resp io.Writer) {
	br := bufio.NewReader(req)
	bw := bufio.NewWriter(resp)

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			bw.Flush()
			return
		}
		line = strings.Trim(line, " \r\n")
		value := r.getParameter(line)
		if value != "" {
			fmt.Fprintf(bw, "%s: %s\n", line, value)
		}

	}

}

func (r *raop) rtptoms(rtp int64) int {
	return int((rtp * 1000) / r.samplingRate)
}

func (r *raop) setProgress(start, current, end int64) {
	position := r.rtptoms(current - start)
	duration := r.rtptoms(end - start)

	r.plc.SetProgress(position, duration)
}

func (r *raop) handleAudioPacket(pkt *rtpPacket) {
	fmt.Println("Received audio packet ", pkt.seqno)
	mode := cipher.NewCBCDecrypter(r.aeskey, r.aesiv)

	ciphertext := pkt.content[12:]
	l := len(ciphertext) / 16
	l *= 16
	ciphertext = ciphertext[:l]
	if len(ciphertext)%aes.BlockSize != 0 {
		panic("ciphertext is not a multiple of the block size")
	}
	mode.CryptBlocks(ciphertext, ciphertext)

	n := r.alac.Decode(pkt.content[12:], r.audioBuffer)

	of := r.plc.AudioWriter()
	of.Write(r.audioBuffer[0:n])
}
