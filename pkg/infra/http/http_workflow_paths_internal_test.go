package http

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsYAMLResourceFile(t *testing.T) {
	t.Parallel()
	assert.True(t, isYAMLResourceFile("resource.yaml"))
	assert.True(t, isYAMLResourceFile("resource.yml"))
	assert.False(t, isYAMLResourceFile("resource.json"))
	assert.False(t, isYAMLResourceFile("script.py"))
}

func TestWorkflowDirFromPath(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/tmp/project", workflowDirFromPath("/tmp/project/workflow.yaml"))
}

func TestWorkflowResourcesDir(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "/tmp/project/resources", workflowResourcesDir("/tmp/project/workflow.yaml"))
}

func TestClearResourcesDir_NonExistentDir(t *testing.T) {
	t.Parallel()
	clearResourcesDir("/tmp/does-not-exist-kdeps-test-xyz")
}

func TestClearResourcesDir_RemovesYAML(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "res.yaml"), []byte("data"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "other.txt"), []byte("text"), 0600))
	clearResourcesDir(dir)
	_, errYAML := os.Stat(filepath.Join(dir, "res.yaml"))
	assert.True(t, os.IsNotExist(errYAML), "yaml file should be removed")
}
