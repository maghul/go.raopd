package raopd

import (
	"io"
)

type Client interface {
	AudioWriter() io.Writer
	LoadCoverArt(mimetype string, content io.Reader)
	LoadMetadata(content io.Reader)
	SetVolume(volume float32)
	SetProgress(pos, length int)
	Play()
	Pause()
}
