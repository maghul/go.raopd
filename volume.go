package raopd

// Convert 0...100 to 32..0
func dec2iosVolume(vol int) float32 {
	return -(32.0 * float32(vol)) / 100.0
}
