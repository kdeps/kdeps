package archiver

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestPerformCopy_SuccessAndError(t *testing.T) {
	fs := afero.NewMemMapFs()
	// success path
	afero.WriteFile(fs, "/src.txt", []byte("hello"), 0o644)

	if err := performCopy(fs, "/src.txt", "/dst.txt"); err != nil {
		t.Fatalf("performCopy success returned error: %v", err)
	}

	data, _ := afero.ReadFile(fs, "/dst.txt")
	if string(data) != "hello" {
		t.Fatalf("content mismatch: %s", data)
	}

	// error path: source missing
	if err := performCopy(fs, "/missing.txt", "/dst2.txt"); err == nil {
		t.Fatalf("expected error when source missing")
	}
}

func TestCopyDir_Basic(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create source directory with nested content
	_ = fs.MkdirAll("/src/sub", 0o755)
	afero.WriteFile(fs, "/src/file1.txt", []byte("one"), 0o644)
	afero.WriteFile(fs, "/src/sub/file2.txt", []byte("two"), 0o644)

	if err := CopyDir(fs, ctx, "/src", "/dst", logger); err != nil {
		t.Fatalf("CopyDir error: %v", err)
	}

	// Verify copied files
	for _, p := range []struct{ path, expect string }{
		{"/dst/file1.txt", "one"},
		{"/dst/sub/file2.txt", "two"},
	} {
		data, err := afero.ReadFile(fs, p.path)
		if err != nil {
			t.Fatalf("missing copied file %s: %v", p.path, err)
		}
		if string(data) != p.expect {
			t.Fatalf("file %s content mismatch", p.path)
		}
	}
}
