package docker

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

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
