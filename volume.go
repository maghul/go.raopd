package raopd

import (
	"fmt"
	"time"
)

var volumelog = getLogger("raopd.volume")
var volumetracelog bool

const volumespan = 1.5 // +/- 5%

// Handle volume changes.

type volumeHandler struct {
	absoluteModeChan  chan bool
	deviceVolumeChan  chan float32
	serviceVolumeChan chan float32
	deviceVolume      float32

	poke bool

	info *SinkInfo
	tl   tracelog
}

func (v *volumeHandler) VolumeMode(absolute bool) {
	if v.tl.traceing {
		v.tl.trace("CALL: VolumeMode: absolute=", absolute)
	}
	v.absoluteModeChan <- absolute
}

func (v *volumeHandler) SetDeviceVolume(vol float32) {
	if v.tl.traceing {
		v.tl.trace("CALL: SetDeviceVolume to vol=", vol)
	}
	v.serviceVolumeChan <- vol
}

func (v *volumeHandler) DeviceVolume() float32 {
	return v.deviceVolume
}

func (v *volumeHandler) SetServiceVolume(vol float32) {
	if v.tl.traceing {
		v.tl.trace("CALL: SetServiceVolume to vol=", vol)
	}
	v.deviceVolumeChan <- vol
}

func newVolumeHandler(info *SinkInfo, setServiceVolume func(volume float32), send func(cmd string) error) *volumeHandler {
	v := &volumeHandler{}
	v.absoluteModeChan = make(chan bool)
	v.serviceVolumeChan = make(chan float32, 8)
	v.deviceVolumeChan = make(chan float32, 8)

	v.startVolumeHandler(info, setServiceVolume, send)
	v.info = info
	if volumetracelog {
		v.tl.initTraceLog(v.info.Name, "volumetrace", true)
	}
	return v
}

func between(a, b, c float32) bool {
	// Return true of a<=b<=c or a>=b>=c
	if a > c {
		return a >= b && b >= c
	} else {
		return a <= b && b <= c
	}
}

func inCenter(a float32) bool {
	return a > -15-volumespan && a < -15+volumespan
}

func (v *volumeHandler) checkTrace() {
	if v.tl.traceing == volumetracelog {
		return
	}

	if v.tl.traceing {
		v.tl.closeTraceLog()
	} else {
		// Open a new trace
		v.tl.initTraceLog(v.info.Name, "volumetrace", true)
	}
}

func (v *volumeHandler) startVolumeHandler(info *SinkInfo, setServiceVolume func(volume float32), send func(cmd string) error) {

	serviceVolume := float32(0)
	targetVolume := float32(0)

	volChange := func(target, current float32) {
		v.tl.trace("poke=", v.poke)
		if !v.poke {
			dv := fmt.Sprintf("setproperty?dmcp.device-volume=%4.6f", target)
			v.tl.trace("SEND: ", dv)
			err := send(dv)
			if err != nil {
				v.poke = true
			} else {
				return
			}
		}
		delta := current - target
		switch {
		case delta > 0:
			v.tl.trace("SEND: volumedown")
			send("volumedown")
		case delta < -0:
			v.tl.trace("SEND: volumeup")
			send("volumeup")
		default:
			// Do nothing
		}
	}

	mode := "Initial"

	go func() {
		absoluteMode := true
	init: // Wait for device volume but listen to mode changes.
		for {
			select {
			case v.deviceVolume = <-v.deviceVolumeChan:
				serviceVolume = v.deviceVolume
				v.tl.trace(mode, "deviceVolume=", v.deviceVolume, " -->  serviceVolume=", serviceVolume)
				break init
			case absoluteMode = <-v.absoluteModeChan:
				v.tl.trace("INIT switching mode: absoluteMode=", absoluteMode)
			}
		}
		if absoluteMode {
			setServiceVolume(serviceVolume)
		}
		for {
		mode:
			for {
				if absoluteMode {

					// Absolute volume mode. Try to match the volume on the iDevice to the volume
					// on the service by pushing volume up and down
					mode = ":Absolute:Normal: "
					v.tl.trace(mode, " Starting")
				normal:
					for {
						v.checkTrace()
						select {
						case dVolume := <-v.deviceVolumeChan:
							v.deviceVolume = dVolume
							serviceVolume = dVolume
							v.tl.trace(mode, "deviceVolume=", v.deviceVolume, " -->  serviceVolume=", serviceVolume)
							setServiceVolume(serviceVolume)

						case targetVolume = <-v.serviceVolumeChan:
							v.tl.trace(mode, "targetVolume=", targetVolume)
							volChange(targetVolume, serviceVolume)
							break normal

						case absoluteMode = <-v.absoluteModeChan:
							v.tl.trace(mode, "switching mode: absoluteMode=", absoluteMode)
							break mode
						}
					}

					mode = ":Absolute:Recover: "
					v.tl.trace(mode, " Starting")
				finder:
					for {
						v.checkTrace()
						select {
						case dVolume := <-v.deviceVolumeChan:
							v.deviceVolume = dVolume
							newVolume := dVolume
							v.tl.trace(mode, "deviceVolume=", v.deviceVolume, " -->  newVolume=", newVolume)
							if between(newVolume, targetVolume, serviceVolume) {
								v.tl.trace(mode, "STOP [ newVolume=", newVolume, ", targetVolume=", targetVolume, ", serviceVolume=", serviceVolume, "]")
								serviceVolume = newVolume
								v.tl.trace(mode, "serviceVolume=", serviceVolume)
								setServiceVolume(serviceVolume)
								break finder
							}

							v.tl.trace(mode, "CONTINUE [ newVolume=", newVolume, ", targetVolume=", targetVolume, ", serviceVolume=", serviceVolume, "]")
							serviceVolume = newVolume
							volChange(targetVolume, serviceVolume)

						case targetVolume = <-v.serviceVolumeChan:
							v.tl.trace(mode, "targetVolume=", targetVolume)
							volChange(targetVolume, serviceVolume)

						case absoluteMode = <-v.absoluteModeChan:
							v.tl.trace(mode, "switching mode: absoluteMode=", absoluteMode)
							break mode
						}
					}
				} else {
					// Relative volume mode: Send up and down volume to the service while
					// keeping the iDevice volume at the center.
					mode = ":Relative:Normal: "
					v.tl.trace(mode, " Starting")
					if !inCenter(serviceVolume) {
						v.tl.trace(mode, "BOUNCE   deviceVolume=", v.deviceVolume, ", serviceVolume=", serviceVolume)
						volChange(-15, serviceVolume)
					}
					mode = ":Relative: "
					v.tl.trace(mode, " Starting")
					centered := time.Now()
					for {
						v.checkTrace()
						select {
						case dVolume := <-v.deviceVolumeChan:
							v.deviceVolume = dVolume
							newVolume := dVolume
							v.tl.trace(mode, "deviceVolume=", v.deviceVolume, " -->  newVolume=", newVolume)
							if between(newVolume, -15, serviceVolume) && inCenter(newVolume) {
								v.tl.trace(mode, "STOP [ newVolume=", newVolume, ", targetVolume=", -15, ", serviceVolume=", serviceVolume, "], inCenter=", inCenter(newVolume))
								serviceVolume = newVolume
								centered = time.Now()
							} else {
								guardTime := time.Now().Sub(centered)
								if guardTime > 100*time.Millisecond {
									// Don't send volume up and down unless the volume knob
									// centered more than 100ms ago, we do get stray volume
									// calls after pushing the button around.
									if newVolume > serviceVolume && newVolume > -15 {
										v.tl.trace(mode, "Send Service Volume UP")
										setServiceVolume(1000)
									}

									if newVolume < serviceVolume && newVolume < -15 {
										v.tl.trace(mode, "Send Service Volume Down DOWN")
										setServiceVolume(-1000)
									}
								}
								serviceVolume = newVolume
								volChange(-15, serviceVolume)
							}

						case <-v.serviceVolumeChan:
							v.tl.trace(mode, "targetVolume=", targetVolume, "   IGNORED")
							// Volume changes from service is not interesting

						case absoluteMode = <-v.absoluteModeChan:
							v.tl.trace(mode, "switching mode: absoluteMode=", absoluteMode)
							break mode
						}
					}

				}
			}
		}
	}()

}
