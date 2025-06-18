package archiver_test

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/spf13/afero"
)

func TestGetFileMD5(t *testing.T) {
	memFs := afero.NewMemMapFs()

	// Write a simple file.
	content := "hello world"
	if err := afero.WriteFile(memFs, "/tmp.txt", []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Compute MD5 with full length.
	md5Full, err := archiver.GetFileMD5(memFs, "/tmp.txt", 32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(md5Full) != 32 {
		t.Fatalf("expected 32-char MD5, got %d", len(md5Full))
	}

	// Same call with truncated length should return prefix.
	md5Short, err := archiver.GetFileMD5(memFs, "/tmp.txt", 8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if md5Short != md5Full[:8] {
		t.Fatalf("truncated hash mismatch: %s vs %s", md5Short, md5Full[:8])
	}

	// Non-existent file should raise an error.
	if _, err := archiver.GetFileMD5(memFs, "/does-not-exist", 8); err == nil {
		t.Fatalf("expected error for missing file")
	}
}
