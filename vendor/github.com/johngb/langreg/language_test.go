package langreg

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type langTestInfo struct {
	code            string
	expectedEn      string
	expectedNat     string
	errExpected     bool
	testDescription string
}

var langTests = []langTestInfo{
	{"en", "English", "English", false, "code with ascii character set"},
	{"EN", "", "", true, "code with uppercase ascii characters"},
	{"iu", "Inuktitut", "ᐃᓄᒃᑎᑐᑦ", false, "code with non-ascii Unicode character set"},
	{"  en ", "", "", true, "code with leading and trailng space"},
	{"\nen ", "", "", true, "code with leading and trailing whitespace"},
	{"zzz", "", "", true, "code that is too long"},
	{"z", "", "", true, "code that is too short"},
	{"zz", "", "", true, "code that is invalid"},
	{"*z", "", "", true, "code with invalid characters"},
	{"73", "", "", true, "code with numbers"},
}

func TestLanguageCodeInfo(t *testing.T) {
	for _, tt := range langTests {
		Convey("Given a "+tt.testDescription, t, func() {
			actualEn, actualNat, actualErr := LangCodeInfo(tt.code)
			So(actualEn, ShouldEqual, tt.expectedEn)
			So(actualNat, ShouldEqual, tt.expectedNat)
			if !tt.errExpected {
				So(actualErr, ShouldBeNil)
			} else {
				So(actualErr, ShouldNotBeNil)
			}
		})
	}
}

func TestIsLanguageCode(t *testing.T) {
	for _, tt := range langTests {
		Convey("Given a "+tt.testDescription, t, func() {
			actualIsValid := IsValidLanguageCode(tt.code)
			So(!actualIsValid, ShouldEqual, tt.errExpected)
		})
	}
}

func TestLangEnglishName(t *testing.T) {
	for _, tt := range langTests {
		Convey("Given a "+tt.testDescription, t, func() {
			actualEnName, actualErr := LangEnglishName(tt.code)
			So(actualEnName, ShouldEqual, tt.expectedEn)
			if !tt.errExpected {
				So(actualErr, ShouldBeNil)
			} else {
				So(actualErr, ShouldNotBeNil)
			}
		})
	}
}

func TestLangNativeName(t *testing.T) {
	for _, tt := range langTests {
		Convey("Given a "+tt.testDescription, t, func() {
			actualNatName, actualErr := LangNativeName(tt.code)
			So(actualNatName, ShouldEqual, tt.expectedNat)
			if !tt.errExpected {
				So(actualErr, ShouldBeNil)
			} else {
				So(actualErr, ShouldNotBeNil)
			}
		})
	}
}

func BenchmarkLangCodeInfo(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _, err := LangCodeInfo("zu")
		if err != nil {
			b.Error(err.Error())
		}
	}
}

func BenchmarkIsValidLanguageCode(b *testing.B) {
	for n := 0; n < b.N; n++ {
		isValid := IsValidLanguageCode("zu")
		if !isValid {
			b.Error("invalid language code")
		}
	}
}

func BenchmarkLangEnglishName(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, err := LangEnglishName("zu")
		if err != nil {
			b.Error(err.Error())
		}
	}
}

func BenchmarkLangNativeName(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, err := LangEnglishName("zu")
		if err != nil {
			b.Error(err.Error())
		}
	}
}
