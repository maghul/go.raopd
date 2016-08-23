package raopd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
)

// AirplaySinkCollection is is used to register AirplayerSink instances
type AirplaySinkCollection struct {
	i       *info // actually only crypto stuff
	sources map[AirplaySink]*AirplaySource
	m       sync.Mutex
}

// AirplaySource can be used by the audio output to
// emit commands back to the source.
type AirplaySource struct {
	raop
}

/*
NewAirplaySinkCollection creates a new collection to register services.
*/
func NewAirplaySinkCollection() (*AirplaySinkCollection, error) {
	rf := &AirplaySinkCollection{}
	var err error
	rf.i, err = makeInfo()
	if err != nil {
		return nil, err
	}
	rf.sources = make(map[AirplaySink]*AirplaySource)

	return rf, nil
}

/*
Close all services created in this registry
*/
func (rf *AirplaySinkCollection) Close() {
	var s map[AirplaySink]*AirplaySource

	{
		rf.m.Lock()
		defer rf.m.Unlock()
		s = rf.sources
		rf.sources = make(map[AirplaySink]*AirplaySource)
	}

	for sink, source := range s {
		source.br.Unpublish()
		sink.Closed()
	}
}

/*
Register will create and publish a new Airplay output.
*/
func (acs *AirplaySinkCollection) Register(sink AirplaySink) (*AirplaySource, error) {
	var source *AirplaySource
	{
		acs.m.Lock()
		defer acs.m.Unlock()

		if source, ok := acs.sources[sink]; ok {
			return source, nil
		}
		source = &AirplaySource{}
		acs.sources[sink] = source
	}

	source.raop.sink = sink
	source.raop.acs = acs
	source.raop.startRtspProcess()

	source.raop.br = makeAPBonjourRecord(&source.raop)
	err := source.raop.br.Publish()
	if err != nil {
		source.raop.close()
		return nil, err
	}

	return source, nil
}

/*
Register will create and publish a new Airplay output.
*/
func (acs *AirplaySinkCollection) Unregister(sink AirplaySink) {
	var source *AirplaySource
	{
		acs.m.Lock()
		defer acs.m.Unlock()

		source = acs.sources[sink]
		delete(acs.sources, sink)
	}

	netlog.Debug.Println("Service Close")
	source.br.Unpublish()
	sink.Closed()
}

/*
Returns the port of the RAOP server. This is useful if the service
was created with an ephemeral port, i.e. port==0.
*/
func (source *AirplaySource) Port() uint16 {
	return source.port()
}

/*
String returns a brief description of the service. Useful for logging
and debugging.
*/
func (source *AirplaySource) String() string {
	return source.raop.String()
}

// Command will send a DACP command to the connected source.
func (source *AirplaySource) Command(cmd string) {
	source.dacp.tx(cmd)
}

func (source *AirplaySource) Volume(vol string) {
	ivol, err := strconv.ParseFloat(vol, 32)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error converting volume ", vol, " to integer:", err)
	}
	source.raop.vol.SetDeviceVolume(float32(ivol))
}

// VolumeMode will set the volume mode of the source device. If absolute
// is false then the volume changes sent to SentVolume of the AirplaySink
// interface will be relative, i.e. up and down volume commands. If absolute
// is true then the volume will be in the range 0..100.
func (source *AirplaySource) VolumeMode(absolute bool) {
	source.raop.vol.VolumeMode(absolute)
}

// NewAudioStream will start a new audio output stream for the source.
// Only raw PCM with two channel
// 16-bit depth at 44100 samples/second is currently supported.
// The stream is sent as w and s is a channel to indicate that
// the stream has been closed by the receiver or source.
func (source *AirplaySource) NewAudioStream(w io.Writer, s chan bool) {
	source.raop.newStream(w, s)
}
