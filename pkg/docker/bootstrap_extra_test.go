package docker

import (
	"context"
	"net"
	"testing"
	"time"

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
