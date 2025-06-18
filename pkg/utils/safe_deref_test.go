package utils

import "testing"

func TestSafeDerefSliceAndMap(t *testing.T) {
	// Slice
	if len(SafeDerefSlice[int](nil)) != 0 {
		t.Fatalf("expected empty slice")
	}
	origSlice := []int{1, 2}
	ptrSlice := &origSlice
	gotSlice := SafeDerefSlice[int](ptrSlice)
	if len(gotSlice) != 2 || gotSlice[0] != 1 || gotSlice[1] != 2 {
		t.Fatalf("unexpected slice result %#v", gotSlice)
	}

	// Map
	if len(SafeDerefMap[string, int](nil)) != 0 {
		t.Fatalf("expected empty map")
	}
	m := map[string]int{"a": 1}
	ptrMap := &m
	gotMap := SafeDerefMap[string, int](ptrMap)
	if gotMap["a"] != 1 {
		t.Fatalf("unexpected map value")
	}
}
