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

func TestEncodePythonEnv(t *testing.T) {
	dr := &DependencyResolver{Logger: logging.GetLogger()}

	env := map[string]string{"A": "alpha", "B": "beta"}
	encoded := dr.encodePythonEnv(&env)
	if encoded == nil || len(*encoded) != 2 {
		t.Fatalf("expected 2 encoded entries")
	}
	if (*encoded)["A"] == "alpha" {
		t.Errorf("value A not encoded")
	}
}

func TestEncodePythonOutputs(t *testing.T) {
	dr := &DependencyResolver{}
	stderr := "some err"
	stdout := "some out"
	e1, e2 := dr.encodePythonOutputs(&stderr, &stdout)
	if *e1 == stderr || *e2 == stdout {
		t.Errorf("outputs not encoded: %s %s", *e1, *e2)
	}

	// nil pass-through
	n1, n2 := dr.encodePythonOutputs(nil, nil)
	if n1 != nil || n2 != nil {
		t.Errorf("expected nil return for nil inputs")
	}
}

func TestEncodePythonStderrStdoutFormatting(t *testing.T) {
	dr := &DependencyResolver{}
	msg := "line1\nline2"
	got := dr.encodePythonStderr(&msg)
	if len(got) == 0 || got[0] != ' ' {
		t.Errorf("unexpected format: %s", got)
	}
	got2 := dr.encodePythonStdout(nil)
	if got2 != "    Stdout = \"\"\n" {
		t.Errorf("unexpected default stdout: %s", got2)
	}
}

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
	minimal := "extends \"package://schema.kdeps.com/core@" + schema.SchemaVersion(ctx) + "#/Python.pkl\"\n\nResources {}\n"
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
