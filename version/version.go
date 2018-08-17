package version

import (
	"fmt"
	"strconv"
	"strings"
)

var (
	// The full version string
	Version = "1.0.4"
	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
	Checked   = false
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
}

type VerNum struct {
	Major  uint64
	Middle uint64
	Minor  uint64
}

func Parse(verStr string) (*VerNum, error) {
	spl := strings.Split(verStr, ".")
	if len(spl) != 3 {
		return nil, fmt.Errorf("Invalid version format %v", verStr)
	}
	spl[2] = strings.Split(spl[2], "-")[0]

	vMajor, err0 := strconv.ParseUint(spl[0], 10, 64)
	vMiddle, err1 := strconv.ParseUint(spl[1], 10, 64)
	vMinor, err2 := strconv.ParseUint(spl[2], 10, 64)
	if err0 != nil || err1 != nil || err2 != nil {
		return nil, fmt.Errorf("Invalid version format %v", verStr)
	}

	return &VerNum{
		Major:  vMajor,
		Middle: vMiddle,
		Minor:  vMinor,
	}, nil
}

func (v1 *VerNum) first2() (float64, error) {
	fmt.Sprintf("%d.%d", Major, Middle)
}

func (v1 *VerNum) last2() (float64, error) {
	fmt.Sprintf("%d.%d", Major, Middle)
}

func (v1 *VerNum) GreaterThan(v2 *VerNum) bool {

	return true
}
