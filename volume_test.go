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

func TestVolumeDec2Ios(t *testing.T) {
	assert.Equal(t, float32(-30.0), dec2iosVolume(0))
	assert.Equal(t, float32(-22.5), dec2iosVolume(25))
	assert.Equal(t, float32(-15.0), dec2iosVolume(50))
	assert.Equal(t, float32(-7.5), dec2iosVolume(75))
	assert.Equal(t, float32(0.0), dec2iosVolume(100))
}

func TestVolumeIos2Dec(t *testing.T) {
	assert.Equal(t, float32(0), ios2decVolume(-40))
	assert.Equal(t, float32(0), ios2decVolume(-30))
	assert.Equal(t, float32(25), ios2decVolume(-22.5))
	assert.Equal(t, float32(50), ios2decVolume(-15))
	assert.Equal(t, float32(75), ios2decVolume(-7.5))
	assert.Equal(t, float32(100), ios2decVolume(0))
	assert.Equal(t, float32(100), ios2decVolume(10))
}

func TestVolume1(t *testing.T) {
	// Volume in normal mode, driven from service
	_, dvc, svc, resp := makeVolumeTest(true)
	dvc <- -9
	waitFor(t, "serviceVolume:70", resp)
	svc <- 60
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -10
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -11
	waitFor(t, "cmd:volumedown", resp)
	dvc <- -12
	waitFor(t, "serviceVolume:60", resp)
	doneWaiting(t, resp)
}

func TestVolume2(t *testing.T) {
	// Volume in normal mode, driven from device.
	_, dvc, _, resp := makeVolumeTest(true)
	dvc <- -10
	waitFor(t, "serviceVolume:66.66667", resp)
	dvc <- -11
	waitFor(t, "serviceVolume:63.333332", resp)
	dvc <- -12
	waitFor(t, "serviceVolume:60", resp)
	dvc <- -13
	waitFor(t, "serviceVolume:56.666668", resp)
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
	svc <- 77
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
	svc <- 77
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
	svc <- 77
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
