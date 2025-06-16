package docker

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

func TestBootstrapDockerSystem_NoLogger(t *testing.T) {
	dr := &resolver.DependencyResolver{}
	if _, err := BootstrapDockerSystem(context.Background(), dr); err == nil {
		t.Fatalf("expected error when Logger is nil")
	}
}

func TestBootstrapDockerSystem_NonDockerMode(t *testing.T) {
	fs := afero.NewMemMapFs()
	env := &environment.Environment{DockerMode: "0"}
	dr := &resolver.DependencyResolver{
		Fs:          fs,
		Logger:      logging.NewTestLogger(),
		Environment: env,
	}
	ok, err := BootstrapDockerSystem(context.Background(), dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected apiServerMode false, got true")
	}
}
