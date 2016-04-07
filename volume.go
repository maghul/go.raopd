package raopd

import ()

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
	deviceVolumeChan  chan float32
	serviceVolumeChan chan float32
	deviceVolume      float32
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

func newVolumeHandler(setServiceVolume func(volume float32), send func(cmd string)) *volumeHandler {
	v := &volumeHandler{}
	v.startVolumeHandler(setServiceVolume, send)
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

func (v *volumeHandler) startVolumeHandler(setServiceVolume func(volume float32), send func(cmd string)) {

	v.serviceVolumeChan = make(chan float32, 8)
	v.deviceVolumeChan = make(chan float32, 8)

	// TODO: Get this from ServiceInfo
	absoluteVolume := true

	serviceVolume := float32(0)
	targetVolume := float32(0)

	volChange := func(up bool) {
		if up {
			send("volumeup")
		} else {
			send("volumedown")
		}
	}

	go func() {
		for {
			if absoluteVolume {

				// Absolute volume mode. Try to match the volume on the iDevice to the volume
				// on the service by pushing volume up and down
			normal:
				for {
					select {
					case dVolume := <-v.deviceVolumeChan:
						v.deviceVolume = dVolume
						serviceVolume = ios2decVolume(dVolume)
						//fmt.Println( "deviceVolume=", deviceVolume, ", serviceVolume=", serviceVolume )
						setServiceVolume(serviceVolume)

					case targetVolume = <-v.serviceVolumeChan:
						//fmt.Println( "targetVolume=", targetVolume )
						volChange(targetVolume > serviceVolume)
						break normal
					}
				}

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

						//fmt.Println( "deviceVolume=", deviceVolume, ", targetVolume=", targetVolume, ", newVolume=", newVolume, ", serviceVolume=", serviceVolume )
						serviceVolume = newVolume
						volChange(targetVolume > serviceVolume)

					case targetVolume = <-v.serviceVolumeChan:
						//fmt.Println( "INTER targetVolume=", targetVolume )
						volChange(targetVolume > serviceVolume)
					}
				}
			} else {
				// Relative volume mode: Send up and down volume to the service while
				// keeping the iDevice volume at the center.
				panic("NYI")
			}
		}
	}()

}
