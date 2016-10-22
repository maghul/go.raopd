package raopd

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"github.com/maghul/go.alac"
)

var alacNotInitialized = errors.New("Alac has not been initialized")

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
	alac        *alac.Alac

	streamsMutex sync.Mutex
	streams      []*audioStream
}

var audiolog = getLogger("raopd.audio")

func (r *audioStreams) initAlac(rtpmap, fmtpstr string) error {
	var err error
	r.alac, err = alac.NewFromFmtp(fmtpstr)
	return err
}

func (r *audioStreams) newStream(w io.Writer, s chan bool) {
	audiolog.Debug.Println("audioStreams:newStream w=", w)
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

	r.audioBuffer = r.alac.Decode(pkt.content[12:])
	pkt.Reclaim()

	r.writeToStreams(r.audioBuffer)
}

const audioTimeout = time.Millisecond

func (r *audioStreams) writeToStreams(b []byte) {
	r.streamsMutex.Lock()
	defer r.streamsMutex.Unlock()

	for ii, as := range r.streams {
		of := as.audioWriter
		_, err := of.Write(b)
		//		audiolog.Debug.Print( "WTS: of=",of, ", n=", n, ", err = ", err )

		if err != nil {
			as.audioWriteEnd <- true
			r.streams = append(r.streams[0:ii], r.streams[ii+1:]...)
		}
	}
}

func (a *audioStreams) rtptoms(rtp int64) (int, error) {
	if a.alac == nil {
		return 0, alacNotInitialized
	}
	sampleRate := a.alac.SampleRate()
	return int((rtp * 1000) / int64(sampleRate)), nil
}
