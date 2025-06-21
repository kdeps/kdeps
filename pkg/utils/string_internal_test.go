package utils_test

import (
	"testing"

	. "github.com/kdeps/kdeps/pkg/utils"
)

func TestStringPtr_Internal(t *testing.T) {
	p := StringPtr("foo")
	if p == nil || *p != "foo" {
		t.Errorf("StringPtr failed: %v", p)
	}
}

func TestBoolPtr_Internal(t *testing.T) {
	p := BoolPtr(true)
	if p == nil || *p != true {
		t.Errorf("BoolPtr failed: %v", p)
	}
}

func TestContainsString_Internal(t *testing.T) {
	s := []string{"a", "b"}
	if !ContainsString(s, "a") {
		t.Error("ContainsString failed for present value")
	}
	if ContainsString(s, "c") {
		t.Error("ContainsString returned true for absent value")
	}
}

func TestContainsStringInsensitive_Internal(t *testing.T) {
	s := []string{"Foo", "Bar"}
	if !ContainsStringInsensitive(s, "foo") {
		t.Error("ContainsStringInsensitive failed for present value")
	}
	if ContainsStringInsensitive(s, "baz") {
		t.Error("ContainsStringInsensitive returned true for absent value")
	}
}

func TestSafeDerefString_Internal(t *testing.T) {
	var p *string
	if SafeDerefString(p) != "" {
		t.Error("SafeDerefString failed for nil pointer")
	}
	v := "x"
	if SafeDerefString(&v) != "x" {
		t.Error("SafeDerefString failed for non-nil pointer")
	}
}

func TestSafeDerefBool_Internal(t *testing.T) {
	var p *bool
	if SafeDerefBool(p) != false {
		t.Error("SafeDerefBool failed for nil pointer")
	}
	v := true
	if SafeDerefBool(&v) != true {
		t.Error("SafeDerefBool failed for non-nil pointer")
	}
}

func TestSafeDerefSlice_Internal(t *testing.T) {
	var p *[]int
	if out := SafeDerefSlice(p); len(out) != 0 {
		t.Error("SafeDerefSlice failed for nil pointer")
	}
	s := []int{1, 2}
	if out := SafeDerefSlice(&s); len(out) != 2 || out[0] != 1 {
		t.Error("SafeDerefSlice failed for non-nil pointer")
	}
}

func TestSafeDerefMap_Internal(t *testing.T) {
	var p *map[string]int
	if out := SafeDerefMap(p); len(out) != 0 {
		t.Error("SafeDerefMap failed for nil pointer")
	}
	m := map[string]int{"a": 1}
	if out := SafeDerefMap(&m); out["a"] != 1 {
		t.Error("SafeDerefMap failed for non-nil pointer")
	}
}
