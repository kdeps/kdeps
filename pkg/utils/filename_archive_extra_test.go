package utils

import (
	"testing"
)

func TestGenerateResourceIDFilenameAdditional(t *testing.T) {
	cases := []struct {
		input string
		reqID string
		want  string
	}{
		{"@foo/bar:baz", "req", "req_foo_bar_baz"},
		{"hello/world", "id", "idhello_world"},
		{"simple", "", "simple"},
	}

	for _, c := range cases {
		got := GenerateResourceIDFilename(c.input, c.reqID)
		if got != c.want {
			t.Fatalf("GenerateResourceIDFilename(%q,%q) = %q; want %q", c.input, c.reqID, got, c.want)
		}
	}
}

func TestSanitizeArchivePathAdditional(t *testing.T) {
	base := "/safe/root"

	// Good path
	if _, err := SanitizeArchivePath(base, "folder/file.txt"); err != nil {
		t.Fatalf("unexpected error for safe path: %v", err)
	}

	// Attempt path traversal should error
	if _, err := SanitizeArchivePath(base, "../evil.txt"); err == nil {
		t.Fatalf("expected error for tainted path")
	}
}
