package resolver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func newTestResolver() *DependencyResolver {
	tmpDir := filepath.Join(os.TempDir(), "kdeps_test_", uuid.NewString())
	// We purposely ignore error for MkdirAll because temp dir should exist
	_ = os.MkdirAll(tmpDir, 0o755)

	// Use the real OS filesystem so that any spawned external tools (e.g. pkl)
	// can read the files. This still keeps everything inside a unique tmpdir.
	fs := afero.NewOsFs()

	actionDir := filepath.Join(tmpDir, "action")
	projectDir := filepath.Join(tmpDir, "project")
	workflowDir := filepath.Join(tmpDir, "workflow")

	_ = fs.MkdirAll(actionDir, 0o755)
	_ = fs.MkdirAll(projectDir, 0o755)
	_ = fs.MkdirAll(workflowDir, 0o755)

	return &DependencyResolver{
		Fs:          fs,
		Logger:      logging.NewTestLogger(),
		Context:     context.Background(),
		ActionDir:   actionDir,
		RequestID:   "test-request",
		ProjectDir:  projectDir,
		WorkflowDir: workflowDir,
	}
}

func TestPrepareImportFiles_CreatesFiles(t *testing.T) {
	dr := newTestResolver()
	assert.NoError(t, dr.PrepareImportFiles())

	// Expected files
	base := dr.ActionDir
	files := []string{
		filepath.Join(base, "llm/"+dr.RequestID+"__llm_output.pkl"),
		filepath.Join(base, "client/"+dr.RequestID+"__client_output.pkl"),
		filepath.Join(base, "exec/"+dr.RequestID+"__exec_output.pkl"),
		filepath.Join(base, "python/"+dr.RequestID+"__python_output.pkl"),
		filepath.Join(base, "data/"+dr.RequestID+"__data_output.pkl"),
	}
	for _, f := range files {
		exists, _ := afero.Exists(dr.Fs, f)
		assert.True(t, exists, "%s should exist", f)
		content, _ := afero.ReadFile(dr.Fs, f)
		assert.Contains(t, string(content), "extends \"package://schema.kdeps.com/core@", "header present")
	}
}

func TestPrependDynamicImports_AddsImports(t *testing.T) {
	dr := newTestResolver()
	pklFile := filepath.Join(dr.ActionDir, "file.pkl")
	_ = dr.Fs.MkdirAll(dr.ActionDir, 0o755)
	// initial content with amends line
	initial := "amends \"base.pkl\"\n\n"
	_ = afero.WriteFile(dr.Fs, pklFile, []byte(initial), 0o644)

	// create exec file to satisfy import existence check
	execFile := filepath.Join(dr.ActionDir, "exec/"+dr.RequestID+"__exec_output.pkl")
	_ = dr.Fs.MkdirAll(filepath.Dir(execFile), 0o755)
	_ = afero.WriteFile(dr.Fs, execFile, []byte("{}"), 0o644)

	assert.NoError(t, dr.PrependDynamicImports(pklFile))
	content, _ := afero.ReadFile(dr.Fs, pklFile)
	s := string(content)
	// Should still start with amends line
	assert.True(t, strings.HasPrefix(s, "amends"))
	// Import for exec alias should be present
	assert.Contains(t, s, "import \""+execFile+"\" as exec")
}

func TestPrepareWorkflowDir_CopiesAndCleans(t *testing.T) {
	dr := newTestResolver()
	fs := dr.Fs
	// setup project files
	_ = fs.MkdirAll(filepath.Join(dr.ProjectDir, "dir"), 0o755)
	_ = afero.WriteFile(fs, filepath.Join(dr.ProjectDir, "file.txt"), []byte("hello"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(dr.ProjectDir, "dir/file2.txt"), []byte("world"), 0o644)

	// first copy
	assert.NoError(t, dr.PrepareWorkflowDir())
	exists, _ := afero.Exists(fs, filepath.Join(dr.WorkflowDir, "file.txt"))
	assert.True(t, exists)

	// create stale file in workflow dir and ensure second run cleans it
	_ = afero.WriteFile(fs, filepath.Join(dr.WorkflowDir, "stale.txt"), []byte("x"), 0o644)
	assert.NoError(t, dr.PrepareWorkflowDir())
	staleExists, _ := afero.Exists(fs, filepath.Join(dr.WorkflowDir, "stale.txt"))
	assert.False(t, staleExists)
}
