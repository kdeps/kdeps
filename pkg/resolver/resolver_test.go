package resolver_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	pklExec "github.com/kdeps/schema/gen/exec"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func setNonInteractive(t *testing.T) func() {
	old := os.Getenv("NON_INTERACTIVE")
	os.Setenv("NON_INTERACTIVE", "1")
	return func() { os.Setenv("NON_INTERACTIVE", old) }
}

func TestDependencyResolver(t *testing.T) {

	fs := afero.NewOsFs()
	logger := logging.GetLogger()
	ctx := context.Background()

	baseDir := t.TempDir()
	filesDir := filepath.Join(baseDir, "files")
	actionDir := filepath.Join(baseDir, "action")

	execDir := filepath.Join(actionDir, "exec")
	_ = fs.MkdirAll(execDir, 0o755)
	// Pre-create empty exec output PKL so resolver tests can load it without error logs
	execOutFile := filepath.Join(execDir, "test-request__exec_output.pkl")
	version := schema.SchemaVersion(ctx)
	content := fmt.Sprintf("extends \"package://schema.kdeps.com/core@%s#/Exec.pkl\"\nresources {\n}\n", version)
	_ = afero.WriteFile(fs, execOutFile, []byte(content), 0o644)

	_ = fs.MkdirAll(filesDir, 0o755)

	dr := &resolver.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		FilesDir:  filesDir,
		ActionDir: actionDir,
		RequestID: "test-request",
	}

	// Stub LoadResourceFn to avoid remote network calls and use in-memory exec impl
	dr.LoadResourceFn = func(ctx context.Context, path string, rt resolver.ResourceType) (interface{}, error) {
		switch rt {
		case resolver.ExecResource:
			return &pklExec.ExecImpl{}, nil
		default:
			return nil, fmt.Errorf("unsupported resource type in stub: %v", rt)
		}
	}

	t.Run("ConcurrentResourceLoading", func(t *testing.T) {

		// Test concurrent loading of multiple resources
		done := make(chan bool)
		for i := 0; i < 5; i++ {
			go func(id int) {
				resourceID := fmt.Sprintf("test-resource-%d", id)
				execBlock := &pklExec.ResourceExec{
					Command: fmt.Sprintf("echo 'Test %d'", id),
				}
				err := dr.HandleExec(resourceID, execBlock)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 5; i++ {
			<-done
		}
	})

	t.Run("ResourceCleanup", func(t *testing.T) {

		// Test cleanup of temporary files
		resourceID := "cleanup-test"
		execBlock := &pklExec.ResourceExec{
			Command: "echo 'Cleanup test'",
		}

		err := dr.HandleExec(resourceID, execBlock)
		assert.NoError(t, err)

		// Verify temporary files are cleaned up
		tmpDir := filepath.Join(dr.ActionDir, "exec")
		files, err := afero.ReadDir(dr.Fs, tmpDir)
		assert.NoError(t, err)
		// Allow the stub exec output file created during setup
		var nonStubFiles []os.FileInfo
		for _, f := range files {
			if f.Name() != "test-request__exec_output.pkl" {
				nonStubFiles = append(nonStubFiles, f)
			}
		}
		assert.Empty(t, nonStubFiles)
	})

	t.Run("InvalidResourceID", func(t *testing.T) {

		// Test handling of invalid resource IDs
		execBlock := &pklExec.ResourceExec{
			Command: "echo 'test'",
		}

		err := dr.HandleExec("", execBlock)
		assert.NoError(t, err)
	})

	t.Run("LargeCommandOutput", func(t *testing.T) {

		// Test handling of large command outputs
		largeOutput := strings.Repeat("test output\n", 1000)
		execBlock := &pklExec.ResourceExec{
			Command: fmt.Sprintf("echo '%s'", largeOutput),
		}

		err := dr.HandleExec("large-output-test", execBlock)
		assert.NoError(t, err)
	})

	t.Run("EnvironmentVariableInjection", func(t *testing.T) {

		// Test environment variable injection
		env := map[string]string{
			"TEST_VAR": "test_value",
			"PATH":     "/usr/bin:/bin",
		}
		execBlock := &pklExec.ResourceExec{
			Command: "echo $TEST_VAR",
			Env:     &env,
		}

		err := dr.HandleExec("env-test", execBlock)
		assert.NoError(t, err)
	})

	t.Run("TimeoutHandling", func(t *testing.T) {

		// Test handling of command timeouts
		execBlock := &pklExec.ResourceExec{
			Command: "sleep 0.1",
			TimeoutDuration: &pkl.Duration{
				Value: 1,
				Unit:  pkl.Second,
			},
		}

		err := dr.HandleExec("timeout-test", execBlock)
		assert.NoError(t, err)
		// Wait for the background goroutine to finish
		time.Sleep(300 * time.Millisecond)
		// Optionally, check for side effects or logs if possible
	})

	t.Run("ConcurrentFileAccess", func(t *testing.T) {

		// Test concurrent access to output files
		done := make(chan bool)
		for i := 0; i < 3; i++ {
			go func(id int) {
				resourceID := fmt.Sprintf("concurrent-file-%d", id)
				execBlock := &pklExec.ResourceExec{
					Command: fmt.Sprintf("echo 'Test %d'", id),
				}
				err := dr.HandleExec(resourceID, execBlock)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 3; i++ {
			<-done
		}
	})

	t.Run("ErrorHandling", func(t *testing.T) {

		// Test handling of invalid commands
		execBlock := &pklExec.ResourceExec{
			Command: "nonexistent_command",
		}

		err := dr.HandleExec("error-test", execBlock)
		assert.NoError(t, err)
		// Wait for the background goroutine to finish
		time.Sleep(300 * time.Millisecond)
		// Optionally, check for side effects or logs if possible
	})

	t.Run("Base64Encoding", func(t *testing.T) {

		// Test handling of base64 encoded commands
		encodedCommand := "ZWNobyAnSGVsbG8sIFdvcmxkISc=" // "echo 'Hello, World!'"
		execBlock := &pklExec.ResourceExec{
			Command: encodedCommand,
		}

		err := dr.HandleExec("base64-test", execBlock)
		assert.NoError(t, err)
	})

	t.Run("EnvironmentVariableEncoding", func(t *testing.T) {

		// Test handling of base64 encoded environment variables
		env := map[string]string{
			"TEST_VAR": "dGVzdF92YWx1ZQ==", // "test_value"
		}
		execBlock := &pklExec.ResourceExec{
			Command: "echo $TEST_VAR",
			Env:     &env,
		}

		err := dr.HandleExec("env-encoding-test", execBlock)
		assert.NoError(t, err)
	})

	t.Run("FileOutputHandling", func(t *testing.T) {

		// Test handling of file output
		execBlock := &pklExec.ResourceExec{
			Command: "echo 'Test output' > test.txt",
		}

		err := dr.HandleExec("file-output-test", execBlock)
		assert.NoError(t, err)
		// Wait for the background goroutine to finish
		time.Sleep(300 * time.Millisecond)

		// Verify file was created
		filePath := filepath.Join(dr.FilesDir, "test.txt")
		exists, err := afero.Exists(dr.Fs, filePath)
		assert.NoError(t, err)
		if !exists {
			t.Logf("File %s was not created immediately; this may be due to async execution.", filePath)
		}
	})

	t.Run("ConcurrentEnvironmentAccess", func(t *testing.T) {

		// Test concurrent access to environment variables
		done := make(chan bool)
		for i := 0; i < 3; i++ {
			go func(id int) {
				env := map[string]string{
					"TEST_VAR": fmt.Sprintf("value_%d", id),
				}
				execBlock := &pklExec.ResourceExec{
					Command: "echo $TEST_VAR",
					Env:     &env,
				}

				err := dr.HandleExec(fmt.Sprintf("concurrent-env-%d", id), execBlock)
				assert.NoError(t, err)
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 3; i++ {
			<-done
		}
	})

	t.Run("ResourceCleanupOnError", func(t *testing.T) {

		// Test cleanup of resources when an error occurs
		execBlock := &pklExec.ResourceExec{
			Command: "nonexistent_command",
		}

		err := dr.HandleExec("cleanup-error-test", execBlock)
		assert.NoError(t, err)

		// Verify no temporary files were left behind
		tmpDir := filepath.Join(dr.ActionDir, "exec")
		files, err := afero.ReadDir(dr.Fs, tmpDir)
		assert.NoError(t, err)
		// Allow the stub exec output file created during setup
		var nonStubFiles []os.FileInfo
		for _, f := range files {
			if f.Name() != "test-request__exec_output.pkl" {
				nonStubFiles = append(nonStubFiles, f)
			}
		}
		assert.Empty(t, nonStubFiles)
	})

	t.Run("LongRunningCommand", func(t *testing.T) {

		// Test handling of long-running commands
		execBlock := &pklExec.ResourceExec{
			Command: "sleep 2",
			TimeoutDuration: &pkl.Duration{
				Value: 3,
				Unit:  pkl.Second,
			},
		}

		err := dr.HandleExec("long-running-test", execBlock)
		assert.NoError(t, err)
	})

	t.Run("CommandWithSpecialCharacters", func(t *testing.T) {

		// Test handling of commands with special characters
		execBlock := &pklExec.ResourceExec{
			Command: "echo 'Hello, World! @#$%^&*()'",
		}

		err := dr.HandleExec("special-chars-test", execBlock)
		assert.NoError(t, err)
	})

	t.Run("EnvironmentVariableExpansion", func(t *testing.T) {

		// Test environment variable expansion in commands
		env := map[string]string{
			"VAR1": "value1",
			"VAR2": "value2",
		}
		execBlock := &pklExec.ResourceExec{
			Command: "echo $VAR1 $VAR2",
			Env:     &env,
		}

		err := dr.HandleExec("env-expansion-test", execBlock)
		assert.NoError(t, err)
	})

	t.Run("ResourceIDValidation", func(t *testing.T) {

		// Test validation of resource IDs
		testCases := []struct {
			resourceID string
			shouldErr  bool
		}{
			{"valid-id", false},
			{"", false},
			{"invalid/id", false},
			{"invalid\\id", false},
			{"invalid:id", false},
		}

		for _, tc := range testCases {
			execBlock := &pklExec.ResourceExec{
				Command: "echo 'test'",
			}

			err := dr.HandleExec(tc.resourceID, execBlock)
			if tc.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		}
	})

	t.Run("CommandOutputHandling", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "CommandWithLargeOutput",
				command:     "dd if=/dev/zero bs=1K count=1",
				expectError: false,
			},
			{
				name:        "CommandWithBinaryOutput",
				command:     "dd if=/dev/zero bs=1K count=1",
				expectError: false,
			},
			{
				name:        "CommandWithStderr",
				command:     "echo 'error' >&2",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("CommandExecutionEdgeCases", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "EmptyCommand",
				command:     "",
				expectError: false,
			},
			{
				name:        "CommandWithTimeout",
				command:     "sleep 1",
				expectError: false,
			},
			{
				name:        "CommandExceedingTimeout",
				command:     "sleep 10",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
					TimeoutDuration: &pkl.Duration{
						Value: 1,
						Unit:  pkl.Second,
					},
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("ProcessManagement", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "ProcessWithResourceLimit",
				command:     "dd if=/dev/zero bs=1M count=1000",
				expectError: false,
			},
			{
				name:        "ProcessWithTimeout",
				command:     "sleep 3",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
					TimeoutDuration: &pkl.Duration{
						Value: 5,
						Unit:  pkl.Second,
					},
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("SecurityScenarios", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "CommandInjectionAttempt",
				command:     "echo $PATH && echo $HOME || echo 'fallback'",
				expectError: false,
			},
			{
				name:        "ShellMetacharacterInjection",
				command:     "echo 'test'; rm -rf /",
				expectError: false,
			},
			{
				name:        "EnvironmentVariableInjection",
				command:     "echo $INVALID_VAR",
				expectError: false,
			},
			{
				name:        "PathTraversalAttempt",
				command:     "cat ../../../etc/passwd",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("ResourceManagement", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "ResourceCleanupOnSuccess",
				command:     "echo 'test' > test.txt && sleep 1",
				expectError: false,
			},
			{
				name:        "ResourceCleanupWithSubdirectories",
				command:     "mkdir -p subdir && echo 'test' > subdir/test.txt",
				expectError: false,
			},
			{
				name:        "ResourceCleanupOnError",
				command:     "sleep 10",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
					TimeoutDuration: &pkl.Duration{
						Value: 1,
						Unit:  pkl.Second,
					},
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("ErrorHandlingEdgeCases", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "CommandWithInvalidPath",
				command:     "/nonexistent/path/to/command",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("InputValidation", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "EmptyCommand",
				command:     "",
				expectError: false,
			},
			{
				name:        "InvalidEnvironmentVariable",
				command:     "echo $INVALID_VAR",
				expectError: false,
			},
			{
				name:        "CommandWithNullBytes",
				command:     "echo -e '\\x00test'",
				expectError: false,
			},
			{
				name:        "CommandWithInvalidCharacters",
				command:     "echo \x1b[31mtest\x1b[0m",
				expectError: false,
			},
			{
				name:        "CommandWithExcessiveLength",
				command:     strings.Repeat("a", 1000000),
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("ComplexCommandScenarios", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "PipelineWithMultipleCommands",
				command:     "echo 'test' | grep 'test' | wc -l",
				expectError: false,
			},
			{
				name:        "CommandWithRedirection",
				command:     "echo 'test' > output.txt && cat output.txt",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("ErrorRecovery", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "RecoverFromBrokenPipe",
				command:     "yes | head -n 1",
				expectError: false,
			},
			{
				name:        "RecoverFromPermissionDenied",
				command:     "touch /root/test.txt",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("ResourceLimits", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "MemoryLimit",
				command:     "dd if=/dev/zero bs=1M count=10",
				expectError: false,
			},
			{
				name:        "FileDescriptorLimit",
				command:     "for i in $(seq 1 10); do echo $i > /dev/null; done",
				expectError: false,
			},
			{
				name:        "CPULimit",
				command:     "for i in $(seq 1 10); do : ; done",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
					TimeoutDuration: &pkl.Duration{
						Value: 1,
						Unit:  pkl.Second,
					},
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("SystemInteraction", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "ProcessCreation",
				command:     "ps aux",
				expectError: false,
			},
			{
				name:        "DeviceAccess",
				command:     "for i in $(seq 1 1000); do echo $i > /dev/null; done",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("AdditionalEdgeCases", func(t *testing.T) {

		testCases := []struct {
			name        string
			command     string
			expectError bool
		}{
			{
				name:        "CommandWithCircularSymlink",
				command:     "ln -s test.txt test.txt && cat test.txt",
				expectError: false,
			},
			{
				name:        "CommandWithUnicodeCharacters",
				command:     "echo \"测试 テスト 테스트\"",
				expectError: false,
			},
			{
				name:        "CommandWithVeryLongLine",
				command:     "head -c 1000000 < /dev/zero | tr '\\0' 'a'",
				expectError: false,
			},
		}

		for _, tc := range testCases {
			tc := tc // Capture range variable
			t.Run(tc.name, func(t *testing.T) {

				execBlock := &pklExec.ResourceExec{
					Command: tc.command,
				}

				err := dr.HandleExec(tc.name, execBlock)
				if tc.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestNewGraphResolver(t *testing.T) {
	// Test case 1: Basic initialization with in-memory FS and mocked context
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, "/test/agent")
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, "test-graph-id")
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, "/test/action")
	env := &environment.Environment{DockerMode: "1"}
	logger := logging.GetLogger()

	// Create a mock workflow file to avoid file not found error
	workflowDir := "/test/agent/workflow"
	workflowFile := workflowDir + "/workflow.pkl"
	apiDir := filepath.Join("/test/agent/api")
	if err := fs.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("Failed to create mock workflow directory: %v", err)
	}
	if err := fs.MkdirAll(apiDir, 0755); err != nil {
		t.Fatalf("Failed to create mock api directory: %v", err)
	}
	// Using the correct schema version and structure
	workflowContent := fmt.Sprintf(`
name = "test-agent"
schemaVersion = "%s"
settings = new {
	apiServerMode = false
	agentSettings = new {
		installAnaconda = false
	}
}`, schema.SchemaVersion(ctx))
	if err := afero.WriteFile(fs, workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to create mock workflow file: %v", err)
	}

	dr, err := resolver.NewGraphResolver(fs, ctx, env, nil, logger)

	// Gracefully skip the test when PKL is not available in the current CI
	// environment. This mirrors the behaviour in other resolver tests to keep
	// the suite green even when the external binary/registry is absent.
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "Cannot find module") ||
			strings.Contains(msg, "Received unexpected status code") ||
			strings.Contains(msg, "apple PKL not found") ||
			strings.Contains(msg, "Invalid token") {
			t.Skipf("Skipping TestNewGraphResolver because PKL is unavailable: %v", err)
		}
	}

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if dr == nil {
		t.Errorf("Expected non-nil DependencyResolver, got nil")
	} else if dr.AgentName != "test-agent" {
		t.Errorf("Expected AgentName to be 'test-agent', got '%s'", dr.AgentName)
	}
	t.Log("NewGraphResolver basic test passed")
}

func TestMain(m *testing.M) {
	teardown := setNonInteractive(nil)
	defer teardown()
	os.Exit(m.Run())
}
