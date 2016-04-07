package raopd

import (
	"io"
	"net"
)

/*
This structure is passed to
*/
type ServiceInfo struct {
	// If the service supports absolute volume, 0 - 100
	SupportsAbsoluteVolume bool

	// If the service supports relative volume. UP=1000 and DOWN=-1000
	SupportsRelativeVolume bool

	// If the sink wants coverart. See SetCoverArt in AirplaySink
	SupportsCoverArt bool

	// If the client wants metadata, song info, artist and track name.
	// Should be "XML" or "JSON", if it is "" then no metadata will be supplied.
	SupportsMetaData string

	// The name of the sink
	Name string

	// The hardware address of the sink. This is used as an identifier to avoid
	// identity collision and does not need to use a real hardware address. If it is
	// set to nil the server hardware address will be used.
	HardwareAddress net.HardwareAddr

	// The port the RAOP server should start at. Set to 0 to get an ephemeral port selected at random.
	Port uint16
}

/*
This is the interface for the service implementation
*/
type Service interface {
	// Get the service info for the service.
	ServiceInfo() *ServiceInfo

	// Get a writer for the audio stream. Only raw PCM with two channel
	// 16-bit depth at 44100 samples/second is currently supported.
	AudioWriter() io.Writer
	AudioWriterErr(error)

	// SetCoverArt will set the cover art of the currently playing track.
	// May be ignored and can be disables by setting SupportsCoverArt to
	// false in AirplaySinkInfo.
	SetCoverArt(mimetype string, content []byte)

	// SetMetadata will set the metadata of the currently playing track.
	// The data is DMAP data in a JSON or XML representation. This is controlled
	// by setting SupportsMetadata in AirplaySinkInfo to "JSON" or "XML"
	SetMetadata(content string)

	// Set the volume of the output device. The volume value may be an absolute
	// value from 0 - 100, or it may be up down values using UP=1000 and DOWN=-1000
	SetVolume(volume float32)

	// Shows the progress of the track in milliseconds.
	// pos is the current position, length is the total length of the current track
	SetProgress(pos, length int)

	// Called when the stream is started.
	Play()

	// Called when the stream is paused
	Pause()

	// Called when the connection to source is terminated
	Close()
}
