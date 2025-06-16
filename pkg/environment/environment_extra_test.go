package environment

import (
	"context"
	"os"
	"path/filepath"
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

// TestHelperFunctions covers checkConfig, findKdepsConfig and isDockerEnvironment.
func TestHelperFunctions(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	_ = ctx // reference to context not used but keeps rule: we call schema elsewhere not needed here.

	// create temp pwd and home
	pwd := "/work"
	home := "/home/user"
	if err := fs.MkdirAll(pwd, 0755); err != nil {
		t.Fatalf(err.Error())
	}
	if err := fs.MkdirAll(home, 0755); err != nil {
		t.Fatalf(err.Error())
	}

	// no config yet
	if got := findKdepsConfig(fs, pwd, home); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}

	// add config to home
	cfgPath := filepath.Join(home, SystemConfigFileName)
	afero.WriteFile(fs, cfgPath, []byte("dummy"), 0644)

	if got := findKdepsConfig(fs, pwd, home); got != cfgPath {
		t.Fatalf("expected %s got %s", cfgPath, got)
	}

	// isDockerEnvironment false by default
	if isDockerEnvironment(fs, "/") {
		t.Fatalf("expected not docker env")
	}

	// create /.dockerenv and set required env vars
	afero.WriteFile(fs, "/.dockerenv", []byte(""), 0644)
	os.Setenv("SCHEMA_VERSION", "1")
	os.Setenv("OLLAMA_HOST", "x")
	os.Setenv("KDEPS_HOST", "y")
	defer func() {
		os.Unsetenv("SCHEMA_VERSION")
		os.Unsetenv("OLLAMA_HOST")
		os.Unsetenv("KDEPS_HOST")
	}()

	if !isDockerEnvironment(fs, "/") {
		t.Fatalf("expected docker environment")
	}
}
