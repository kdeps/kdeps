package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestCopyDirSimple(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")

	// create nested dirs & files
	files := []string{
		filepath.Join(src, "a.txt"),
		filepath.Join(src, "sub", "b.txt"),
		filepath.Join(src, "sub", "sub2", "c.txt"),
	}
	for _, f := range files {
		_ = fs.MkdirAll(filepath.Dir(f), 0o755)
		_ = afero.WriteFile(fs, f, []byte("x"), 0o644)
	}

	if err := CopyDir(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyDir error: %v", err)
	}

	// ensure all files exist in dst
	for _, f := range files {
		rel, _ := filepath.Rel(src, f)
		if ok, _ := afero.Exists(fs, filepath.Join(dst, rel)); !ok {
			t.Fatalf("file not copied: %s", rel)
		}
	}
}
