package environment

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var (
	ctx context.Context
)

func TestCheckConfig(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	baseDir := "/test"
	configFilePath := filepath.Join(baseDir, SystemConfigFileName)

	// Test when file does not exist
	_, err := checkConfig(fs, ctx, baseDir)
	assert.NoError(t, err, "Expected no error when file does not exist")

	// Test when file exists
	afero.WriteFile(fs, configFilePath, []byte{}, 0o644)
	foundConfig, err := checkConfig(fs, ctx, baseDir)
	assert.NoError(t, err, "Expected no error when file exists")
	assert.Equal(t, configFilePath, foundConfig, "Expected correct file path")
}

func TestFindKdepsConfig(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	pwd := "/current"
	home := "/home"

	// Test when no kdeps.pkl file exists
	config := findKdepsConfig(fs, ctx, pwd, home)
	assert.Empty(t, config, "Expected empty result when no config file exists")

	// Test when kdeps.pkl exists in Pwd
	afero.WriteFile(fs, filepath.Join(pwd, SystemConfigFileName), []byte{}, 0o644)
	config = findKdepsConfig(fs, ctx, pwd, home)
	assert.Equal(t, filepath.Join(pwd, SystemConfigFileName), config, "Expected config file from Pwd directory")

	// Test when kdeps.pkl exists in Home and not in Pwd
	fs = afero.NewMemMapFs() // Reset file system
	afero.WriteFile(fs, filepath.Join(home, SystemConfigFileName), []byte{}, 0o644)
	config = findKdepsConfig(fs, ctx, pwd, home)
	assert.Equal(t, filepath.Join(home, SystemConfigFileName), config, "Expected config file from Home directory")
}

func TestIsDockerEnvironment(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	root := "/"

	// Test when .dockerenv does not exist
	isDocker := isDockerEnvironment(fs, ctx, root)
	assert.False(t, isDocker, "Expected not to be in a Docker environment")

	// Test when .dockerenv exists
	afero.WriteFile(fs, filepath.Join(root, ".dockerenv"), []byte{}, 0o644)
	isDocker = isDockerEnvironment(fs, ctx, root)
	assert.False(t, isDocker, "Expected false due to missing required Docker environment variables")

	// Test when required Docker environment variables are set
	os.Setenv("SCHEMA_VERSION", "1.0")
	os.Setenv("OLLAMA_HOST", "localhost")
	os.Setenv("KDEPS_HOST", "localhost")
	isDocker = isDockerEnvironment(fs, ctx, root)
	assert.True(t, isDocker, "Expected true when .dockerenv exists and required environment variables are set")
}

func TestAllDockerEnvVarsSet(t *testing.T) {
	t.Parallel()

	// Ensure environment is clean
	os.Unsetenv("SCHEMA_VERSION")
	os.Unsetenv("OLLAMA_HOST")
	os.Unsetenv("KDEPS_HOST")

	// Test when no variables are set
	assert.False(t, allDockerEnvVarsSet(ctx), "Expected false when no variables are set")

	// Test when all variables are set
	os.Setenv("SCHEMA_VERSION", "1.0")
	os.Setenv("OLLAMA_HOST", "localhost")
	os.Setenv("KDEPS_HOST", "localhost")
	assert.True(t, allDockerEnvVarsSet(ctx), "Expected true when all required variables are set")

	// Clean up
	os.Unsetenv("SCHEMA_VERSION")
	os.Unsetenv("OLLAMA_HOST")
	os.Unsetenv("KDEPS_HOST")
}

func TestNewEnvironment(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()

	// Test with provided environment
	providedEnv := &Environment{
		Root: "/",
		Home: "/home",
		Pwd:  "/current",
	}
	env, err := NewEnvironment(fs, ctx, providedEnv)
	assert.NoError(t, err, "Expected no error")
	assert.Equal(t, providedEnv.Home, env.Home, "Expected Home directory to match")
	assert.Equal(t, "1", env.NonInteractive, "Expected NonInteractive to be prioritized")

	// Test loading from default environment
	os.Setenv("ROOT_DIR", "/")
	os.Setenv("HOME", "/home")
	os.Setenv("PWD", "/current")
	env, err = NewEnvironment(fs, ctx, nil)
	assert.NoError(t, err, "Expected no error")
	assert.Equal(t, "/home", env.Home, "Expected Home directory to match")
	assert.Equal(t, "/current", env.Pwd, "Expected Pwd to match")
}
