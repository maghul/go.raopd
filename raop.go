package raopd

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

type raop struct {
	rf *ServiceRegistry
	l  net.Listener

	plc Service
	audioStreams

	hwaddr net.HardwareAddr

	vol *volumeHandler
	// TODO: This should be considered session data. There is a 1-1 relationship
	//       between an Raop instance and a session instance but cover different
	//       functionality
	clientUserAgent string
	br              *bonjourRecord

	dacp                  *dacp
	rtsp                  *rtspServer
	data, control, timing *rtp
	remote                net.IP

	seqchan   chan *rtpPacket
	rrchan    chan rerequest
	sequencer sequencer
}

func (r *raop) String() string {
	return fmt.Sprint("RAOP: hw=", r.hwaddr)
}

func (r *raop) startRtspProcess() (err error) {
	si := r.plc.ServiceInfo()
	r.hwaddr = si.HardwareAddress

	r.l, err = net.Listen("tcp", fmt.Sprintf(":%d", si.Port))
	if err != nil {
		return
	}

	fmt.Println("Starting RTSP server at ", r.l.Addr())
	s := makeRtspServer(r.rf.i, r)
	r.rtsp = s
	go s.Serve(r.l)

	r.startRaopProcess()

	return
}

func (r *raop) startRaopProcess() {
	r.dacp = newDacp()

	r.vol = newVolumeHandler(r.plc.SetVolume, r.dacp.tx)

	r.audioBuffer = make([]byte, 8192)

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

func (r *raop) startRtp(controlAddr, timingAddr *net.UDPAddr) (err error) {
	fmt.Println("startRtp...")
	if r.seqchan == nil {
		r.seqchan = make(chan *rtpPacket, 256)
		r.rrchan = make(chan rerequest, 128)
		r.sequencer = startSequencer(r.seqchan, r.handleAudioPacket, r.rrchan)
	}
	if r.control == nil {
		r.control, err = startRtp(r.getControlHandler, controlAddr)
		if err == nil {
			r.data, err = startRtp(r.getDataHandler, nil)
			if err == nil {
				r.timing, err = startRtp(r.getTimingHandler, timingAddr)
			}
		}
	}
	if err != nil {
		r.sequencer.stop()
		fmt.Println("Failed to start RTP:", err)
	}
	return
}

func (r *raop) setRemote(remote string) error {
	var err error
	r.remote, err = cToIP(remote)
	return err
}

func (r *raop) teardown() {
	r.plc.Close()
	r.sequencer.stop()
	r.data.Close()
	r.control.Close()
	r.timing.Close()
	r.control = nil
}

func (r *raop) close() {
	r.rtsp.Close()
	r.data.Close()
	r.control.Close()
	r.timing.Close()
	r.control = nil
}

func (r *raop) getParameter(name string) string {
	fmt.Println("------------------ getParameter: <", name, ">")
	switch name {
	case "volume":
		return fmt.Sprintf("%f", r.vol.DeviceVolume())
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

func (r *raop) setProgress(start, current, end int64) {
	position := r.rtptoms(current - start)
	duration := r.rtptoms(end - start)

	r.plc.SetProgress(position, duration)
}
