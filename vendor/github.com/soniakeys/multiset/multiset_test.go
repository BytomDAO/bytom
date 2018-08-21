// Copyright 2014 Sonia Keys
// License MIT: http://opensource.org/licenses/MIT

package multiset_test

import (
	"fmt"

	"github.com/soniakeys/multiset"
)

func ExampleMultiset() {
	// Construct from a map literal.
	m := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m)
	// Output:
	// [a a b]
}

func ExampleMultiset_String() {
	m := multiset.Multiset{"a": 2, "b": 1}
	s := m.String()
	fmt.Println(s)
	// Output:
	// [a a b]
}

func ExampleMultiset_Format() {
	m := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m)         // Default format.
	fmt.Printf("%q\n", m)  // Specified verb.
	fmt.Printf("%#v\n", m) // Alternate formatting prints counts.
	// Output:
	// [a a b]
	// ["a" "a" "b"]
	// [a:2 b:1]
}

func ExampleMultiset_AssignCount() {
	m := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m)
	m.AssignCount("a", 0)
	m.AssignCount("b", 3)
	m.AssignCount("c", 1)
	fmt.Println(m)
	// Output:
	// [a a b]
	// [b b b c]
}

func ExampleMultiset_Normalize() {
	// A map with a negative value.
	g := map[interface{}]int{"a": 2, "b": -1}
	fmt.Println(g["a"], g["b"])
	// Convert and normalize.
	m := multiset.Multiset(g)
	m.Normalize()
	fmt.Println(m)
	// Output:
	// 2 -1
	// [a a]
}

func ExampleMultiset_UnionElement() {
	m := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m)
	m.UnionElement("a", 0)
	m.UnionElement("b", 3)
	m.UnionElement("c", 1)
	fmt.Println(m)
	// Output:
	// [a a b]
	// [a a b b b c]
}

func ExampleMultiset_Union() {
	m := multiset.Multiset{"a": 2, "b": 1}
	m2 := multiset.Multiset{"b": 3, "c": 1}
	fmt.Println(m)
	fmt.Println(m2)
	m.Union(m2)
	fmt.Println(m)
	// Output:
	// [a a b]
	// [b b b c]
	// [a a b b b c]
}

func ExampleUnion() {
	m1 := multiset.Multiset{"a": 2, "b": 1}
	m2 := multiset.Multiset{}
	m3 := multiset.Multiset{"b": 3, "c": 1}
	fmt.Println(m1, "∪", m2, "∪", m3)
	fmt.Println(multiset.Union(m1, m2, m3))
	// Output:
	// [a a b] ∪ [] ∪ [b b b c]
	// [a a b b b c]
}

func ExampleMultiset_IntersectElement() {
	m := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m)
	m.IntersectElement("a", 1)
	m.IntersectElement("c", 1)
	fmt.Println(m)
	// Output:
	// [a a b]
	// [a b]
}

func ExampleMultiset_Intersect() {
	m := multiset.Multiset{"a": 2, "b": 1}
	m2 := multiset.Multiset{"b": 3, "c": 1}
	fmt.Println(m)
	fmt.Println(m2)
	m.Intersect(m2)
	fmt.Println(m)
	// Output:
	// [a a b]
	// [b b b c]
	// [b]
}

func ExampleIntersect() {
	m1 := multiset.Multiset{"a": 2, "b": 1}
	m2 := multiset.Multiset{"b": 2}
	m3 := multiset.Multiset{"b": 3, "c": 1}
	fmt.Println(m1, "∩", m2, "∩", m3)
	fmt.Println(multiset.Intersect(m1, m2, m3))
	// Output:
	// [a a b] ∩ [b b] ∩ [b b b c]
	// [b]
}

func ExampleIntersectionCardinality() {
	m1 := multiset.Multiset{"a": 2, "b": 1}
	m2 := multiset.Multiset{"b": 2}
	m3 := multiset.Multiset{"b": 3, "c": 1}
	fmt.Println(m1, "∩", m2, "∩", m3)
	i := multiset.Intersect(m1, m2, m3)
	fmt.Println(i, i.Cardinality())
	fmt.Println(multiset.IntersectionCardinality(m1, m2, m3))
	// Output:
	// [a a b] ∩ [b b] ∩ [b b b c]
	// [b] 1
	// 1
}

func ExampleSubset() {
	m1 := multiset.Multiset{"a": 2}
	m2 := multiset.Multiset{"a": 2, "b": 1}
	m3 := multiset.Multiset{"b": 3, "c": 1}
	fmt.Println(m1, "⊆", m1, multiset.Subset(m1, m1))
	fmt.Println(m1, "⊆", m2, multiset.Subset(m1, m2))
	fmt.Println(m2, "⊆", m3, multiset.Subset(m2, m3))
	// Output:
	// [a a] ⊆ [a a] true
	// [a a] ⊆ [a a b] true
	// [a a b] ⊆ [b b b c] false
}

func ExampleEqual() {
	m1 := multiset.Multiset{"a": 2}
	m2 := multiset.Multiset{"a": 2, "b": 1}
	m3 := multiset.Multiset{"b": 1, "a": 2}
	fmt.Println(m1, "==", m2, multiset.Equal(m1, m2))
	fmt.Println(m2, "==", m3, multiset.Equal(m2, m3))
	// Output:
	// [a a] == [a a b] false
	// [a a b] == [a a b] true
}

func ExampleMultiset_Cardinality() {
	m1 := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m1, m1.Cardinality())
	// Output:
	// [a a b] 3
}

func ExampleMultiset_AddElementCount() {
	m1 := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m1)
	m1.AddElementCount("a", 1)
	fmt.Println(m1)
	m1.AddElementCount("b", -2)
	fmt.Println(m1)
	// Output:
	// [a a b]
	// [a a a b]
	// [a a a]
}

func ExampleMultiset_AddCounts() {
	m := multiset.Multiset{"a": 2, "b": 1}
	m2 := multiset.Multiset{"b": 3, "c": 1}
	fmt.Println(m)
	fmt.Println(m2)
	m.AddCounts(m2)
	fmt.Println(m)
	// Output:
	// [a a b]
	// [b b b c]
	// [a a b b b b c]
}

func ExampleMultiset_AddElements() {
	m := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m)
	m.AddElements("b", "c")
	fmt.Println(m)
	// Output:
	// [a a b]
	// [a a b b c]
}

func ExampleSum() {
	m1 := multiset.Multiset{"a": 2, "b": 1}
	m2 := multiset.Multiset{"b": 1}
	m3 := multiset.Multiset{"b": 2, "c": 1}
	fmt.Println(m1, "+", m2, "+", m3)
	fmt.Println(multiset.Sum(m1, m2, m3))
	// Output:
	// [a a b] + [b] + [b b c]
	// [a a b b b b c]
}

func ExampleMultiset_SubtractCount() {
	m := multiset.Multiset{"a": 2, "b": 1}
	m2 := multiset.Multiset{"a": 1, "b": 2, "c": 1}
	fmt.Println(m)
	fmt.Println(m2)
	m.SubtractCounts(m2)
	fmt.Println(m)
	// Output:
	// [a a b]
	// [a b b c]
	// [a]
}

func ExampleDifference() {
	m1 := multiset.Multiset{"a": 2, "b": 1}
	m2 := multiset.Multiset{"a": 1, "b": 2, "c": 1}
	fmt.Println(m1, "-", m2)
	fmt.Println(multiset.Difference(m1, m2))
	// Output:
	// [a a b] - [a b b c]
	// [a]
}

func ExampleMultiset_Scale() {
	m := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m)
	m.Scale(3)
	fmt.Println(m)
	m.Scale(0)
	fmt.Println(m)
	// Output:
	// [a a b]
	// [a a a a a a b b b]
	// []
}

func ExampleScale() {
	m := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m)
	fmt.Println(multiset.Scale(m, 3))
	fmt.Println(multiset.Scale(m, 0))
	// Output:
	// [a a b]
	// [a a a a a a b b b]
	// []
}

func ExampleMultiset_Contains() {
	m := multiset.Multiset{"a": 2, "b": 1}
	fmt.Println(m)
	fmt.Println(m.Contains("a", 1))
	fmt.Println(m.Contains("c", 0))
	// Output:
	// [a a b]
	// true
	// true
}

func ExampleMultiset_Mode() {
	m := multiset.Multiset{"a": 3, "b": 1, "c": 3}
	fmt.Println(m)
	e, c := m.Mode()
	// (mode result order is indeterminate
	// convert mode to a multiset just to leverage sorted output.)
	d := multiset.Multiset{}
	d.AddElements(e...)
	fmt.Println(d, c)
	// Output:
	// [a a a b c c c]
	// [a c] 3
}
