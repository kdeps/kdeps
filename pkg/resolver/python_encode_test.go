package resolver_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	resolverpkg "github.com/kdeps/kdeps/pkg/resolver"
	assets "github.com/kdeps/schema/assets"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// TestEncodePythonEnv removed - function no longer exists

// TestEncodePythonOutputs removed - function no longer exists

// TestEncodePythonStderrStdoutFormatting removed - functions no longer exist

func TestPythonEncodeBasic(t *testing.T) {
	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	fs := afero.NewOsFs()
	tmpdir := t.TempDir()
	actionDir := filepath.Join(tmpdir, "action")
	pythonDir := filepath.Join(actionDir, "python")
	require.NoError(t, fs.MkdirAll(pythonDir, 0o755))

	ctx := context.Background()
	dr := &resolverpkg.DependencyResolver{
		Fs:        fs,
		Context:   ctx,
		ActionDir: actionDir,
		RequestID: "req",
		Logger:    logging.NewTestLogger(),
	}

	// Create minimal PKL content using assets workspace
	minimal := "extends \"" + workspace.GetImportPath("Python.pkl") + "\"\n\nResources {}\n"
	pklPath := filepath.Join(pythonDir, dr.RequestID+"__python_output.pkl")
	require.NoError(t, afero.WriteFile(fs, pklPath, []byte(minimal), 0o644))

	// ... existing code ...
}
