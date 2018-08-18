package version

import (
	"fmt"
	"strconv"
	"strings"
)

type VerNum struct {
	major    uint64
	minor    uint64
	revision uint64
}

func parse(verStr string) (*VerNum, error) {
	spl := strings.Split(verStr, ".")
	if len(spl) != 3 {
		return nil, fmt.Errorf("Invalid version format %v", verStr)
	}
	spl[2] = strings.Split(spl[2], "-")[0]

	vMajor, err0 := strconv.ParseUint(spl[0], 10, 64)
	vMinor, err1 := strconv.ParseUint(spl[1], 10, 64)
	vRevision, err2 := strconv.ParseUint(spl[2], 10, 64)
	if err0 != nil || err1 != nil || err2 != nil {
		return nil, fmt.Errorf("Invalid version format %v", verStr)
	}

	return &VerNum{
		major:    vMajor,
		minor:    vMinor,
		revision: vRevision,
	}, nil
}

func (v *VerNum) first2() (float64, error) {
	f2Str := fmt.Sprintf("%d.%d", v.major, v.minor)
	if f2, err := strconv.ParseFloat(f2Str, 64); err != nil {
		return float64(0), err
	} else {
		return f2, nil
	}
}

func (v *VerNum) last2() (float64, error) {
	l2Str := fmt.Sprintf("%d.%d", v.minor, v.revision)
	if l2, err := strconv.ParseFloat(l2Str, 64); err != nil {
		return float64(0), err
	} else {
		return l2, nil
	}
}

func (v1 *VerNum) greaterThan(v2 *VerNum) (bool, error) {
	v1_f2, err := v1.first2()
	if err != nil {
		return false, err
	}
	v1_l2, err := v1.last2()
	if err != nil {
		return false, err
	}
	v2_f2, err := v2.first2()
	if err != nil {
		return false, err
	}
	v2_l2, err := v2.last2()
	if err != nil {
		return false, err
	}

	greaterThanV2 := (v1_f2 > v2_f2) || ((v1_f2 == v2_f2) && (v1_l2 > v2_l2))
	return greaterThanV2, nil
}
