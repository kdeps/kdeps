package environment

import (
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
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
	_ = afero.WriteFile(fs, "/.dockerenv", []byte("x"), 0o644)
	os.Setenv("SCHEMA_VERSION", schema.SchemaVersion(nil))
	os.Setenv("OLLAMA_HOST", "0.0.0.0:1234")
	os.Setenv("KDEPS_HOST", "host")
	t.Cleanup(func() {
		os.Unsetenv("SCHEMA_VERSION")
		os.Unsetenv("OLLAMA_HOST")
		os.Unsetenv("KDEPS_HOST")
	})

	env, err := NewEnvironment(fs, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.DockerMode != "1" {
		t.Fatalf("expected DockerMode '1', got %s", env.DockerMode)
	}
}

func TestNewEnvironment_NonDocker(t *testing.T) {
	fs := afero.NewMemMapFs()
	env, err := NewEnvironment(fs, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if env.DockerMode != "0" {
		t.Fatalf("expected DockerMode '0' in non-docker env")
	}
}

func TestNewEnvironment_Override(t *testing.T) {
	fs := afero.NewMemMapFs()
	over := &Environment{Root: "/", Home: "/home/user", Pwd: "/proj", TimeoutSec: 30}
	env, err := NewEnvironment(fs, over)
	if err != nil {
		t.Fatalf("override error: %v", err)
	}
	if env.NonInteractive != "1" {
		t.Fatalf("override should force NonInteractive=1")
	}
	if env.TimeoutSec != 30 {
		t.Fatalf("expected TimeoutSec propagated")
	}
}
