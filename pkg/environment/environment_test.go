package environment

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckConfig(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	baseDir := "/test"
	configFilePath := filepath.Join(baseDir, SystemConfigFileName)

	// Test when file does not exist
	_, err := checkConfig(fs, baseDir)
	require.NoError(t, err, "Expected no error when file does not exist")

	// Test when file exists
	if err := afero.WriteFile(fs, configFilePath, []byte{}, 0o644); err != nil {
		fmt.Println(err)
	}

	foundConfig, err := checkConfig(fs, baseDir)
	require.NoError(t, err, "Expected no error when file exists")
	assert.Equal(t, configFilePath, foundConfig, "Expected correct file path")
}

func TestFindKdepsConfig(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	pwd := "/current"
	home := "/home"

	// Test when no kdeps.pkl file exists
	config := findKdepsConfig(fs, pwd, home)
	assert.Empty(t, config, "Expected empty result when no config file exists")

	// Test when kdeps.pkl exists in Pwd
	if err := afero.WriteFile(fs, filepath.Join(pwd, SystemConfigFileName), []byte{}, 0o644); err != nil {
		fmt.Println(err)
	}
	config = findKdepsConfig(fs, pwd, home)
	assert.Equal(t, filepath.Join(pwd, SystemConfigFileName), config, "Expected config file from Pwd directory")

	// Test when kdeps.pkl exists in Home and not in Pwd
	fs = afero.NewMemMapFs() // Reset file system
	if err := afero.WriteFile(fs, filepath.Join(home, SystemConfigFileName), []byte{}, 0o644); err != nil {
		fmt.Println(err)
	}
	config = findKdepsConfig(fs, pwd, home)
	assert.Equal(t, filepath.Join(home, SystemConfigFileName), config, "Expected config file from Home directory")
}

func TestIsDockerEnvironment(t *testing.T) {
	fs := afero.NewMemMapFs()
	root := "/"

	// Test when .dockerenv does not exist
	isDocker := isDockerEnvironment(fs, root)
	assert.False(t, isDocker, "Expected not to be in a Docker environment")

	// Test when .dockerenv exists
	if err := afero.WriteFile(fs, filepath.Join(root, ".dockerenv"), []byte{}, 0o644); err != nil {
		fmt.Println(err)
	}

	isDocker = isDockerEnvironment(fs, root)
	assert.False(t, isDocker, "Expected false due to missing required Docker environment variables")

	// Test when required Docker environment variables are set
	t.Setenv("SCHEMA_VERSION", "1.0")
	t.Setenv("OLLAMA_HOST", "localhost")
	t.Setenv("KDEPS_HOST", "localhost")
	isDocker = isDockerEnvironment(fs, root)
	assert.True(t, isDocker, "Expected true when .dockerenv exists and required environment variables are set")
}

func TestAllDockerEnvVarsSet(t *testing.T) {
	// Ensure environment is clean
	t.Setenv("SCHEMA_VERSION", "")
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("KDEPS_HOST", "")

	// Test when no variables are set
	assert.False(t, allDockerEnvVarsSet(), "Expected false when no variables are set")

	// Test when all variables are set
	t.Setenv("SCHEMA_VERSION", "1.0")
	t.Setenv("OLLAMA_HOST", "localhost")
	t.Setenv("KDEPS_HOST", "localhost")
	assert.True(t, allDockerEnvVarsSet(), "Expected true when all required variables are set")

	// Clean up
	t.Setenv("SCHEMA_VERSION", "")
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("KDEPS_HOST", "")
}

func TestNewEnvironment(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Test with provided environment
	providedEnv := &Environment{
		Root: "/",
		Home: "/home",
		Pwd:  "/current",
	}
	env, err := NewEnvironment(fs, providedEnv)
	require.NoError(t, err, "Expected no error")
	assert.Equal(t, providedEnv.Home, env.Home, "Expected Home directory to match")
	assert.Equal(t, "1", env.NonInteractive, "Expected NonInteractive to be prioritized")

	// Test loading from default environment
	t.Setenv("ROOT_DIR", "/")
	t.Setenv("HOME", "/home")
	t.Setenv("PWD", "/current")
	env, err = NewEnvironment(fs, nil)
	require.NoError(t, err, "Expected no error")
	assert.Equal(t, "/home", env.Home, "Expected Home directory to match")
	assert.Equal(t, "/current", env.Pwd, "Expected Pwd to match")
}
