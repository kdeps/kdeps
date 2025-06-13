package evaluator

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

// TestEnsurePklBinaryExistsPositive adds a dummy `pkl` binary to PATH and
// asserts that EnsurePklBinaryExists succeeds.
func TestEnsurePklBinaryExistsPositive(t *testing.T) {
	logger := logging.NewTestLogger()

	tmpDir := t.TempDir()
	bin := "pkl"
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	dummy := filepath.Join(tmpDir, bin)
	// create executable shell script file
	err := os.WriteFile(dummy, []byte("#!/bin/sh\nexit 0"), 0o755)
	assert.NoError(t, err)

	// prepend to PATH so lookPath finds it
	old := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+old)

	err = EnsurePklBinaryExists(context.Background(), logger)
	assert.NoError(t, err)
}
