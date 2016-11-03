package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/maghul/go.raopd"
	"github.com/maghul/go.slf"
	"github.com/mesilliac/pulse-simple"
)

var airplayers *raopd.SinkCollection

type player struct {
	info   *raopd.SinkInfo
	source *raopd.Source
	device *pulse.Stream
}

func initLogging() {
	logg := slf.GetLogger("raopd")
	fmt.Println("Enabling Logger: ", logg.Name())
	logg.SetOutputLogger(os.Stdout)
	logg.SetLevel(slf.Info)
}

func main() {
	initLogging()

	keyfilename := "/tmp/airport.key"
	airplayers, err := raopd.NewSinkCollection(keyfilename)
	if err != nil {
		panic(err)
	}

	sc := make(chan os.Signal)
	signal.Notify(sc)

	si := &raopd.SinkInfo{
		Name:            "My Player",
		HardwareAddress: net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		Port:            0,
	}

	ss := pulse.SampleSpec{pulse.SAMPLE_S16LE, 44100, 2}
	stream, err := pulse.Playback("my app", "my stream", &ss)
	if err != nil {
		panic(err)
	}

	sp := &player{
		info:   si,
		source: nil,
		device: stream,
	}

	sp.source, err = airplayers.Register(sp)
	if err != nil {
		panic(err)
	}

	<-sc

	airplayers.Close()
	time.Sleep(100 * time.Millisecond)
	os.Exit(0)
}

func (p *player) Info() *raopd.SinkInfo {
	return p.info
}

func (p *player) Connected(name string) {
}

func (p *player) SetCoverArt(mimetype string, content []byte) {
}

func (p *player) SetMetadata(content string) {
}

func (p *player) SetVolume(volume float32) {
}

func (p *player) SetProgress(pos, length int) {
}

func (p *player) Play() {
	ctx := context.Background()
	p.source.NewAudioStream(ctx, p.device)
}

func (p *player) Pause() {

}

func (p *player) Stopped() {

}

func (p *player) Closed() {
}
