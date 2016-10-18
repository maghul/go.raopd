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

type Raop struct {
	rf *RaopFactory
	l  net.Listener

	plc         Client
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
	br              *BonjourRecord

	data, control, timing *rtp

	seqchan chan *rtpPacket
	rrchan  chan rerequest

	aeskey cipher.Block
	aesiv  []byte
}

type RaopFactory struct {
	i *info
}

func (r *Raop) String() string {
	return fmt.Sprint("RAOP: hw=", r.hwaddr)
}

func makeRaop(w io.Writer) *Raop {
	r := &Raop{}

	var err error
	r.hwaddr, err = net.ParseMAC("48:5D:60:7C:EE:22")
	if err != nil {
		panic(err)
	}

	return r
}

func (r *Raop) Close() {
	fmt.Println("Raop::close")
	r.br.Unpublish()
}

func (r *Raop) Port() uint16 {
	a := r.l.Addr()
	ta := a.(*net.TCPAddr)
	return (uint16)(ta.Port)
}

func (r *Raop) getBonjourData(key string) string {
	panic("NYI")
}

func MakeRaopFactory(maxClients int, keyfile io.Reader) (*RaopFactory, error) {
	rf := &RaopFactory{}
	var err error
	rf.i, err = makeInfo(keyfile)
	if err != nil {
		return nil, err
	}

	return rf, nil
}

func (rf *RaopFactory) Close() {
}

type rtspHandler struct {
	r *Raop
}

func (r *rtspHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("RTSP REQUEST: ", req)
	rw.WriteHeader(http.StatusOK)
}

// TODO: Not a session, it is an end-point, a device or something like that
func (rf *RaopFactory) MakeRaopSession(port uint16, client Client) (*Raop, error) {
	r := &Raop{}

	r.plc = client

	r.audioBuffer = make([]byte, 8192)

	r.samplingRate = 44100 // TODO: Get this from paramaters

	var err error
	r.hwaddr, err = net.ParseMAC("48:5D:60:7C:EE:22")
	if err != nil {
		panic(err)
	}

	r.rf = rf

	r.l, err = net.Listen("tcp", ":5100") // TODO: use port
	if err != nil {
		return nil, err
	}

	fmt.Println("Starting RTSP server at ", r.l.Addr())
	s := makeRtspServer(rf.i, r)
	//	r.s = &http.Server{}
	//	r.s.Handler = &rtspHandler{r}
	go s.Serve(r.l)

	r.br = makeAPBonjourRecord(r)
	r.br.Publish()

	return r, nil
}

func (r *Raop) startRtp(remote, rtpmap, fmtpstr string, aeskey []byte, aesiv []byte) {
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

func (r *Raop) teardown() {
	fmt.Println("What do I need to teardown actually?")
}

func (r *Raop) getParameter(name string) string {
	switch name {
	case "volume":
		return "0.000000"
	default:
		return ""
	}
}

func (r *Raop) getParameters(req io.Reader, resp io.Writer) {
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

func (r *Raop) rtptoms(rtp int64) int {
	return int((rtp * 1000) / r.samplingRate)
}

func (r *Raop) setProgress(start, current, end int64) {
	position := r.rtptoms(current - start)
	duration := r.rtptoms(end - start)

	r.plc.SetProgress(position, duration)
}

func (r *Raop) handleAudioPacket(pkt *rtpPacket) {
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
