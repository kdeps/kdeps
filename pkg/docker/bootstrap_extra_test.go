package docker

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/schema/gen/project"
	webserver "github.com/kdeps/schema/gen/web_server"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestStartAndWaitForOllamaReady(t *testing.T) {
	// Spin up dummy listener to simulate Ollama server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}
	defer ln.Close()

	_, portStr, _ := net.SplitHostPort(ln.Addr().String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := logging.NewTestLogger()
	if err := startAndWaitForOllama(ctx, "127.0.0.1", portStr, logger); err != nil {
		t.Errorf("expected nil error when server already ready, got %v", err)
	}
}

// TestStartAPIServerWrapper_Error ensures that the startAPIServer helper
// forwards the error coming from StartAPIServerMode when the API server
// is not properly configured (i.e., workflow settings are missing).
func TestStartAPIServerWrapper_Error(t *testing.T) { //nolint:paralleltest
	mw := &MockWorkflow{} // GetSettings will return nil ➜ configuration missing

	dr := &resolver.DependencyResolver{
		Workflow: mw,
		Logger:   logging.NewTestLogger(),
		Fs:       afero.NewMemMapFs(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := startAPIServer(ctx, dr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "configuration is missing")
}

// TestStartWebServerWrapper_Success verifies that the startWebServer helper
// returns nil when the underlying StartWebServerMode succeeds with a minimal
// (but valid) WebServer configuration.
func TestStartWebServerWrapper_Success(t *testing.T) { //nolint:paralleltest
	portNum := uint16(0) // Ask gin to use any free port

	settings := &project.Settings{
		WebServer: &webserver.WebServerSettings{
			HostIP:  "127.0.0.1",
			PortNum: portNum,
			Routes:  []*webserver.WebServerRoutes{},
		},
	}

	mw := &MockWorkflow{settings: settings}

	dr := &resolver.DependencyResolver{
		Workflow: mw,
		Logger:   logging.NewTestLogger(),
		Fs:       afero.NewMemMapFs(),
		DataDir:  "/tmp",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := startWebServer(ctx, dr)
	require.NoError(t, err)
}

func TestCreateFlagFileExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	filename := "flag.txt"
	// Create new flag file
	err := CreateFlagFile(fs, context.Background(), filename)
	require.NoError(t, err)
	exists, err := afero.Exists(fs, filename)
	require.NoError(t, err)
	require.True(t, exists)

	// Record modification time
	fi, err := fs.Stat(filename)
	require.NoError(t, err)
	mt1 := fi.ModTime()

	// Wait to ensure time difference if updated
	time.Sleep(1 * time.Millisecond)

	// Call again on existing file, should not alter modtime and return no error
	err = CreateFlagFile(fs, context.Background(), filename)
	require.NoError(t, err)
	fi2, err := fs.Stat(filename)
	require.NoError(t, err)
	require.Equal(t, mt1, fi2.ModTime())
}

// minimalDependencyResolver returns a DependencyResolver with only fields
// required by BootstrapDockerSystem when DockerMode != "1" (fast-path).
func minimalDependencyResolver(fs afero.Fs) *resolver.DependencyResolver {
	return &resolver.DependencyResolver{
		Fs:          fs,
		Environment: &environment.Environment{DockerMode: "0"},
		Logger:      logging.NewTestLogger(),
	}
}

func TestBootstrapDockerSystem_NonDockerMode2(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := minimalDependencyResolver(fs)

	apiMode, err := BootstrapDockerSystem(context.Background(), dr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if apiMode {
		t.Fatalf("expected apiMode=false for non-docker environment")
	}
}

func TestBootstrapDockerSystem_NilLogger2(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &resolver.DependencyResolver{
		Fs:          fs,
		Environment: &environment.Environment{DockerMode: "0"},
		Logger:      nil,
	}
	if _, err := BootstrapDockerSystem(context.Background(), dr); err == nil {
		t.Fatalf("expected error when logger is nil")
	}
}

func TestCreateFlagFileAgain(t *testing.T) {
	fs := afero.NewMemMapFs()
	filename := "/tmp/test.flag"

	// First creation should succeed
	if err := CreateFlagFile(fs, context.Background(), filename); err != nil {
		t.Fatalf("unexpected error creating flag file: %v", err)
	}

	// Verify file exists and timestamps are recent
	info, err := fs.Stat(filename)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if time.Since(info.ModTime()) > time.Minute {
		t.Fatalf("unexpected mod time: %v", info.ModTime())
	}

	// Second call should not error (file already exists)
	if err := CreateFlagFile(fs, context.Background(), filename); err != nil {
		t.Fatalf("expected nil error when flag already exists, got: %v", err)
	}
}
