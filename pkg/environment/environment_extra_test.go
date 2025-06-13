package environment

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewEnvironment_Provided_NoConfig_NoDocker(t *testing.T) {
	fs := afero.NewMemMapFs()
	envIn := &Environment{
		Root: "/",
		Pwd:  "/pwd",
		Home: "/home",
	}
	newEnv, err := NewEnvironment(fs, envIn)
	assert.NoError(t, err)
	assert.Empty(t, newEnv.KdepsConfig)
	assert.Equal(t, "0", newEnv.DockerMode)
	assert.Equal(t, "1", newEnv.NonInteractive)
}

func TestNewEnvironment_Provided_ConfigInPwd(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/pwd", 0755)
	_ = afero.WriteFile(fs, "/pwd/.kdeps.pkl", []byte(""), 0644)
	envIn := &Environment{Root: "/", Pwd: "/pwd", Home: "/home"}
	newEnv, err := NewEnvironment(fs, envIn)
	assert.NoError(t, err)
	assert.Equal(t, "/pwd/.kdeps.pkl", newEnv.KdepsConfig)
}

func TestNewEnvironment_Provided_ConfigInHomeOnly(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/home", 0755)
	_ = afero.WriteFile(fs, "/home/.kdeps.pkl", []byte(""), 0644)
	envIn := &Environment{Root: "/", Pwd: "/pwd", Home: "/home"}
	newEnv, err := NewEnvironment(fs, envIn)
	assert.NoError(t, err)
	assert.Equal(t, "/home/.kdeps.pkl", newEnv.KdepsConfig)
}

func TestNewEnvironment_DockerDetection(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "/.dockerenv", []byte(""), 0644)
	os.Setenv("SCHEMA_VERSION", "v")
	os.Setenv("OLLAMA_HOST", "h")
	os.Setenv("KDEPS_HOST", "h")
	defer func() {
		os.Unsetenv("SCHEMA_VERSION")
		os.Unsetenv("OLLAMA_HOST")
		os.Unsetenv("KDEPS_HOST")
	}()
	envIn := &Environment{Root: "/", Pwd: "/pwd", Home: "/home"}
	newEnv, err := NewEnvironment(fs, envIn)
	assert.NoError(t, err)
	assert.Equal(t, "1", newEnv.DockerMode)
}
