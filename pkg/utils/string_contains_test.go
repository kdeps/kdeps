package utils

import "testing"

func TestStringHelpers(t *testing.T) {
    slice := []string{"apple", "Banana", "cherry"}

    if !ContainsString(slice, "Banana") {
        t.Fatalf("expected exact match present")
    }
    if ContainsString(slice, "banana") {
        t.Fatalf("ContainsString should be case sensitive")
    }
    if !ContainsStringInsensitive(slice, "banana") {
        t.Fatalf("expected case-insensitive match")
    }

    // Ptr helpers
    s := "foo"
    sptr := StringPtr(s)
    if *sptr != s {
        t.Fatalf("StringPtr failed")
    }
    b := true
    bptr := BoolPtr(b)
    if *bptr != b {
        t.Fatalf("BoolPtr failed")
    }
} 