package compiler

import (
	"github.com/Masterminds/semver"
)

const ivyVersion string = "1.9.9"

func parseVersion(p *parser) bool {
	if peekKeyword(p) == "pragma" {
		consumeKeyword(p, "pragma")
		if peekKeyword(p) == "ivy" {
			consumeKeyword(p, "ivy")
			strliteral, newOffset := scanVersionStr(p.buf, p.pos)
			if newOffset < 0 {
				p.errorf("Invalid version character format!")
			}
			p.pos = newOffset

			//After removing the quotes is the version info
			version := strliteral[1 : len(strliteral)-1]
			if ok := checkVersion(string(version)); ok {
				return true
			}
			return false
		}
		return false
	}

	//when contract is not contain the version info, return true
	return true
}

func checkVersion(version string) bool {
	c, err := semver.NewConstraint(version)
	if err != nil {
		panic(err)
	}

	v, err := semver.NewVersion(ivyVersion)
	if err != nil {
		panic(err)
	}

	return c.Check(v)
}

func scanVersionStr(buf []byte, offset int) ([]byte, int) {
	offset = skipWsAndComments(buf, offset)
	if offset >= len(buf) || !(buf[offset] == '\'' || buf[offset] == '"') {
		return nil, -1
	}

	for i := offset + 1; i < len(buf); i++ {
		if (buf[offset] == '\'' && buf[i] == '\'') || (buf[offset] == '"' && buf[i] == '"') {
			return buf[offset : i+1], i + 1
		}
		if buf[i] == '\\' {
			i++
		}
	}

	panic(parseErr(buf, offset, "unterminated version string literal"))
}
