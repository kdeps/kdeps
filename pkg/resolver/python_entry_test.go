package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	pklPython "github.com/kdeps/schema/gen/python"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestAppendPythonEntry_CreatesResource(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewOsFs()
	tmpDir := t.TempDir()

	actionDir := filepath.Join(tmpDir, "action")
	filesDir := filepath.Join(tmpDir, "files")
	require.NoError(t, fs.MkdirAll(filepath.Join(actionDir, "python"), 0o755))
	require.NoError(t, fs.MkdirAll(filesDir, 0o755))

	requestID := "req123"
	pythonPklPath := filepath.Join(actionDir, "python", requestID+"__python_output.pkl")

	// Create minimal initial PKL file with empty resources map
	minimal := "extends \"package://schema.kdeps.com/core@" + schema.SchemaVersion(ctx) + "#/Python.pkl\"\n\nresources {}\n"
	require.NoError(t, afero.WriteFile(fs, pythonPklPath, []byte(minimal), 0o644))

	dr := &DependencyResolver{
		Fs:        fs,
		Logger:    logging.NewTestLogger(),
		Context:   ctx,
		ActionDir: actionDir,
		FilesDir:  filesDir,
		RequestID: requestID,
	}

	scriptContent := "print('hello')"
	newPy := &pklPython.ResourcePython{
		Script: scriptContent,
	}

	err := dr.AppendPythonEntry("myPython", newPy)
	require.NoError(t, err)

	// Verify the PKL file now exists and contains our resource id
	content, err := afero.ReadFile(fs, pythonPklPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "[\"myPython\"]")
}
