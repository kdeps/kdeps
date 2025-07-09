package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	assets "github.com/kdeps/schema/assets"
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

func TestAppendPythonEntry_SimpleEncode(t *testing.T) {
	// Setup PKL workspace with embedded schema files
	workspace, err := assets.SetupPKLWorkspaceInTmpDir()
	require.NoError(t, err)
	defer workspace.Cleanup()

	fs := afero.NewOsFs() // Use real FS for PKL operations
	tmpdir := t.TempDir()
	actionDir := filepath.Join(tmpdir, "action")
	pythonDir := filepath.Join(actionDir, "python")
	require.NoError(t, fs.MkdirAll(pythonDir, 0o755))

	ctx := context.Background()
	dr := &DependencyResolver{
		Fs:        fs,
		Context:   ctx,
		ActionDir: actionDir,
		RequestID: "req",
		Logger:    logging.NewTestLogger(),
	}

	// Create minimal initial PKL file with empty resources map using assets
	minimal := "extends \"" + workspace.GetImportPath("Python.pkl") + "\"\n\nResources {}\n"
	pythonPklPath := filepath.Join(pythonDir, dr.RequestID+"__python_output.pkl")
	require.NoError(t, afero.WriteFile(fs, pythonPklPath, []byte(minimal), 0o644))

	// Create new Python resource
	pyResource := &pklPython.ResourcePython{
		Script: "print('hello world')",
		Env:    &map[string]string{"KEY": "value"},
	}

	// Test encoding and appending
	err = dr.AppendPythonEntry("myPython", pyResource)
	require.NoError(t, err)

	// Read back and verify the structure
	content, err := afero.ReadFile(fs, pythonPklPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "[\"myPython\"]")
}

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
	dr := &DependencyResolver{
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
