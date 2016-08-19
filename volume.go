package raopd

import (
	"emh/logger"
)

var volumelog = logger.GetLogger("raopd.volume")

// Handle volume changes.

// Convert 0...100 to 32..0
func dec2iosVolume(vol float32) float32 {
	return -(32.0 * float32(vol)) / 100.0
}

func ios2decVolume(vol float32) float32 {
	if vol < -32 {
		return 0
	}
	if vol > 0 {
		return 100
	}
	return 100 + (vol*100)/32
}

type volumeHandler struct {
	absoluteModeChan  chan bool
	deviceVolumeChan  chan float32
	serviceVolumeChan chan float32
	deviceVolume      float32
}

func (v *volumeHandler) VolumeMode(absolute bool) {
	v.absoluteModeChan <- absolute
}

func (v *volumeHandler) SetDeviceVolume(vol float32) {
	v.serviceVolumeChan <- vol
}

func (v *volumeHandler) DeviceVolume() float32 {
	return v.deviceVolume
}

func (v *volumeHandler) SetServiceVolume(vol float32) {
	v.deviceVolumeChan <- vol
}

func newVolumeHandler(info *SinkInfo, setServiceVolume func(volume float32), send func(cmd string)) *volumeHandler {
	v := &volumeHandler{}
	v.absoluteModeChan = make(chan bool)
	v.serviceVolumeChan = make(chan float32, 8)
	v.deviceVolumeChan = make(chan float32, 8)

	v.startVolumeHandler(info, setServiceVolume, send)
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

func (v *volumeHandler) startVolumeHandler(info *SinkInfo, setServiceVolume func(volume float32), send func(cmd string)) {

	serviceVolume := float32(0)
	targetVolume := float32(0)
	ref := info.Name

	volChange := func(up bool) {
		if up {
			send("volumeup")
		} else {
			send("volumedown")
		}
	}

	mode := "Initial"

	go func() {
		absoluteMode := true
		v.deviceVolume = <-v.deviceVolumeChan
		serviceVolume = ios2decVolume(v.deviceVolume)
		volumelog.Debug.Println(ref, ": serviceVolume=", serviceVolume)
		for {
		mode:
			for {
				if absoluteMode {

					// Absolute volume mode. Try to match the volume on the iDevice to the volume
					// on the service by pushing volume up and down
					mode = ":Absolute:Normal: "
					volumelog.Debug.Println(ref, mode, " Starting")
				normal:
					for {
						select {
						case dVolume := <-v.deviceVolumeChan:
							v.deviceVolume = dVolume
							serviceVolume = ios2decVolume(dVolume)
							volumelog.Debug.Println(ref, mode, "deviceVolume=", dVolume, ", serviceVolume=", serviceVolume)
							setServiceVolume(serviceVolume)

						case targetVolume = <-v.serviceVolumeChan:
							volumelog.Debug.Println(ref, mode, "targetVolume=", targetVolume)
							volChange(targetVolume > serviceVolume)
							break normal

						case absoluteMode = <-v.absoluteModeChan:
							volumelog.Debug.Println(ref, mode, "switching mode: absoluteMode=", absoluteMode)
							break mode
						}
					}

					mode = ":Absolute:Recover: "
					volumelog.Debug.Println(ref, mode, " Starting")
				finder:
					for {
						select {
						case dVolume := <-v.deviceVolumeChan:
							v.deviceVolume = dVolume
							newVolume := ios2decVolume(dVolume)
							if between(newVolume, targetVolume, serviceVolume) {
								serviceVolume = newVolume
								setServiceVolume(serviceVolume)
								break finder
							}

							volumelog.Debug.Println(ref, mode, "deviceVolume=", dVolume, ", targetVolume=", targetVolume, ", newVolume=", newVolume, ", serviceVolume=", serviceVolume)
							serviceVolume = newVolume
							volChange(targetVolume > serviceVolume)

						case targetVolume = <-v.serviceVolumeChan:
							volumelog.Debug.Println(ref, mode, "targetVolume=", targetVolume)
							volChange(targetVolume > serviceVolume)

						case absoluteMode = <-v.absoluteModeChan:
							volumelog.Debug.Println(ref, mode, "switching mode: absoluteMode=", absoluteMode)
							break mode
						}
					}
				} else {
					// Relative volume mode: Send up and down volume to the service while
					// keeping the iDevice volume at the center.
					if serviceVolume > 55 && serviceVolume < 45 {
						volumelog.Debug.Println(ref, ": ------ KICK!")
						volChange(50 > serviceVolume)
					}

					mode = ":Relative:Normal: "
					volumelog.Debug.Println(ref, mode, " Starting")
				normal_r:
					for {
						select {
						case dVolume := <-v.deviceVolumeChan:
							v.deviceVolume = dVolume
							newVolume := ios2decVolume(dVolume)
							//volumelog.Debug.Println(ref, ": nr: deviceVolume=", v.deviceVolume, ", targetVolume=", 50, ", newVolume=", newVolume, ", serviceVolume=", serviceVolume )
							volumelog.Debug.Println(ref, mode, "deviceVolume=", dVolume, ", targetVolume=", 50, ", newVolume=", newVolume, ", serviceVolume=", serviceVolume)

							if newVolume > serviceVolume && newVolume > 50 {
								volumelog.Debug.Println(ref, mode, "VOLUME UP")
								setServiceVolume(1000)
							}

							if newVolume < serviceVolume && newVolume < 50 {
								volumelog.Debug.Println(ref, mode, "VOLUME DOWN")
								setServiceVolume(-1000)
							}

							volumelog.Debug.Println(ref, mode, "vol change...")
							serviceVolume = newVolume
							volChange(50 > serviceVolume)
							break normal_r

						case <-v.serviceVolumeChan:
							volumelog.Debug.Println(ref, mode, "targetVolume=", targetVolume, "   IGNORED")
							// Volume changes from service is not interesting

						case absoluteMode = <-v.absoluteModeChan:
							volumelog.Debug.Println(ref, mode, "switching mode: absoluteMode=", absoluteMode)
							break mode
						}
					}

					mode = ":Relative:Bounce: "
					volumelog.Debug.Println(ref, mode, " Starting")
				bounce_r:
					for {
						select {
						case dVolume := <-v.deviceVolumeChan:
							v.deviceVolume = dVolume
							newVolume := ios2decVolume(dVolume)
							volumelog.Debug.Println(ref, mode, "deviceVolume=", v.deviceVolume, ", targetVolume=", 50, ", newVolume=", newVolume, ", serviceVolume=", serviceVolume)
							if between(newVolume, 50, serviceVolume) && newVolume < 55 && newVolume > 45 {
								serviceVolume = newVolume
								break bounce_r
							} else {
								if newVolume > serviceVolume && newVolume > 50 {
									volumelog.Debug.Println(ref, mode, "VOLUME UP")
									setServiceVolume(1000)
								}

								if newVolume < serviceVolume && newVolume < 50 {
									volumelog.Debug.Println(ref, mode, "VOLUME DOWN")
									setServiceVolume(-1000)
								}

								volChange(50 > serviceVolume)
							}
							serviceVolume = newVolume

						case <-v.serviceVolumeChan:
							volumelog.Debug.Println(ref, mode, "targetVolume=", targetVolume, "   IGNORED")
							// Volume changes from service is not interesting

						case absoluteMode = <-v.absoluteModeChan:
							volumelog.Debug.Println(ref, mode, "switching mode: absoluteMode=", absoluteMode)
							break mode
						}
					}

				}
			}
		}
	}()

}
