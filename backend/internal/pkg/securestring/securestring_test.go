package securestring

import "testing"

func TestEqual(t *testing.T) {
	if !Equal("abc", "abc") {
		t.Fatal()
	}
	if Equal("abc", "abC") {
		t.Fatal()
	}
	if Equal("a", "ab") {
		t.Fatal()
	}
}
