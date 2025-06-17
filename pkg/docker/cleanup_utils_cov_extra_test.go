package docker

import (
	"context"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestCreateFlagFileAndCleanup(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	flag1 := "/tmp/flag1"
	flag2 := "/tmp/flag2"

	// Create first flag file via helper.
	if err := CreateFlagFile(fs, ctx, flag1); err != nil {
		t.Fatalf("CreateFlagFile returned error: %v", err)
	}

	// Second call with same path should NO-OP (exists) and return nil.
	if err := CreateFlagFile(fs, ctx, flag1); err != nil {
		t.Fatalf("CreateFlagFile second call expected nil err, got %v", err)
	}

	// Manually create another flag for removal.
	if err := afero.WriteFile(fs, flag2, []byte("test"), 0o644); err != nil {
		t.Fatalf("setup write file: %v", err)
	}

	// Ensure both files exist before cleanup.
	for _, p := range []string{flag1, flag2} {
		if ok, _ := afero.Exists(fs, p); !ok {
			t.Fatalf("expected %s to exist", p)
		}
	}

	logger := logging.NewTestLogger()
	cleanupFlagFiles(fs, []string{flag1, flag2}, logger)

	// Confirm they are removed.
	for _, p := range []string{flag1, flag2} {
		if ok, _ := afero.Exists(fs, p); ok {
			t.Fatalf("expected %s to be removed by cleanupFlagFiles", p)
		}
	}

	// Verify CreateFlagFile sets timestamps (basic sanity: non-zero ModTime).
	path := "/tmp/flag3"
	if err := CreateFlagFile(fs, ctx, path); err != nil {
		t.Fatalf("CreateFlagFile: %v", err)
	}
	info, _ := fs.Stat(path)
	if info.ModTime().IsZero() || time.Since(info.ModTime()) > time.Minute {
		t.Fatalf("unexpected ModTime on created flag file: %v", info.ModTime())
	}
}
