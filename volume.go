package raopd

var volumelog = getLogger("raopd.volume")

const volumespan = 5 // +/- 5%

// Handle volume changes.

// Convert 0...100 to 32..0
func dec2iosVolume(vol float32) float32 {
	return -(30.0 * float32(100-vol)) / 100.0
}

func ios2decVolume(vol float32) float32 {
	if vol < -30 {
		return 0
	}
	if vol > 0 {
		return 100
	}
	return 100 + (vol*100)/30
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

func inCenter(a float32) bool {
	return a > 50-volumespan && a < 50+volumespan
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
	init: // Wait for device volume but listen to mode changes.
		for {
			select {
			case v.deviceVolume = <-v.deviceVolumeChan:
				volumelog.Debug.Println(ref, mode, "deviceVolume=", v.deviceVolume)
				serviceVolume = ios2decVolume(v.deviceVolume)
				volumelog.Debug.Println(ref, ": serviceVolume=", serviceVolume)
				break init
			case absoluteMode = <-v.absoluteModeChan:
				volumelog.Debug.Println(ref, "INIT switching mode: absoluteMode=", absoluteMode)
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
					volumelog.Debug.Println(ref, mode, " Starting")
				normal:
					for {
						select {
						case dVolume := <-v.deviceVolumeChan:
							v.deviceVolume = dVolume
							volumelog.Debug.Println(ref, mode, "deviceVolume=", v.deviceVolume)
							serviceVolume = ios2decVolume(dVolume)
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
							volumelog.Debug.Println(ref, mode, "deviceVolume=", v.deviceVolume)
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
					mode = ":Relative:Normal: "
					volumelog.Debug.Println(ref, mode, " Starting")
					if !inCenter(serviceVolume) {
						volumelog.Debug.Println(ref, mode, "BOUNCE   deviceVolume=", v.deviceVolume, ", serviceVolume=", serviceVolume)
						volChange(50 > serviceVolume)
					}
					mode = ":Relative: "
					volumelog.Debug.Println(ref, mode, " Starting")
					for {
						select {
						case dVolume := <-v.deviceVolumeChan:
							v.deviceVolume = dVolume
							newVolume := ios2decVolume(dVolume)
							volumelog.Debug.Println(ref, mode, "deviceVolume=", v.deviceVolume, ", targetVolume=", 50, ", newVolume=", newVolume, ", serviceVolume=", serviceVolume)
							if between(newVolume, 50, serviceVolume) && inCenter(newVolume) {
								serviceVolume = newVolume
							} else {
								if newVolume > serviceVolume && newVolume > 50 {
									volumelog.Debug.Println(ref, mode, "VOLUME UP")
									setServiceVolume(1000)
								}

								if newVolume < serviceVolume && newVolume < 50 {
									volumelog.Debug.Println(ref, mode, "VOLUME DOWN")
									setServiceVolume(-1000)
								}

								serviceVolume = newVolume
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
