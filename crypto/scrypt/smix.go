package scrypt

// Smix create tensority cache
// Some value is fixed: r = 1, N = 1024.
func Smix(b []byte, v []uint32) {
	xy := make([]uint32, 64)
	smix(b, 1, 1024, v, xy)
}
