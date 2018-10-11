package langreg

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type langRegTestInfo struct {
	code            string
	expectedIsValid bool
	testDescription string
}

var langRegTests = []langRegTestInfo{
	{"en_GB", true, "valid code en_GB"},
	{"en_ZA", true, "valid code en_ZA"},
	{"zz_GB", false, "invalid language, valid region"},
	{"en_UK", false, "valid language, invalid region"},
	{"EN_GB", false, "uppercase language"},
	{"en_gb", false, "lowercase region"},
	{"en-GB", false, "no underscore separator"},
	{"en_GBB", false, "too long"},
	{"en_G", false, "too short"},
}

func TestIsValidLangRegCode(t *testing.T) {
	for _, tt := range langRegTests {
		Convey("Given a "+tt.testDescription, t, func() {
			So(IsValidLangRegCode(tt.code), ShouldEqual, tt.expectedIsValid)
		})
	}
}

func BenchmarkIsValidLangRegCode(b *testing.B) {
	for n := 0; n < b.N; n++ {
		isValid := IsValidLangRegCode("zu_ZW")
		if !isValid {
			b.Error("invalid language code")
		}
	}
}
