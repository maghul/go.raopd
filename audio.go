package raopd

import (
	"crypto/aes"
	"crypto/cipher"
	"emh/audio/alac"

	"io"
	"os"
	"sync"
	"time"
)

type audioStream struct {
	audioWriter   io.Writer
	audioWriteEnd chan bool
	count         int
}

type audioStreams struct {
	audioBuffer []byte
	mode        cipher.BlockMode
	aeskey      cipher.Block
	aesiv       []byte
	alac        *alac.AlacDecoder
	alacConf    *alac.AlacConf

	streamsMutex sync.Mutex
	streams      []*audioStream
}

var audiolog = GetLogger("raopd.audio")

func (r *audioStreams) initAlac(rtpmap, fmtpstr string) {
	r.alacConf = alac.NewAlacConfFromFmtp(fmtpstr)
	r.alac = alac.NewAlacDecoder(r.alacConf)
}

func (r *audioStreams) newStream(w io.Writer, s chan bool) {
	audiolog.Debug().Println("audioStreams:newStream w=", w)
	// Sets a timeout count of 10.
	ns := &audioStream{w, s, 10}

	r.streamsMutex.Lock()
	defer r.streamsMutex.Unlock()

	r.streams = append(r.streams, ns)
}

func (r *audioStreams) handleAudioPacket(pkt *rtpPacket) {
	if pkt.sn%100 == 0 {
		audiolog.Debug.Println("Received audio packet ", pkt.sn)
	}
	r.mode = cipher.NewCBCDecrypter(r.aeskey, r.aesiv)

	ciphertext := pkt.content[12:]
	l := len(ciphertext) / 16
	l *= 16
	ciphertext = ciphertext[:l]
	if len(ciphertext)%aes.BlockSize != 0 {
		audiolog.Info.Println("ciphertext is not a multiple of the block size")
		os.Exit(0)
	}
	r.mode.CryptBlocks(ciphertext, ciphertext)

	n := r.alac.Decode(pkt.content[12:], r.audioBuffer)
	pkt.Reclaim()

	r.writeToStreams(r.audioBuffer[0:n])
}

const audioTimeout = time.Millisecond

func (r *audioStreams) writeToStreams(b []byte) {
	r.streamsMutex.Lock()
	defer r.streamsMutex.Unlock()

	for ii, as := range r.streams {
		of := as.audioWriter
		_, err := of.Write(b)
		//		audiolog.Debug().Print( "WTS: of=",of, ", n=", n, ", err = ", err )

		if err != nil {
			as.audioWriteEnd <- true
			r.streams = append(r.streams[0:ii], r.streams[ii+1:]...)
		}
	}
}

func (a *audioStreams) rtptoms(rtp int64) int {
	if a.alac == nil {
		panic("Alac has not been initialized")
	}
	sampleRate := a.alac.SampleRate()
	return int((rtp * 1000) / int64(sampleRate))
}
