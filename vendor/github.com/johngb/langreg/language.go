//go:generate go run datagen/language/genlang.go

package langreg

// IsValidLanguageCode returns true if s is a valid ISO 639-1 language code
func IsValidLanguageCode(s string) bool {
	_, _, err := LangCodeInfo(s)
	if err != nil {
		return false
	}
	return true
}

// LangEnglishName returns the English name(s) corresponding to the language code
// s.  If there are multiple names, they are separated by a `;`.
func LangEnglishName(s string) (string, error) {
	en, _, err := LangCodeInfo(s)
	return en, err
}

// LangNativeName returns the native name(s) corresponding to the language code s
// in the native script(s).  If there are multiple names, they are separated
// by a `;`.
func LangNativeName(s string) (string, error) {
	_, nat, err := LangCodeInfo(s)
	return nat, err
}
