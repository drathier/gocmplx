package main

import (
	"testing"
)

type tc struct {
	hay    string
	needle string
	expect []int
}

func tcase(tc tc, t *testing.T) {
	got := indexAll([]byte(tc.hay), []byte(tc.needle))
	if len(got) != len(tc.expect) {
		t.Fatal("len differs:", tc, got, "!=", tc.expect)
	}
	for i := range got {
		if got[i] != tc.expect[i] {
			t.Fatal(i, tc, got[i], "!=", tc.expect[i])
		}
	}
}

func TestIndexAll(t *testing.T) {
	tcase(tc{hay: "abcdef", needle: "c", expect: []int{2}}, t)
	tcase(tc{hay: "abcdefabc", needle: "c", expect: []int{2, 8}}, t)
	tcase(tc{hay: "abcdefabc", needle: "z", expect: []int{}}, t)
	tcase(tc{hay: "", needle: "something", expect: []int{}}, t)
}
