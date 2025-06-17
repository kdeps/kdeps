package docker

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

// TestCleanOldFiles ensures that the helper deletes the ResponseTargetFile when it exists
// and returns nil when the file is absent. Both branches of the conditional are exercised.
func TestCleanOldFilesMemFS(t *testing.T) {
	mem := afero.NewMemMapFs()
	dr := &resolver.DependencyResolver{
		Fs:                 mem,
		ResponseTargetFile: "/tmp/response.json",
		Logger:             logging.NewTestLogger(),
		Context:            context.Background(),
	}

	// Branch 1: File exists and should be removed without error.
	if err := afero.WriteFile(mem, dr.ResponseTargetFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to seed response file: %v", err)
	}
	if err := cleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles returned error for existing file: %v", err)
	}
	if exists, _ := afero.Exists(mem, dr.ResponseTargetFile); exists {
		t.Fatalf("expected response file to be removed")
	}

	// Branch 2: File does not exist – function should still return nil (no error).
	if err := cleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles returned error when file absent: %v", err)
	}
}

// TestCleanOldFilesRemoveError exercises the branch where RemoveAll returns an
// error. It uses a read-only filesystem wrapper so the delete fails without
// depending on OS-specific permissions.
func TestCleanOldFilesRemoveError(t *testing.T) {
	mem := afero.NewMemMapFs()
	target := "/tmp/response.json"
	if err := afero.WriteFile(mem, target, []byte("data"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	dr := &resolver.DependencyResolver{
		Fs:                 afero.NewReadOnlyFs(mem), // makes RemoveAll fail
		ResponseTargetFile: target,
		Logger:             logging.NewTestLogger(),
		Context:            context.Background(),
	}

	if err := cleanOldFiles(dr); err == nil {
		t.Fatalf("expected error from RemoveAll, got nil")
	}
}
