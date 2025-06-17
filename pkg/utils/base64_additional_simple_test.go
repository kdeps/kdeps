package utils

import (
	"encoding/base64"
	"reflect"
	"testing"
)

func TestDecodeBase64StringHelpers(t *testing.T) {
	orig := "hello world"
	encoded := base64.StdEncoding.EncodeToString([]byte(orig))

	t.Run("ValidString", func(t *testing.T) {
		out, err := DecodeBase64String(encoded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != orig {
			t.Fatalf("want %q got %q", orig, out)
		}
	})

	t.Run("InvalidString", func(t *testing.T) {
		in := "$$invalid$$"
		out, err := DecodeBase64String(in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out != in {
			t.Fatalf("want %q got %q", in, out)
		}
	})
}

func TestDecodeStringMapAndSliceHelpers(t *testing.T) {
	m := map[string]string{"a": "foo", "b": "bar"}
	for k, v := range m {
		m[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	gotMapPtr, err := DecodeStringMap(&m, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotMap := *gotMapPtr
	if !reflect.DeepEqual(gotMap, map[string]string{"a": "foo", "b": "bar"}) {
		t.Fatalf("decoded map mismatch: %v", gotMap)
	}

	sl := []string{"foo", "bar"}
	encodedSlice := []string{
		base64.StdEncoding.EncodeToString([]byte(sl[0])),
		base64.StdEncoding.EncodeToString([]byte(sl[1])),
	}
	gotSlicePtr, err := DecodeStringSlice(&encodedSlice, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	gotSlice := *gotSlicePtr
	if !reflect.DeepEqual(gotSlice, sl) {
		t.Fatalf("decoded slice mismatch: %v", gotSlice)
	}
}
