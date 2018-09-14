//go:generate go run datagen/region/genreg.go

package langreg

// IsValidRegionCode returns true if s is a valid ISO1366-1_alpa-2 region code.
func IsValidRegionCode(s string) bool {
	_, err := RegionCodeInfo(s)
	if err != nil {
		return false
	}
	return true
}

// RegionName returns the English name of the ISO1366-1_alpa-2 region code s.
func RegionName(s string) (string, error) {
	name, err := RegionCodeInfo(s)
	return name, err
}
