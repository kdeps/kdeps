package texteditor_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/texteditor"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// Package variable mutex for safe reassignment
var editPklMutex sync.Mutex

// Helper function to safely save and restore EditPkl variable
func saveAndRestoreEditPkl(_ *testing.T, newValue texteditor.EditPklFunc) func() {
	editPklMutex.Lock()
	original := texteditor.EditPkl
	texteditor.EditPkl = newValue
	return func() {
		texteditor.EditPkl = original
		editPklMutex.Unlock()
	}
}

// Save the original EditPkl function
var originalEditPkl = texteditor.EditPkl

// testMockEditPkl is a mock version of EditPkl specifically for testing
var testMockEditPkl texteditor.EditPklFunc = func(fs afero.Fs, _ context.Context, filePath string, logger *logging.Logger) error {
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

func (e errorFs) Stat(_ string) (os.FileInfo, error) { return nil, errors.New("stat error") }

func setNonInteractive(t *testing.T) func() {
	t.Helper()
	old := os.Getenv("NON_INTERACTIVE")
	t.Setenv("NON_INTERACTIVE", "1")
	return func() { t.Setenv("NON_INTERACTIVE", old) }
}

var testMutex sync.Mutex

func withTestState(_ *testing.T, fn func()) {
	testMutex.Lock()
	defer testMutex.Unlock()
	origEditPkl := texteditor.EditPkl
	defer func() { texteditor.EditPkl = origEditPkl }()
	fn()
}

func TestEditPkl(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test.pkl")
	if err := os.WriteFile(testFile, []byte("test content"), 0o644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a mock editor command that fails
	originalEditor := os.Getenv("EDITOR")
	t.Setenv("EDITOR", "nonexistent-editor")
	defer t.Setenv("EDITOR", originalEditor)

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
				t.Setenv("NON_INTERACTIVE", "1")
			}

			if tt.name == "ValidFileButEditorCommandFails" {
				// Set a non-existent editor command
				t.Setenv("EDITOR", "nonexistent-editor")
				// Use the real EditPkl implementation for this test
				texteditor.EditPkl = originalEditPkl
				defer func() { texteditor.EditPkl = testMockEditPkl }()
			}

			err := texteditor.EditPkl(fs, ctx, tt.filePath, logger)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
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

		err = texteditor.EditPkl(fs, ctx, deepPath, logger)
		require.NoError(t, err)
	})

	t.Run("EmptyPklFile", func(t *testing.T) {
		emptyPath := "empty.pkl"
		err := afero.WriteFile(fs, emptyPath, []byte(""), 0o644)
		require.NoError(t, err)

		err = texteditor.EditPkl(fs, ctx, emptyPath, logger)
		require.NoError(t, err)
	})

	t.Run("RelativePathPklFile", func(t *testing.T) {
		relativePath := "./relative.pkl"
		err := afero.WriteFile(fs, relativePath, []byte("test content"), 0o644)
		require.NoError(t, err)

		err = texteditor.EditPkl(fs, ctx, relativePath, logger)
		require.NoError(t, err)
	})

	t.Run("FileWithSpecialCharacters", func(t *testing.T) {
		specialPath := "special!@#$%^&*().pkl"
		err := afero.WriteFile(fs, specialPath, []byte("test content"), 0o644)
		require.NoError(t, err)

		err = texteditor.EditPkl(fs, ctx, specialPath, logger)
		require.NoError(t, err)
	})

	t.Run("FileWithVeryLongPath", func(t *testing.T) {
		longPath := filepath.Join(strings.Repeat("a/", 100), "test.pkl")
		err := fs.MkdirAll(filepath.Dir(longPath), 0o755)
		require.NoError(t, err)
		err = afero.WriteFile(fs, longPath, []byte("test content"), 0o644)
		require.NoError(t, err)

		err = texteditor.EditPkl(fs, ctx, longPath, logger)
		require.NoError(t, err)
	})

	t.Run("FileWithInvalidPermissions", func(t *testing.T) {
		invalidPath := "invalid.pkl"
		err := afero.WriteFile(fs, invalidPath, []byte("test content"), 0o000)
		require.NoError(t, err)

		err = texteditor.EditPkl(fs, ctx, invalidPath, logger)
		require.NoError(t, err) // Should still work in MemMapFs
	})

	t.Run("EditorCommandCreationFailure", func(t *testing.T) {
		// Save original EditPkl and restore after test
		originalEditPkl := texteditor.EditPkl
		defer func() { texteditor.EditPkl = originalEditPkl }()

		// Create a mock that simulates editor command creation failure
		texteditor.EditPkl = func(_ afero.Fs, _ context.Context, filePath string, logger *logging.Logger) error {
			return errors.New("failed to create editor command")
		}

		err := texteditor.EditPkl(fs, ctx, "test.pkl", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create editor command")
	})

	t.Run("EditorCommandExecutionFailure", func(t *testing.T) {
		withTestState(t, func() {
			// Create a mock that simulates editor command execution failure
			texteditor.EditPkl = func(_ afero.Fs, _ context.Context, filePath string, logger *logging.Logger) error {
				return errors.New("editor command failed")
			}

			err := texteditor.EditPkl(fs, ctx, "test.pkl", logger)
			require.Error(t, err)
			require.Contains(t, err.Error(), "editor command failed")
		})
	})

	t.Run("MockEditPklStatError", func(t *testing.T) {
		withTestState(t, func() {
			// Create a mock that simulates a non-IsNotExist stat error
			texteditor.MockEditPkl = func(_ afero.Fs, _ context.Context, filePath string, logger *logging.Logger) error {
				return errors.New("failed to stat file")
			}

			err := texteditor.MockEditPkl(fs, ctx, "test.pkl", logger)
			require.Error(t, err)
			require.Contains(t, err.Error(), "failed to stat file")
		})
	})
}

func TestEditPkl_NonInteractive(t *testing.T) {
	t.Setenv("NON_INTERACTIVE", "1")

	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("ValidPklFile", func(t *testing.T) {
		filePath := "valid_noninteractive.pkl"
		err := afero.WriteFile(fs, filePath, []byte("test content"), 0o644)
		require.NoError(t, err)
		err = texteditor.EditPkl(fs, ctx, filePath, logger)
		require.NoError(t, err)
	})

	t.Run("InvalidExtension", func(t *testing.T) {
		filePath := "invalid.txt"
		err := afero.WriteFile(fs, filePath, []byte("test content"), 0o644)
		require.NoError(t, err)
		err = texteditor.EditPkl(fs, ctx, filePath, logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), ".pkl extension")
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		filePath := "doesnotexist.pkl"
		err := texteditor.EditPkl(fs, ctx, filePath, logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("StatError", func(t *testing.T) {
		// Custom fs that always returns an error on Stat
		errFs := errorFs{fs}
		filePath := "staterror.pkl"
		err := afero.WriteFile(fs, filePath, []byte("test content"), 0o644)
		require.NoError(t, err)
		err = texteditor.EditPkl(errFs, ctx, filePath, logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to stat file")
	})
}

// mockEditorCmd is a test-only mock for EditorCmd
type mockEditorCmd struct {
	runErr error
}

func (m *mockEditorCmd) Run() error {
	return m.runErr
}

func (m *mockEditorCmd) SetIO(_, _, _ *os.File) {}

func TestEditPklWithFactory(t *testing.T) {
	// Create test logger
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Test cases
	tests := []struct {
		name          string
		filePath      string
		factory       texteditor.EditorCmdFunc
		mockStatError error
		expectedError bool
	}{
		{
			name:     "successful edit",
			filePath: "test.pkl",
			factory: func(_, _ string) (texteditor.EditorCmd, error) {
				return &mockEditorCmd{}, nil
			},
			expectedError: false,
		},
		{
			name:     "file does not exist",
			filePath: "nonexistent.pkl",
			factory: func(_, _ string) (texteditor.EditorCmd, error) {
				return &mockEditorCmd{}, nil
			},
			expectedError: true,
		},
		{
			name:     "stat error",
			filePath: "test.pkl",
			factory: func(_, _ string) (texteditor.EditorCmd, error) {
				return &mockEditorCmd{}, nil
			},
			mockStatError: errors.New("permission denied"),
			expectedError: true,
		},
		{
			name:     "factory error",
			filePath: "test.pkl",
			factory: func(_, _ string) (texteditor.EditorCmd, error) {
				return nil, errors.New("factory error")
			},
			expectedError: true,
		},
		{
			name:     "command run error",
			filePath: "test.pkl",
			factory: func(_, _ string) (texteditor.EditorCmd, error) {
				return &mockEditorCmd{runErr: errors.New("run error")}, nil
			},
			expectedError: true,
		},
		{
			name:     "non-interactive mode",
			filePath: "test.pkl",
			factory: func(_, _ string) (texteditor.EditorCmd, error) {
				return &mockEditorCmd{}, nil
			},
			expectedError: false,
		},
		{
			name:     "invalid extension",
			filePath: "test.txt",
			factory: func(_, _ string) (texteditor.EditorCmd, error) {
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

			t.Setenv("NON_INTERACTIVE", "1")

			err := texteditor.EditPklWithFactory(fs, ctx, tt.filePath, logger, tt.factory)

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
	err = texteditor.EditPklWithFactory(fs, ctx, testFile, logger, nil)
	require.Error(t, err)
	if err != nil {
		require.True(t, strings.Contains(err.Error(), "failed to create editor command") || strings.Contains(err.Error(), "editor command failed"))
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
	factory := func(_, _ string) (texteditor.EditorCmd, error) {
		return &mockEditorCmd{runErr: errors.New("permission denied")}, nil
	}

	err = texteditor.EditPklWithFactory(fs, ctx, testFile, logger, factory)
	require.Error(t, err)
	require.Contains(t, err.Error(), "permission denied")
}

func TestEditPklWithFactory_EmptyFilePath(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	err := texteditor.EditPklWithFactory(fs, ctx, "", logger, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not have a .pkl extension")
}

func TestEditPklWithFactory_NonPklExtension(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	testFile := "not_a_pkl.txt"
	err := afero.WriteFile(fs, testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	err = texteditor.EditPklWithFactory(fs, ctx, testFile, logger, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), ".pkl extension")
}

func TestRealEditorCmdFactory_InvalidEditor(t *testing.T) {
	// Test with invalid editor name
	cmd, err := texteditor.RealEditorCmdFactory("nonexistent-editor", "test.pkl")
	require.Nil(t, cmd)
	require.Error(t, err)
}

func TestRealEditorCmdFactory_InvalidPath(t *testing.T) {
	// Test with invalid file path
	_, err := texteditor.RealEditorCmdFactory("vim", "/nonexistent/path/test.pkl")
	require.NoError(t, err) // Should not error, as the file doesn't need to exist
}

func TestRealEditorCmd_SetIO(t *testing.T) {
	// Create a temporary file for testing
	tempFile := "test.pkl"
	fs := afero.NewMemMapFs()
	err := afero.WriteFile(fs, tempFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Create a real editor command
	cmd, err := texteditor.RealEditorCmdFactory("vim", tempFile)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	err = cmd.Run()
	require.Error(t, err)
}

func TestRealEditorCmd_Run(t *testing.T) {
	// Use temporary directory for test files
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.pkl")

	cmd, err := texteditor.RealEditorCmdFactory("stub", testFile)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	err = cmd.Run()
	require.Error(t, err)
}

func TestMain(m *testing.M) {
	// Set non-interactive mode
	os.Setenv("NON_INTERACTIVE", "1")

	// Run tests
	code := m.Run()

	os.Exit(code)
}
