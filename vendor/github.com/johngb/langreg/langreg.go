// Copyright 2014 John G. Beckett. All rights reserved.
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// Package langreg is a library for validating ISO 639-1 language
// and ISO 1366-1_alpa-2 region codes.
//
// ISO 639-1 language codes are two charcters long and use only lowercase ASCII
// character a-z. E.g.:
//
// 	"en", "es", "ru"
//
// ISO 1366-1_alpa-2 region codes are two charcters long and use only uppercase
// ASCII character A-Z. E.g.:
//
// 	"US", "UK", "ZA"
//
// When combined as a composite language and region code, they are concatented
// with an underscore. E.g.:
//
// 	"en_US", "en_ZA", "fr_FR"
//
// Any codes not meeting these formatting requirement will fail validation.
package langreg

// IsValidLangRegCode returns true if the string s is a valid ISO 639-1 language
// and ISO1366-1_alpa-2 region code separated by an underscore.  E.g. "en_US".
func IsValidLangRegCode(s string) bool {

	// all valid codes are 5 characters long
	if len(s) != 5 {
		return false
	}

	// the middle (third) character must be a '_' char
	if s[2] != '_' {
		return false
	}

	// check the language code, which should be the first two characters in s
	if !IsValidLanguageCode(s[:2]) {
		return false
	}

	// check the region code, which should be the last two characters in s
	if !IsValidRegionCode(s[3:]) {
		return false
	}
	return true
}
