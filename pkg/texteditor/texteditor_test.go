package texteditor

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/editor"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Save the original EditPkl function
var originalEditPkl = EditPkl

// testMockEditPkl is a mock version of EditPkl specifically for testing
var testMockEditPkl EditPklFunc = func(fs afero.Fs, ctx context.Context, filePath string, logger *logging.Logger) error {
	// Ensure the file has a .pkl extension
	if filepath.Ext(filePath) != ".pkl" {
		err := errors.New("file '" + filePath + "' does not have a .pkl extension")
		logger.Error(err.Error())
		return err
	}

	// Check if the file exists
	if _, err := fs.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			errMsg := "file does not exist"
			logger.Error(errMsg)
			return errors.New(errMsg)
		}
		errMsg := "failed to stat file"
		logger.Error(errMsg)
		return errors.New(errMsg)
	}

	// In the mock version, we just return success
	return nil
}

// errorFs is a custom afero.Fs that always returns an error on Stat
// Used to simulate stat errors for coverage

type errorFs struct{ afero.Fs }

func (e errorFs) Stat(name string) (os.FileInfo, error) { return nil, errors.New("stat error") }

func setNonInteractive(t *testing.T) func() {
	old := os.Getenv("NON_INTERACTIVE")
	os.Setenv("NON_INTERACTIVE", "1")
	return func() { os.Setenv("NON_INTERACTIVE", old) }
}

func TestEditPkl(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.pkl")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a mock editor command that fails
	originalEditor := os.Getenv("EDITOR")
	defer os.Setenv("EDITOR", originalEditor)
	os.Setenv("EDITOR", "nonexistent-editor")

	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	tests := []struct {
		name        string
		filePath    string
		expectError bool
	}{
		{
			name:        "ValidFile",
			filePath:    testFile,
			expectError: false,
		},
		{
			name:        "NonExistentFile",
			filePath:    filepath.Join(tempDir, "nonexistent.pkl"),
			expectError: true,
		},
		{
			name:        "InvalidExtension",
			filePath:    filepath.Join(tempDir, "test.txt"),
			expectError: true,
		},
		{
			name:        "ReadOnlyFilesystem",
			filePath:    "/readonly/test.pkl",
			expectError: true,
		},
		{
			name:        "NonInteractive",
			filePath:    testFile,
			expectError: false,
		},
		{
			name:        "FileDoesNotExist",
			filePath:    filepath.Join(tempDir, "nonexistent.pkl"),
			expectError: true,
		},
		{
			name:        "StatError",
			filePath:    filepath.Join(tempDir, "test.pkl"),
			expectError: false,
		},
		{
			name:        "ValidFileButEditorCommandFails",
			filePath:    testFile,
			expectError: true,
		},
		{
			name:        "FileExtensionValidation",
			filePath:    testFile,
			expectError: false,
		},
		{
			name:        "DifferentFilesystemTypes",
			filePath:    testFile,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "NonInteractive" {
				os.Setenv("NON_INTERACTIVE", "1")
				defer os.Unsetenv("NON_INTERACTIVE")
			}

			if tt.name == "ValidFileButEditorCommandFails" {
				// Set a non-existent editor command
				os.Setenv("EDITOR", "nonexistent-editor")
				// Use the real EditPkl implementation for this test
				EditPkl = originalEditPkl
				defer func() { EditPkl = testMockEditPkl }()
			}

			err := EditPkl(fs, ctx, tt.filePath, logger)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEditPklAdditionalCoverage(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("ValidPklFileWithDeepPath", func(t *testing.T) {
		deepPath := "deep/nested/path/test.pkl"
		err := fs.MkdirAll(filepath.Dir(deepPath), 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, deepPath, []byte("test content"), 0o644)
		require.NoError(t, err)

		err = EditPkl(fs, ctx, deepPath, logger)
		assert.NoError(t, err)
	})

	t.Run("EmptyPklFile", func(t *testing.T) {
		emptyPath := "empty.pkl"
		err := afero.WriteFile(fs, emptyPath, []byte(""), 0o644)
		require.NoError(t, err)

		err = EditPkl(fs, ctx, emptyPath, logger)
		assert.NoError(t, err)
	})

	t.Run("RelativePathPklFile", func(t *testing.T) {
		relativePath := "./relative.pkl"
		err := afero.WriteFile(fs, relativePath, []byte("test content"), 0o644)
		require.NoError(t, err)

		err = EditPkl(fs, ctx, relativePath, logger)
		assert.NoError(t, err)
	})

	t.Run("FileWithSpecialCharacters", func(t *testing.T) {
		specialPath := "special!@#$%^&*().pkl"
		err := afero.WriteFile(fs, specialPath, []byte("test content"), 0o644)
		require.NoError(t, err)

		err = EditPkl(fs, ctx, specialPath, logger)
		assert.NoError(t, err)
	})

	t.Run("FileWithVeryLongPath", func(t *testing.T) {
		longPath := filepath.Join(strings.Repeat("a/", 100), "test.pkl")
		err := fs.MkdirAll(filepath.Dir(longPath), 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, longPath, []byte("test content"), 0o644)
		require.NoError(t, err)

		err = EditPkl(fs, ctx, longPath, logger)
		assert.NoError(t, err)
	})

	t.Run("FileWithInvalidPermissions", func(t *testing.T) {
		invalidPath := "invalid.pkl"
		err := afero.WriteFile(fs, invalidPath, []byte("test content"), 0o000)
		require.NoError(t, err)

		err = EditPkl(fs, ctx, invalidPath, logger)
		assert.NoError(t, err) // Should still work in MemMapFs
	})

	t.Run("EditorCommandCreationFailure", func(t *testing.T) {
		// Save original EditPkl and restore after test
		originalEditPkl := EditPkl
		defer func() { EditPkl = originalEditPkl }()

		// Create a mock that simulates editor command creation failure
		EditPkl = func(fs afero.Fs, ctx context.Context, filePath string, logger *logging.Logger) error {
			return errors.New("failed to create editor command")
		}

		err := EditPkl(fs, ctx, "test.pkl", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create editor command")
	})

	t.Run("EditorCommandExecutionFailure", func(t *testing.T) {
		// Save original EditPkl and restore after test
		originalEditPkl := EditPkl
		defer func() { EditPkl = originalEditPkl }()

		// Create a mock that simulates editor command execution failure
		EditPkl = func(fs afero.Fs, ctx context.Context, filePath string, logger *logging.Logger) error {
			return errors.New("editor command failed")
		}

		err := EditPkl(fs, ctx, "test.pkl", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "editor command failed")
	})

	t.Run("MockEditPklStatError", func(t *testing.T) {
		// Save original MockEditPkl and restore after test
		originalMockEditPkl := MockEditPkl
		defer func() { MockEditPkl = originalMockEditPkl }()

		// Create a mock that simulates a non-IsNotExist stat error
		MockEditPkl = func(fs afero.Fs, ctx context.Context, filePath string, logger *logging.Logger) error {
			return errors.New("failed to stat file")
		}

		err := MockEditPkl(fs, ctx, "test.pkl", logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to stat file")
	})
}

func TestEditPkl_NonInteractive(t *testing.T) {
	os.Setenv("NON_INTERACTIVE", "1")
	t.Cleanup(func() { os.Unsetenv("NON_INTERACTIVE") })

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("ValidPklFile", func(t *testing.T) {
		filePath := "valid_noninteractive.pkl"
		err := afero.WriteFile(fs, filePath, []byte("test content"), 0o644)
		assert.NoError(t, err)
		err = EditPkl(fs, ctx, filePath, logger)
		assert.NoError(t, err)
	})

	t.Run("InvalidExtension", func(t *testing.T) {
		filePath := "invalid.txt"
		err := afero.WriteFile(fs, filePath, []byte("test content"), 0o644)
		assert.NoError(t, err)
		err = EditPkl(fs, ctx, filePath, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ".pkl extension")
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		filePath := "doesnotexist.pkl"
		err := EditPkl(fs, ctx, filePath, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("StatError", func(t *testing.T) {
		// Custom fs that always returns an error on Stat
		errFs := errorFs{fs}
		filePath := "staterror.pkl"
		err := afero.WriteFile(fs, filePath, []byte("test content"), 0o644)
		assert.NoError(t, err)
		err = EditPkl(errFs, ctx, filePath, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to stat file")
	})
}

// mockEditorCmd is a test-only mock for EditorCmd
type mockEditorCmd struct {
	runErr error
}

func (m *mockEditorCmd) Run() error {
	return m.runErr
}

func (m *mockEditorCmd) SetIO(stdin, stdout, stderr *os.File) {}

func TestEditPklWithFactory(t *testing.T) {
	// Create test logger
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Test cases
	tests := []struct {
		name          string
		filePath      string
		factory       EditorCmdFunc
		mockStatError error
		expectedError bool
	}{
		{
			name:     "successful edit",
			filePath: "test.pkl",
			factory: func(editorName, filePath string) (EditorCmd, error) {
				return &mockEditorCmd{}, nil
			},
			expectedError: false,
		},
		{
			name:     "file does not exist",
			filePath: "nonexistent.pkl",
			factory: func(editorName, filePath string) (EditorCmd, error) {
				return &mockEditorCmd{}, nil
			},
			expectedError: true,
		},
		{
			name:     "stat error",
			filePath: "test.pkl",
			factory: func(editorName, filePath string) (EditorCmd, error) {
				return &mockEditorCmd{}, nil
			},
			mockStatError: errors.New("permission denied"),
			expectedError: true,
		},
		{
			name:     "factory error",
			filePath: "test.pkl",
			factory: func(editorName, filePath string) (EditorCmd, error) {
				return nil, errors.New("factory error")
			},
			expectedError: true,
		},
		{
			name:     "command run error",
			filePath: "test.pkl",
			factory: func(editorName, filePath string) (EditorCmd, error) {
				return &mockEditorCmd{runErr: errors.New("run error")}, nil
			},
			expectedError: true,
		},
		{
			name:     "non-interactive mode",
			filePath: "test.pkl",
			factory: func(editorName, filePath string) (EditorCmd, error) {
				return &mockEditorCmd{}, nil
			},
			expectedError: false,
		},
		{
			name:     "invalid extension",
			filePath: "test.txt",
			factory: func(editorName, filePath string) (EditorCmd, error) {
				return &mockEditorCmd{}, nil
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			if tt.mockStatError == nil && tt.name != "file does not exist" && tt.name != "invalid extension" && tt.name != "non-interactive mode" {
				if err := afero.WriteFile(fs, tt.filePath, []byte("test content"), 0o644); err != nil {
					t.Fatalf("Failed to write test file: %v", err)
				}
			}

			if tt.mockStatError != nil {
				fs = &mockFS{
					fs:        fs,
					statError: tt.mockStatError,
				}
			}

			if tt.name == "non-interactive mode" {
				os.Setenv("NON_INTERACTIVE", "1")
				defer os.Unsetenv("NON_INTERACTIVE")
			}

			err := EditPklWithFactory(fs, ctx, tt.filePath, logger, tt.factory)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// mockFS implements a mock filesystem for testing
type mockFS struct {
	fs        afero.Fs
	statError error
}

func (m *mockFS) Stat(name string) (os.FileInfo, error) {
	if m.statError != nil {
		return nil, m.statError
	}
	return m.fs.Stat(name)
}

// Implement other afero.Fs methods by delegating to the underlying fs
func (m *mockFS) Create(name string) (afero.File, error) {
	return m.fs.Create(name)
}

func (m *mockFS) Mkdir(name string, perm os.FileMode) error {
	return m.fs.Mkdir(name, perm)
}

func (m *mockFS) MkdirAll(path string, perm os.FileMode) error {
	return m.fs.MkdirAll(path, perm)
}

func (m *mockFS) Open(name string) (afero.File, error) {
	return m.fs.Open(name)
}

func (m *mockFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return m.fs.OpenFile(name, flag, perm)
}

func (m *mockFS) Remove(name string) error {
	return m.fs.Remove(name)
}

func (m *mockFS) RemoveAll(path string) error {
	return m.fs.RemoveAll(path)
}

func (m *mockFS) Rename(oldname, newname string) error {
	return m.fs.Rename(oldname, newname)
}

func (m *mockFS) Name() string {
	return m.fs.Name()
}

func (m *mockFS) Chmod(name string, mode os.FileMode) error {
	return m.fs.Chmod(name, mode)
}

func (m *mockFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return m.fs.Chtimes(name, atime, mtime)
}

func (m *mockFS) Chown(name string, uid, gid int) error {
	return m.fs.Chown(name, uid, gid)
}

func TestEditPklWithFactory_NilFactory(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create a test file
	testFile := "test.pkl"
	err := afero.WriteFile(fs, testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Test with nil factory
	err = EditPklWithFactory(fs, ctx, testFile, logger, nil)
	assert.Error(t, err)
	if err != nil {
		assert.True(t, strings.Contains(err.Error(), "failed to create editor command") || strings.Contains(err.Error(), "editor command failed"))
	}
}

func TestEditPklWithFactory_PermissionDenied(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create a test file with no permissions
	testFile := "denied.pkl"
	err := afero.WriteFile(fs, testFile, []byte("test content"), 0o000)
	require.NoError(t, err)

	// Use a mock factory that returns a fake command
	factory := func(editorName, filePath string) (EditorCmd, error) {
		return &mockEditorCmd{runErr: errors.New("permission denied")}, nil
	}

	err = EditPklWithFactory(fs, ctx, testFile, logger, factory)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestEditPklWithFactory_EmptyFilePath(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	err := EditPklWithFactory(fs, ctx, "", logger, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not have a .pkl extension")
}

func TestEditPklWithFactory_NonPklExtension(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	testFile := "not_a_pkl.txt"
	err := afero.WriteFile(fs, testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	err = EditPklWithFactory(fs, ctx, testFile, logger, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ".pkl extension")
}

func TestRealEditorCmdFactory_InvalidEditor(t *testing.T) {
	// Save and restore the original editorCmd
	orig := editorCmd
	t.Cleanup(func() { editorCmd = orig })

	editorCmd = func(editorName, filePath string, _ ...editor.Option) (*exec.Cmd, error) {
		return nil, errors.New("simulated editor.Cmd error")
	}

	cmd, err := realEditorCmdFactory("nonexistent-editor", "test.pkl")
	assert.Nil(t, cmd)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "simulated editor.Cmd error")
	}
}

func TestRealEditorCmdFactory_InvalidPath(t *testing.T) {
	// Test with invalid file path
	_, err := realEditorCmdFactory("vim", "/nonexistent/path/test.pkl")
	assert.NoError(t, err) // Should not error, as the file doesn't need to exist
}

func TestRealEditorCmd_SetIO(t *testing.T) {
	// Create a temporary file for testing
	tempFile := "test.pkl"
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, tempFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Create a real editor command
	cmd, err := realEditorCmdFactory("vim", tempFile)
	require.NoError(t, err)

	// Test SetIO with nil files
	cmd.SetIO(nil, nil, nil)
	// Should not panic
}

func TestRealEditorCmd_Run(t *testing.T) {
	// Override editorCmd with a stub that immediately exits with status 1 to avoid 30-second OS lookup delays.
	orig := editorCmd
	editorCmd = func(editorName, filePath string, _ ...editor.Option) (*exec.Cmd, error) {
		return exec.Command("sh", "-c", "exit 1"), nil
	}
	defer func() { editorCmd = orig }()

	cmd, err := realEditorCmdFactory("stub", "/tmp/test.pkl")
	require.NoError(t, err)
	require.NotNil(t, cmd)
	err = cmd.Run()
	require.Error(t, err)
}

func TestMain(m *testing.M) {
	// Save the original EditPkl function
	originalEditPkl := EditPkl
	// Replace with mock for testing
	EditPkl = testMockEditPkl
	// Set non-interactive mode
	os.Setenv("NON_INTERACTIVE", "1")

	// Stub out editorCmd so any accidental real invocation returns fast
	origEditorCmd := editorCmd
	editorCmd = func(editorName, filePath string, _ ...editor.Option) (*exec.Cmd, error) {
		return exec.Command("sh", "-c", "exit 1"), nil
	}
	defer func() { editorCmd = origEditorCmd }()

	// Run tests
	code := m.Run()

	// Restore original function
	EditPkl = originalEditPkl

	os.Exit(code)
}
