package raopd

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// AirplaySinkCollection is is used to register AirplayerSink instances
type SinkCollection struct {
	i       *info // actually only crypto stuff
	sources map[Sink]*Source
	m       sync.Mutex
}

// AirplaySource can be used by the audio output to
// emit commands back to the source.
type Source struct {
	raop
}

/*
NewAirplaySinkCollection creates a new collection to register services.
*/
func NewSinkCollection(keyfilename string) (*SinkCollection, error) {
	rf := &SinkCollection{}
	var err error
	rf.i, err = makeInfo(keyfilename)
	if err != nil {
		return nil, err
	}
	rf.sources = make(map[Sink]*Source)

	return rf, nil
}

/*
Close all services created in this registry
*/
func (sc *SinkCollection) Close() {
	var s map[Sink]*Source

	{
		sc.m.Lock()
		defer sc.m.Unlock()
		s = sc.sources
		sc.sources = make(map[Sink]*Source)
	}

	for sink, source := range s {
		zeroconf.Unpublish(source.br)
		sink.Closed()
	}

	zeroconf.zeroconfCleanUp() // TODO: will cleanup too much
}

/*
Register will create and publish a new Airplay output.
*/
func (sc *SinkCollection) Register(sink Sink) (*Source, error) {
	var source *Source
	{
		sc.m.Lock()
		defer sc.m.Unlock()

		if source, ok := sc.sources[sink]; ok {
			return source, nil
		}
		source = &Source{}
		sc.sources[sink] = source
	}

	source.raop.sink = sink
	source.raop.acs = sc
	source.raop.startRtspProcess()

	source.raop.br = makeAPBonjourRecord(&source.raop)
	err := zeroconf.Publish(source.raop.br)
	if err != nil {
		source.raop.close()
		return nil, err
	}

	return source, nil
}

/*
Register will create and publish a new Airplay output.
*/
func (sc *SinkCollection) Unregister(sink Sink) {
	var source *Source
	{
		sc.m.Lock()
		defer sc.m.Unlock()

		source = sc.sources[sink]
		delete(sc.sources, sink)
	}

	netlog.Debug.Println("Service Close")
	zeroconf.Unpublish(source.br)
	sink.Closed()
}

/*
Returns the port of the RAOP server. This is useful if the service
was created with an ephemeral port, i.e. port==0.
*/
func (source *Source) Port() uint16 {
	return source.port()
}

/*
String returns a brief description of the service. Useful for logging
and debugging.
*/
func (source *Source) String() string {
	return source.raop.String()
}

// Command will send a DACP command to the connected source.
//  beginff			begin fast forward
//
//  beginrew			begin rewind
//
//  mutetoggle		toggle mute status
//
//  nextitem			play next item in playlist
//
//  previtem			play previous item in playlist
//
//  pause			pause playback
//
//  playpause		toggle between play and pauses
//
//  play			start playback
//
//  stop			stop playback
//
//  playresume		play after fast forward or rewind
//
//  shuffle_songs		shuffle playlist
//
//  volumedown		turn audio volume down (*)
//
//  volumeup			turn audio volume up (*)
//
// (*) Do not use these, this will only affect the display
// and the volume sent to AirplaySink. Volume control is maintained
// by setting the displayed volume using the Volume and VolumeMode
// functions of AirplaySource.
func (source *Source) Command(cmd string) {
	source.dacp.tx(cmd)
}

// Volume will set the displayed volume on the source device if it is
// in AbsoluteMode. The vol parameters can be in range -30 to 0
func (source *Source) Volume(vol float32) {
	source.raop.vol.SetDeviceVolume(vol)
}

// VolumeMode will set the volume mode of the source device. If absolute
// is false then the volume changes sent to SetVolume of the AirplaySink
// interface will be relative, i.e. up and down volume commands. This is done
// by setting the volume slider on the source device to middle position and check
// if it is dragged right (volume up) or dragged left (volume down). Also the
// volume up/down buttons on the source device will send volume up/down commands
// to SetVolume.
//
// If absolute is true then the volume sent to SetVolume will be in the range 0..100.
// and the volume slider will reflect the volume send using the Volume function
// in AirplaySource.
func (source *Source) VolumeMode(absolute bool) {
	source.raop.vol.VolumeMode(absolute)
}

// NewAudioStream will start a new audio output stream for the source.
// Only raw PCM with two channel
// 16-bit depth at 44100 samples/second is currently supported.
// The stream is sent as w and s is a channel to indicate that
// the stream has been closed by the receiver or source.
func (source *Source) NewAudioStream(w io.Writer, s chan bool) {
	source.raop.newStream(w, s)
}
