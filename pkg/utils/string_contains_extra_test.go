package utils

import "testing"

func TestContainsStringInsensitiveExtra(t *testing.T) {
	slice := []string{"Hello", "World"}
	if !ContainsStringInsensitive(slice, "hello") {
		t.Fatalf("expected to find 'hello' case-insensitively")
	}
	if ContainsStringInsensitive(slice, "missing") {
		t.Fatalf("did not expect to find 'missing'")
	}
}

func TestPointerHelpers(t *testing.T) {
	s := "test"
	if *StringPtr(s) != "test" {
		t.Fatalf("StringPtr failed")
	}
	b := false
	if *BoolPtr(b) != false {
		t.Fatalf("BoolPtr failed")
	}
}
