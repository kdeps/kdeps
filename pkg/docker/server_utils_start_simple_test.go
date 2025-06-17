package docker

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestStartOllamaServerSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := logging.NewTestLogger()

	// Call function under test; it should return immediately and not panic.
	startOllamaServer(ctx, logger)

	// Give the background goroutine a brief moment to run and fail gracefully.
	time.Sleep(10 * time.Millisecond)
}

func TestCheckDevBuildModeVariants(t *testing.T) {
	fs := afero.NewMemMapFs()
	kdepsDir := t.TempDir()
	logger := logging.NewTestLogger()

	// Case 1: file missing -> expect false
	ok, err := checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected false when file absent")
	}

	// Case 2: file present -> expect true
	cacheFile := filepath.Join(kdepsDir, "cache", "kdeps")
	_ = fs.MkdirAll(filepath.Dir(cacheFile), 0o755)
	_ = afero.WriteFile(fs, cacheFile, []byte("bin"), 0o755)

	ok, err = checkDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true when file present")
	}
}
