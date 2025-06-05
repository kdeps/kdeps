package template

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestValidateAgentName(t *testing.T) {
	t.Run("ValidName", func(t *testing.T) {
		err := validateAgentName("agent1")
		assert.NoError(t, err)
	})
	t.Run("EmptyName", func(t *testing.T) {
		err := validateAgentName("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
	t.Run("WhitespaceName", func(t *testing.T) {
		err := validateAgentName("   ")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})
	t.Run("NameWithSpaces", func(t *testing.T) {
		err := validateAgentName("agent name")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot contain spaces")
	})
}

func TestCreateDirectory(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	path := "/tmp/testdir"
	// Should succeed
	err := createDirectory(fs, logger, path)
	assert.NoError(t, err)
	exists, _ := afero.DirExists(fs, path)
	assert.True(t, exists)

	// Should fail (simulate error)
	badFS := afero.NewReadOnlyFs(fs)
	err = createDirectory(badFS, logger, "/tmp/readonly")
	assert.Error(t, err)
}

func TestCreateFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	path := "/tmp/testfile.txt"
	content := "hello world"
	// Should succeed
	err := createFile(fs, logger, path, content)
	assert.NoError(t, err)
	data, err := afero.ReadFile(fs, path)
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Should fail (simulate error)
	badFS := afero.NewReadOnlyFs(fs)
	err = createFile(badFS, logger, "/tmp/readonly.txt", "fail")
	assert.Error(t, err)
}

func TestLoadTemplate(t *testing.T) {
	// Should fail for missing template
	_, err := loadTemplate("templates/doesnotexist.pkl", map[string]string{"Header": "", "Name": ""})
	assert.Error(t, err)

	// Should fail for invalid template syntax (simulate by writing a bad template to a temp FS)
	// Not possible with embed.FS, so we skip this case
}

func TestGenerateWorkflowFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	mainDir := "/tmp/agent"
	name := "agent"

	// Create the main directory first
	afero.WriteFile(fs, mainDir+"/dummy", []byte("x"), 0o644)

	err := generateWorkflowFile(fs, ctx, logger, mainDir, name)
	assert.NoError(t, err)
	// Check that the workflow.pkl file was created
	filePath := mainDir + "/workflow.pkl"
	exists, _ := afero.Exists(fs, filePath)
	assert.True(t, exists)

	// Error case: invalid template path (simulate by changing schema version)
	// Not directly possible, so we skip this for now
}

func TestGenerateResourceFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	mainDir := "/tmp/agent"
	name := "agent"

	// Create the main directory first
	afero.WriteFile(fs, mainDir+"/dummy", []byte("x"), 0o644)

	err := generateResourceFiles(fs, ctx, logger, mainDir, name)
	// Should not error (will create files for all embedded templates except workflow.pkl)
	assert.NoError(t, err)
	// Check that the resources directory exists
	resourceDir := mainDir + "/resources"
	dirExists, _ := afero.DirExists(fs, resourceDir)
	assert.True(t, dirExists)
}

func TestGenerateSpecificFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()
	mainDir := "/tmp/agent"
	agentName := "agent"

	// Create the main directory first
	afero.WriteFile(fs, mainDir+"/dummy", []byte("x"), 0o644)

	t.Run("WorkflowFile", func(t *testing.T) {
		err := generateSpecificFile(fs, ctx, logger, mainDir, "workflow", agentName)
		assert.NoError(t, err)
		// Check that the workflow.pkl file was created in the main directory
		filePath := mainDir + "/workflow.pkl"
		exists, _ := afero.Exists(fs, filePath)
		assert.True(t, exists)
		// Data directory should also be created
		dataDir := mainDir + "/data"
		dirExists, _ := afero.DirExists(fs, dataDir)
		assert.True(t, dirExists)
	})

	t.Run("ResourceFile", func(t *testing.T) {
		err := generateSpecificFile(fs, ctx, logger, mainDir, "client", agentName)
		assert.NoError(t, err)
		// Check that the client.pkl file was created in the resources directory
		filePath := mainDir + "/resources/client.pkl"
		exists, _ := afero.Exists(fs, filePath)
		assert.True(t, exists)
		// Data directory should also be created
		dataDir := mainDir + "/data"
		dirExists, _ := afero.DirExists(fs, dataDir)
		assert.True(t, dirExists)
	})

	t.Run("InvalidTemplate", func(t *testing.T) {
		err := generateSpecificFile(fs, ctx, logger, mainDir, "doesnotexist", agentName)
		assert.Error(t, err)
	})
}
