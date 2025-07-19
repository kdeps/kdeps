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

func TestEncodePythonEnv(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{Logger: logging.GetLogger()}

	env := map[string]string{"A": "alpha", "B": "beta"}
	encoded := dr.EncodePythonEnv(&env)
	if encoded == nil || len(*encoded) != 2 {
		t.Fatalf("expected 2 encoded entries")
	}
	if (*encoded)["A"] == "alpha" {
		t.Errorf("value A not encoded")
	}
}

func TestEncodePythonOutputs(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{}
	stderr := "some err"
	stdout := "some out"
	e1, e2 := dr.EncodePythonOutputs(&stderr, &stdout)
	if *e1 == stderr || *e2 == stdout {
		t.Errorf("outputs not encoded: %s %s", *e1, *e2)
	}

	// nil pass-through
	n1, n2 := dr.EncodePythonOutputs(nil, nil)
	if n1 != nil || n2 != nil {
		t.Errorf("expected nil return for nil inputs")
	}
}

func TestEncodePythonStderrStdoutFormatting(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{}
	msg := "line1\nline2"
	got := dr.EncodePythonStderr(&msg)
	if len(got) == 0 || got[0] != ' ' {
		t.Errorf("unexpected format: %s", got)
	}
	got2 := dr.EncodePythonStdout(nil)
	if got2 != "    Stdout = \"\"\n" {
		t.Errorf("unexpected default stdout: %s", got2)
	}
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
