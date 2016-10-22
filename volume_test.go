package raopd

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func waitFor(t *testing.T, expected string, results chan string) {
	tmr := time.NewTimer(time.Millisecond)
	select {
	case r := <-results:
		assert.Equal(t, expected, r)
	case <-tmr.C:
		assert.Fail(t, fmt.Sprint("Timeout waiting for '", expected, "'"))

	}
}

func doneWaiting(t *testing.T, results chan string) {
	tmr := time.NewTimer(10 * time.Millisecond)
	select {
	case r := <-results:
		assert.Fail(t, fmt.Sprint("Got '", r, "' which wasn't expected"))
	case <-tmr.C:

	}
}

func makeVolumeTest(absoluteMode bool) (chan bool, chan float32, chan float32, chan string) {
	volumetracelog = true
	resp := make(chan string, 12)
	send := func(cmd string) {
		resp <- fmt.Sprint("cmd:", cmd)
	}
	setServiceVolume := func(volume float32) {
		resp <- fmt.Sprint("serviceVolume:", volume)
	}
	info := &SinkInfo{}
	Debug("log.info/*", 1)
	Debug("log.debug/*", 1)
	info.Name = "testvolume"
	v := newVolumeHandler(info, setServiceVolume, send)
	v.absoluteModeChan <- absoluteMode

	time.Sleep(time.Millisecond)
	return v.absoluteModeChan, v.deviceVolumeChan, v.serviceVolumeChan, resp
}

func TestVolume1(t *testing.T) {
	// Volume in normal mode, driven from service
	_, dvc, svc, resp := makeVolumeTest(true)
	dvc <- -9
	waitFor(t, "serviceVolume:-9", resp)
	svc <- -12
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -10
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -11
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -12
	waitFor(t, "serviceVolume:-12", resp)
	doneWaiting(t, resp)
}

func TestVolume2(t *testing.T) {
	// Volume in normal mode, driven from device.
	_, dvc, _, resp := makeVolumeTest(true)
	dvc <- -10
	waitFor(t, "serviceVolume:-10", resp)
	dvc <- -11
	waitFor(t, "serviceVolume:-11", resp)
	dvc <- -12
	waitFor(t, "serviceVolume:-12", resp)
	dvc <- -13
	waitFor(t, "serviceVolume:-13", resp)
	doneWaiting(t, resp)
}

func TestVolume3(t *testing.T) {
	// Volume in relative mode, driven from device
	_, dvc, _, resp := makeVolumeTest(false)
	dvc <- -9
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -10
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -11
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -12
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -13
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -14
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -15
	doneWaiting(t, resp)
}

func TestVolume3B(t *testing.T) {
	// Volume in relative mode, driven from device
	_, dvc, _, resp := makeVolumeTest(false)
	dvc <- -9.1
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -10.1
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -11.1
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -12.1
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -13.1
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -14.1
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -15.1
	doneWaiting(t, resp)

}

func TestVolume4(t *testing.T) {
	// Volume in relative mode, driven from service.
	_, dvc, svc, resp := makeVolumeTest(false)
	dvc <- -18
	svc <- -17
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -18
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -17
	waitFor(t, "cmd:volumeup", resp)
	doneWaiting(t, resp)
}

func TestVolume5(t *testing.T) {
	// Volume in relative mode, driven from device.
	_, dvc, svc, resp := makeVolumeTest(false)
	dvc <- -18
	svc <- -15
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -18
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -17
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -16
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -15

	time.Sleep(100 * time.Millisecond)
	// We should now be stable.
	dvc <- -25 // Poke down
	waitFor(t, "serviceVolume:-1000", resp)
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -24
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -23
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -22
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -21
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -20
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -19
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -18
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -17
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -16
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -15

	time.Sleep(100 * time.Millisecond)
	// And stable again.
	dvc <- -13 // Poke up
	waitFor(t, "serviceVolume:1000", resp)
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -14
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -15

	doneWaiting(t, resp)
}
func TestVolume6(t *testing.T) {
	// Volume in relative mode, driven from device.
	_, dvc, svc, resp := makeVolumeTest(false)
	dvc <- -18
	svc <- -15
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -18
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -17
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -16
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -15

	time.Sleep(100 * time.Millisecond)
	// We should now be stable.
	dvc <- -25 // Poke down
	waitFor(t, "serviceVolume:-1000", resp)
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -24
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -23
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -22
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -21
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -20
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -22 // The volume went down, this means the user dragged the volume slider down
	waitFor(t, "serviceVolume:-1000", resp)
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -21
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -20
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -19
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -18
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -17
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -16
	waitFor(t, "cmd:volumeup", resp)
	dvc <- -15

	time.Sleep(100 * time.Millisecond)
	// And stable again.
	dvc <- -13 // Poke up
	waitFor(t, "serviceVolume:1000", resp)
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -14
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -15

	doneWaiting(t, resp)
}
