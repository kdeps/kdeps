package archiver_test

import (
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestFindWorkflowFile_Success(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	// tmp dir acting as project root
	projectDir, err := afero.TempDir(fs, "", "test-fwf-success")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	// create nested path e.g. path/to/workflow.pkl
	nestedDir := filepath.Join(projectDir, "path", "to")
	assert.NoError(t, fs.MkdirAll(nestedDir, 0o755))
	wfPath := filepath.Join(nestedDir, "workflow.pkl")
	assert.NoError(t, afero.WriteFile(fs, wfPath, []byte("dummy"), 0o644))

	found, err := archiver.FindWorkflowFile(fs, projectDir, logger)
	assert.NoError(t, err)
	assert.Equal(t, wfPath, found)
}

func TestFindWorkflowFile_NotFound(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	projectDir, err := afero.TempDir(fs, "", "test-fwf-notfound")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	_, err = archiver.FindWorkflowFile(fs, projectDir, logger)
	assert.Error(t, err)
}

func TestFindWorkflowFile_PathIsFile(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	tmpFile, err := afero.TempFile(fs, "", "notadir-*.pkl")
	assert.NoError(t, err)
	tmpFile.Close()
	defer fs.Remove(tmpFile.Name())

	_, err = archiver.FindWorkflowFile(fs, tmpFile.Name(), logger)
	assert.Error(t, err)
}
