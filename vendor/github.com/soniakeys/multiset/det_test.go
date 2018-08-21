// Copyright 2014 Sonia Keys
// License MIT: http://opensource.org/licenses/MIT

package multiset_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/soniakeys/multiset"
)

// This file contains deterministic tests that sort results to deal with
// the random ordering of go maps.
//
// Also a goal is for this file provide 100% test coverage.

// eq orders a formatted result and validates it against an expected result.
// expected must be ordered.  (It works with these simple test cases anyway.)
// (well, sort of.  it defeats the line number reporting of go test.  that's
// probably fine for these little tests.  could be improved though.)
func eq(t *testing.T, result, expected string) {
	last := len(result) - 1
	s := strings.Fields(result[1:last])
	sort.Strings(s)
	ordered := result[:1] + strings.Join(s, " ") + result[last:]
	if ordered != expected {
		t.Fatal("Expected", expected, "got", ordered, `
(before ordering`, result, ")")
	}
}

func eqs(t *testing.T, result multiset.Multiset, expected string) {
	eq(t, result.String(), expected)
}

func TestUnion(t *testing.T) {
	eqs(t, multiset.Union(), "[]")
	m1 := multiset.Multiset{}
	m2 := multiset.Multiset{"a": 2, "b": 1}
	m3 := multiset.Multiset{"b": 3, "c": 1}
	eqs(t, multiset.Union(m1, m2, m3), "[a a b b b c]")
}

func TestMultiset_IntersectElement(t *testing.T) {
	m := multiset.Multiset{"a": 2, "b": 1}
	m.IntersectElement("a", 1)
	m.IntersectElement("b", 0)
	eqs(t, m, "[a]")
}

func TestMultiset_Intersect(t *testing.T) {
	m := multiset.Multiset{"a": 2, "b": 3}
	m2 := multiset.Multiset{"b": 2, "c": 1}
	m.Intersect(m2)
	eqs(t, m, "[b b]")
}

func TestIntersect(t *testing.T) {
	eqs(t, multiset.Intersect(), "[]")
	m1 := multiset.Multiset{"a": 2, "b": 1, "c": 2}
	m2 := multiset.Multiset{"a": 2, "b": 3}
	m3 := multiset.Multiset{"b": 3, "c": 1}
	eqs(t, multiset.Intersect(m1, m2, m3), "[b]")
}

func TestIntersectionCardinality(t *testing.T) {
	want := 0
	if got := multiset.IntersectionCardinality(); got != want {
		t.Fatal("got", got, "want", want)
	}
	m1 := multiset.Multiset{"a": 2, "b": 1, "c": 2}
	m2 := multiset.Multiset{"a": 2, "b": 3}
	m3 := multiset.Multiset{"b": 3, "c": 1}
	want = 1
	if got := multiset.IntersectionCardinality(m1, m2, m3); got != want {
		t.Fatal("got", got, "want", want)
	}
}

func TestEqual(t *testing.T) {
	m1 := multiset.Multiset{"a": 2}
	m2 := multiset.Multiset{"a": 2, "b": 1}
	m3 := multiset.Multiset{"a": 2, "c": 1}
	m4 := multiset.Multiset{"b": 1, "a": 2}
	if multiset.Equal(m1, m2) != false {
		t.Fatal(m1, m2)
	}
	if multiset.Equal(m2, m3) != false {
		t.Fatal(m2, m3)
	}
	if multiset.Equal(m2, m4) != true {
		t.Fatal(m2, m3)
	}
}

func TestSum(t *testing.T) {
	eqs(t, multiset.Sum(), "[]")
	m1 := multiset.Multiset{"b": 1}
	m2 := multiset.Multiset{"a": 2, "b": 1}
	m3 := multiset.Multiset{"b": 2, "c": 1}
	eqs(t, multiset.Sum(m1, m2, m3), "[a a b b b b c]")
}
