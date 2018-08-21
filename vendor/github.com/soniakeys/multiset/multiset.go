// Copyright 2014 Sonia Keys
// License MIT: http://opensource.org/licenses/MIT

// Multiset provides map-based multisets.
//
// This package provides a rather thin layer of functionality over a Go map,
// providing some terminology and basic functionality that implements the
// mathematical concept of a multiset.
//
// A Multiset is an unordered collection of distinct elements with a
// multiplicity, or "count," associated with each element.  A Multiset is
// termed "normal" when all counts are greater than zero.  Functions and
// methods of this package generally maintain normal Multisets, given Multisets
// that are already normal.  Results on non-normal Multisets are undefined.
//
// Use caution however, if you work with large counts.  Counts are maintained
// with the Go int type and integer overflow is not detected.
//
// Multisets can be constructed and manipulated just as with any other map.
// No constructor function is provided.  Use Multiset{} or make(Multiset) to
// construct an empty Multiset.  Consider adding single elements with methods
// AssignCount, UnionCount, or AddElementCount, which are all written to
// maintain normal Multisets.  If you operate on a Multiset directly with
// Go assignments for example, consider calling Normalize afterward to ensure
// a normal Multiset.
//
// Formatted output of Multisets uses a custom formatter.  See method Format.
package multiset

import (
	"fmt"
	"sort"
	"strings"
)

// A Multiset is a named type for a Go map.
type Multiset map[interface{}]int

// String formats a Multiset with default formatting.
func (m Multiset) String() string {
	return fmt.Sprint(m)
}

// Format is a custom formatter, not normally called directly.
//
// Default formatting of Multisets prints elements repeated by their counts,
// with default formatting of the elements.  Formatted output of Multisets
// allows a format verb to be applied to the elements.  Width, precision,
// and flags are ignored, except for the "alternate" flag, '#'.  The alternate
// flag specifies to print elements followed by numeric counts, similar to
// the way maps are normally printed.
//
// Multisets are unordered, but Format orders the formatted output.
// Elements are formatted as strings, then sorted before final formatting
// as a multiset.
func (m Multiset) Format(f fmt.State, c rune) {
	var l []string
	fs := "%" + string(c)
	if f.Flag('#') {
		// alt fmt is element:count
		l = make([]string, len(m))
		fs += ":%d"
		i := 0
		for e, c := range m {
			l[i] = fmt.Sprintf(fs, e, c)
			i++
		}
	} else {
		// default fmt repeats elements
		l = make([]string, m.Cardinality())
		i := 0
		for e, c := range m {
			for j := 0; j < c; j++ {
				l[i] = fmt.Sprintf(fs, e)
				i++
			}
		}
	}
	sort.Strings(l)
	fmt.Fprint(f, "["+strings.Join(l, " ")+"]")
}

// AssignCount assigns the count of element e the value c.
//
// If element e is not in the Multiset, it is added.
//
// This method is safe for zero or negative values of c.  If c is zero or
// negative, e is removed from m.
func (m Multiset) AssignCount(e interface{}, c int) {
	if c <= 0 {
		delete(m, e)
		return
	}
	m[e] = c
}

// Normalize normalizes a Multiset by removing elements with zero or negative
// counts.
func (m Multiset) Normalize() {
	for e, c := range m {
		if c <= 0 {
			delete(m, e)
		}
	}
}

// UnionElement sets count of element e to the maximum of c and its
// current count.
//
// It has no effect if c is less than or equal to the current count.
// Zero and negative values are allowed for c.  In this case also there will
// be no effect on a normalized Multiset.
func (m Multiset) UnionElement(e interface{}, c int) {
	if n := m[e]; c > n {
		m[e] = c
	}
}

// Union sets the receiver m to the multiset union of m and argument m2.
//
// The count of each element of the result will be the maximum of
// the corresponding counts in m and m2.
func (m Multiset) Union(m2 Multiset) {
	for e, c := range m2 {
		m.UnionElement(e, c)
	}
}

// Union constructs a new Multiset that is the multiset union of its arguments.
//
// The count of each element of the result will be the maximum of the
// corresponding counts in the arguments.
func Union(a ...Multiset) Multiset {
	m := Multiset{}
	if len(a) == 0 {
		return m
	}
	// pick biggest to copy
	max := 0
	maxl := len(a[0])
	for i, m := range a[1:] {
		if len(m) > maxl {
			maxl = len(m)
			max = i + 1
		}
	}
	a[0], a[max] = a[max], a[0]
	for e, c := range a[0] {
		m[e] = c
	}
	// union the rest
	for _, m1 := range a[1:] {
		m.Union(m1)
	}
	return m
}

// IntersectElement sets count of element e to the minimum of c and its
// current count.
//
// Zero and negative values are allowed for c.  In this case the element will
// be removed from the Multiset.
func (m Multiset) IntersectElement(e interface{}, c int) {
	if c <= 0 {
		delete(m, e)
		return
	}
	if n := m[e]; c < n {
		m[e] = c
	}
}

// Intersect sets the receiver m to the multiset intersection of m and
// argument m2.
//
// The count of each element of the result will be the minimum of
// the corresponding counts in m and m2.  If an element of m is not a
// member of m2, it will be removed from m.
func (m Multiset) Intersect(m2 Multiset) {
	for e, c := range m {
		switch c2 := m2[e]; {
		case c2 <= 0:
			delete(m, e)
		case c2 < c:
			m[e] = c2
		}
	}
}

// Intersect constructs a new Multiset that is the multiset intersection of
// its arguments.
//
// The count of each element of the result will be the minimum of
// the corresponding counts in the arguments.  An element must be
// a member of all of the argument Multisets to appear in the result.
func Intersect(a ...Multiset) Multiset {
	m := Multiset{}
	if len(a) == 0 {
		return m
	}
	// pick smallest as minimal set
	min := 0
	minl := len(a[0])
	for i, m := range a[1:] {
		if len(m) < minl {
			minl = len(m)
			min = i + 1
		}
	}
	// TEST should this be unswapped at the end?
	a[0], a[min] = a[min], a[0]
a0:
	for e, c := range a[0] {
		// intersect the rest
		for _, m2 := range a[1:] {
			c2 := m2[e]
			if c2 <= 0 {
				continue a0
			}
			if c2 < c {
				c = c2
			}
		}
		m[e] = c
	}
	return m
}

// IntersectionCardinality returns the cardinality of the intersection of
// its arguments.
//
// It is more efficient than Intersect().Cardinality().
func IntersectionCardinality(a ...Multiset) (c int) {
	if len(a) == 0 {
		return
	}
	// pick smallest as minimal set
	min := 0
	minl := len(a[0])
	for i, m := range a[1:] {
		if len(m) < minl {
			minl = len(m)
			min = i + 1
		}
	}
	a[0], a[min] = a[min], a[0]
a0:
	for e, ce := range a[0] {
		// intersect the rest
		for _, m2 := range a[1:] {
			c2 := m2[e]
			if c2 <= 0 {
				continue a0
			}
			if c2 < ce {
				ce = c2
			}
		}
		c += ce
	}
	return
}

// Subset returns true if m1 is a multiset subset of m2.
//
// Multiset m1 ⊆ m2 if all elements of m1 also exist in m2, with elements
// in m1 having counts less than or equal to corresponding counts in m2.
func Subset(m1, m2 Multiset) bool {
	for e, c := range m1 {
		if c > m2[e] {
			return false
		}
	}
	return true
}

// Equal returs true if Multisets have the same elements with the same
// counts.
func Equal(m1, m2 Multiset) bool {
	if len(m1) != len(m2) {
		return false
	}
	for e, c := range m1 {
		if c != m2[e] {
			return false
		}
	}
	return true
}

// Cardinality returns the sum of counts over the Multiset.
func (m Multiset) Cardinality() int {
	s := 0
	for _, c := range m {
		s += c
	}
	return s
}

// AddElementCount sets count of element e to the sum of argument dc and
// its current count.
//
// Argument dc is a delta count and may be negative.  A negative dc will
// decrease the element count.  If the resulting count would be zero or
// negative, the element is removed.
func (m Multiset) AddElementCount(e interface{}, dc int) {
	m.AssignCount(e, m[e]+dc)
}

// AddElements increments the count for each element in argument list l.
func (m Multiset) AddElements(l ...interface{}) {
	for _, e := range l {
		m[e]++
	}
}

// AddCounts sets the receiver m to the multiset sum of m and its
// argument m2.
//
// The count of each element of m will be set to the the sum of the
// counts in the corresponding elements of m and m2.
func (m Multiset) AddCounts(m2 Multiset) {
	for e, c := range m2 {
		m[e] += c
	}
}

// Sum constructs a new Multiset that is the multiset sum of its arguments.
//
// The count of each element of the result will be the sum of
// the corresponding counts in the arguments.
func Sum(a ...Multiset) Multiset {
	m := Multiset{}
	if len(a) == 0 {
		return m
	}
	// pick biggest to copy
	max := 0
	maxl := len(a[0])
	for i, m := range a[1:] {
		if len(m) > maxl {
			maxl = len(m)
			max = i + 1
		}
	}
	a[0], a[max] = a[max], a[0]
	for e, c := range a[0] {
		m[e] = c
	}
	// sum the rest
	for _, m2 := range a[1:] {
		m.AddCounts(m2)
	}
	return m
}

// SubtractCounts sets the receiver m to the multiset difference of m and its
// argument m2.
//
// For each element that exists in both m and m2, the count in m will be
// decreased by the count in m2.  If the resulting count is zero or negative,
// the element is removed from m.
func (m Multiset) SubtractCounts(m2 Multiset) {
	for e, c := range m2 {
		m.AssignCount(e, m[e]-c)
	}
}

// Difference constructs a new Multiset that is the multiset difference
// m1 - m2.
//
// The result will contains elements of m1 with counts reduced by the
// corresponding counts in m2.  If the resulting count for an element is
// zero or negative, it does not exist in the result.
func Difference(m1, m2 Multiset) Multiset {
	m := Multiset{}
	for e, c := range m1 {
		c -= m2[e]
		if c > 0 {
			m[e] = c
		}
	}
	return m
}

// Scale as a method increases the counts of all elements by the factor n.
//
// If n is zero or negative, the Multiset will be cleared.  (Scale(0) is an
// efficient way of clearing a Multiset, short of abandoning it to the
// garbage collector and creating a new one.)
//
// Overflow is not detected.
func (m Multiset) Scale(n int) {
	if n <= 0 {
		for e := range m {
			delete(m, e)
		}
		return
	}
	for e := range m {
		m[e] *= n
	}
}

// Scale as a function constructs a new Multiset as a scaled copy of an
// original.
//
// The result will be a copy of m with counts scaled by n.  If n is zero
// or negative, the result is an empty multiset.
func Scale(m Multiset, n int) Multiset {
	m2 := Multiset{}
	if n <= 0 {
		return m2
	}
	for e, c := range m {
		m2[e] = n * c
	}
	return m2
}

// Contains returns true if the receiver m contains element e with count of
// at least c, for c greater than zero.
//
// If c is zero or negative the method returns true.  The interpretation in
// this case is that e is not required to have a positive count.  That is,
// that e is not even required to exist in m.
func (m Multiset) Contains(e interface{}, c int) bool {
	return c <= m[e]
}

// Mode returns the element or elements with the maximum count, and that
// maximum count.
func (m Multiset) Mode() (elements []interface{}, count int) {
	for e, c := range m {
		switch {
		case c == count:
			elements = append(elements, e)
		case c > count:
			elements = []interface{}{e}
			count = c
		}
	}
	return
}
