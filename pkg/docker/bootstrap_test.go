package docker

import (
	"context"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/schema/gen/project"
	webserver "github.com/kdeps/schema/gen/web_server"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapDockerSystem(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	tmpDir := t.TempDir()
	actionDir := filepath.Join(tmpDir, "action")
	_ = fs.MkdirAll(actionDir, 0o755)
	dr := &resolver.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		ActionDir: actionDir,
		Environment: &environment.Environment{
			DockerMode: "1",
		},
	}

	t.Run("NonDockerMode", func(t *testing.T) {
		dr.Environment.DockerMode = "0"
		apiServerMode, err := BootstrapDockerSystem(ctx, dr)
		require.NoError(t, err)
		assert.False(t, apiServerMode)
	})

	t.Run("DockerMode", func(t *testing.T) {
		dr.Environment.DockerMode = "1"
		apiServerMode, err := BootstrapDockerSystem(ctx, dr)
		require.Error(t, err) // Expected error due to missing OLLAMA_HOST
		assert.False(t, apiServerMode)
	})
}

func TestCreateFlagFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		err := CreateFlagFile(fs, ctx, "/tmp/flag")
		require.NoError(t, err)
		exists, _ := afero.Exists(fs, "/tmp/flag")
		assert.True(t, exists)
	})

	t.Run("FileExists", func(t *testing.T) {
		_ = afero.WriteFile(fs, "/tmp/existing", []byte(""), 0o644)
		err := CreateFlagFile(fs, ctx, "/tmp/existing")
		assert.NoError(t, err)
	})
}

func TestPullModels(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("EmptyModels", func(t *testing.T) {
		err := PullModels(ctx, []string{}, logger)
		assert.NoError(t, err)
	})

	t.Run("ModelPull", func(t *testing.T) {
		// This test requires a running OLLAMA service and may not be suitable for all environments
		// Consider mocking the KdepsExec function for more reliable testing
		t.Skip("Skipping test that requires OLLAMA service")
	})
}

func TestStartAPIServer(t *testing.T) {
	ctx := context.Background()
	dr := &resolver.DependencyResolver{
		Logger: logging.NewTestLogger(),
	}

	t.Run("StartAPIServer", func(t *testing.T) {
		// This test requires a running Docker daemon and may not be suitable for all environments
		// Consider mocking the StartAPIServerMode function for more reliable testing
		t.Skip("Skipping test that requires Docker daemon")
		_ = ctx // Use context to avoid linter error
		_ = dr  // Use dr to avoid linter error
	})
}

func TestStartWebServer(t *testing.T) {
	ctx := context.Background()
	dr := &resolver.DependencyResolver{
		Logger: logging.NewTestLogger(),
	}

	t.Run("StartWebServer", func(t *testing.T) {
		// This test requires a running Docker daemon and may not be suitable for all environments
		// Consider mocking the StartWebServerMode function for more reliable testing
		t.Skip("Skipping test that requires Docker daemon")
		_ = ctx // Use context to avoid linter error
		_ = dr  // Use dr to avoid linter error
	})
}

func TestCreateFlagFileNoDuplicate(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	filename := "/tmp/flag.txt"

	// First creation should succeed and file should exist.
	if err := CreateFlagFile(fs, ctx, filename); err != nil {
		t.Fatalf("CreateFlagFile error: %v", err)
	}
	if ok, _ := afero.Exists(fs, filename); !ok {
		t.Fatalf("expected file to exist after creation")
	}

	// Second creation should be no-op with no error (file already exists).
	if err := CreateFlagFile(fs, ctx, filename); err != nil {
		t.Fatalf("expected no error on second create, got %v", err)
	}
}

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
func TestStartAPIServerWrapper_Error(t *testing.T) {
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
func TestStartWebServerWrapper_Success(t *testing.T) {
	portNum := uint16(0) // Ask gin to use any free port

	settings := &project.Settings{
		WebServer: &webserver.WebServerSettings{
			HostIP:  "127.0.0.1",
			PortNum: portNum,
			Routes:  []webserver.WebServerRoutes{},
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

func TestCreateFlagFile_ReadOnlyFs(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "roflag")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}

	ro := afero.NewReadOnlyFs(fs)
	flagPath := filepath.Join(tmpDir, "flag.txt")

	// Attempting to create a new file on read-only FS should error.
	if err := CreateFlagFile(ro, context.Background(), flagPath); err == nil {
		t.Fatalf("expected error when creating flag file on read-only fs")
	}

	// Reference schema version (requirement in tests)
	_ = schema.SchemaVersion(context.Background())
}

func TestCreateFlagFile_NewFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	filename := "test_flag_file"

	if err := CreateFlagFile(fs, ctx, filename); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exists, _ := afero.Exists(fs, filename)
	if !exists {
		t.Fatalf("expected flag file to be created")
	}

	// Check timestamps roughly current (within 2 seconds)
	info, _ := fs.Stat(filename)
	if time.Since(info.ModTime()) > 2*time.Second {
		t.Fatalf("mod time too old: %v", info.ModTime())
	}
}

func TestCreateFlagFile_FileAlreadyExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	filename := "existing_flag"

	// pre-create file
	afero.WriteFile(fs, filename, []byte{}, 0o644)

	if err := CreateFlagFile(fs, ctx, filename); err != nil {
		t.Fatalf("expected no error when file already exists, got: %v", err)
	}
}

func TestPullModels_Error(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Test with a nonexistent model
	err := PullModels(ctx, []string{"nonexistent-model-1"}, logger)

	if err != nil {
		errorStr := err.Error()
		// Check if the error is about binary availability
		if strings.Contains(errorStr, "ollama binary not available") {
			// This is expected if ollama is not installed in the test environment
			t.Logf("Expected error due to missing ollama binary: %v", err)
			return
		}
		// If there's any other error, that would be unexpected
		t.Fatalf("unexpected error: %v", err)
	}

	// If no error was returned, it means ollama is available and handled the
	// nonexistent model gracefully (logged warning but continued)
	t.Log("Ollama binary is available and handled nonexistent model gracefully")
}

func TestEnsureKdepsDirectories(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("Success", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		baseDir := "/test/base"

		err := ensureKdepsDirectories(fs, baseDir, logger)
		assert.NoError(t, err)

		// Check that directories were created
		exists, _ := afero.DirExists(fs, baseDir)
		assert.True(t, exists)

		exists, _ = afero.DirExists(fs, baseDir+"/agents")
		assert.True(t, exists)

		exists, _ = afero.DirExists(fs, baseDir+"/cache")
		assert.True(t, exists)
	})

	t.Run("EmptyBase", func(t *testing.T) {
		fs := afero.NewMemMapFs()

		err := ensureKdepsDirectories(fs, "", logger)
		assert.NoError(t, err)

		// Should use default path
		exists, _ := afero.DirExists(fs, "/agent/volume/")
		assert.True(t, exists)
	})

	t.Run("BaseWithoutTrailingSlash", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		baseDir := "/test/base"

		err := ensureKdepsDirectories(fs, baseDir, logger)
		assert.NoError(t, err)

		// Should normalize to add trailing slash
		exists, _ := afero.DirExists(fs, baseDir+"/")
		assert.True(t, exists)
	})
}

func TestCopyOfflineModels(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("SourceDirectoryNotFound", func(t *testing.T) {
		// Test when source directory doesn't exist
		err := copyOfflineModels(ctx, []string{"model1"}, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check offline models directory")
	})

	t.Run("Success", func(t *testing.T) {
		// This test would require setting up actual directories and files
		// For now, we'll test the error path when source doesn't exist
		t.Skip("Skipping test that requires file system setup")
	})
}

func TestSetupDockerEnvironment(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("MissingOLLAMAHost", func(t *testing.T) {
		fs := afero.NewOsFs()
		tmpDir := t.TempDir()
		actionDir := filepath.Join(tmpDir, "action")
		require.NoError(t, fs.MkdirAll(actionDir, 0o755))

		dr := &resolver.DependencyResolver{
			Fs:        fs,
			Logger:    logger,
			ActionDir: actionDir,
			Environment: &environment.Environment{
				DockerMode: "1",
			},
		}

		_, err := setupDockerEnvironment(ctx, dr)
		// Should fail due to missing OLLAMA_HOST environment variable
		assert.Error(t, err)
	})
}

func TestStartAndWaitForOllama(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("ServerNotReady", func(t *testing.T) {
		// Test with a port that won't have a server (use a very high port number)
		err := startAndWaitForOllama(context.Background(), "127.0.0.1", "59999", logger)
		assert.Error(t, err)
		// The error message may vary depending on the system, so just check it's an error
		assert.NotNil(t, err)
	})
}

func TestPullModels_NoModels(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	err := PullModels(ctx, nil, logger)
	assert.NoError(t, err)
}

func TestPullModels_SingleModel(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Test with empty/whitespace model names
	err := PullModels(ctx, []string{"", "   ", "valid-model"}, logger)
	// Should handle empty models gracefully and attempt to pull valid ones
	// May fail if ollama is not available, but shouldn't panic
	if err != nil {
		assert.Contains(t, err.Error(), "ollama")
	}
}
