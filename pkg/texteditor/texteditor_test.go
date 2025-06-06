package texteditor

import (
	"context"
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditPkl(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	t.Run("InvalidFileExtension", func(t *testing.T) {
		t.Parallel()

		filePath := "/test/file.txt"

		err := EditPkl(fs, ctx, filePath, logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not have a .pkl extension")
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		t.Parallel()

		filePath := "/test/nonexistent.pkl"

		err := EditPkl(fs, ctx, filePath, logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("StatError", func(t *testing.T) {
		t.Parallel()

		// Use a read-only filesystem that will cause stat to fail
		readOnlyFs := afero.NewReadOnlyFs(afero.NewMemMapFs())
		filePath := "/test/file.pkl"

		err := EditPkl(readOnlyFs, ctx, filePath, logger)

		assert.Error(t, err)
		// Should fail with an error (not necessarily "does not exist" since it's a different error)
		assert.True(t, err != nil)
	})

	t.Run("ValidFileButEditorCommandFails", func(t *testing.T) {
		t.Parallel()

		// Create a valid .pkl file
		filePath := "/test/valid.pkl"
		content := "test content"

		err := afero.WriteFile(fs, filePath, []byte(content), 0644)
		require.NoError(t, err)

		// This will fail at the editor command execution stage since "kdeps" editor command doesn't exist
		// but we can validate that it gets past the validation checks
		err = EditPkl(fs, ctx, filePath, logger)

		// The error should be related to editor command execution, not validation
		assert.Error(t, err)
		// Should get to the editor command stage, not fail on validation
		assert.Contains(t, err.Error(), "editor command failed")
	})

	t.Run("FileExtensionValidation", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name     string
			filePath string
			isValid  bool
		}{
			{"ValidPklFile", "/test/file.pkl", true},
			{"UppercasePkl", "/test/file.PKL", false}, // Extension check is case-sensitive
			{"TxtFile", "/test/file.txt", false},
			{"NoExtension", "/test/file", false},
			{"EmptyExtension", "/test/file.", false},
			{"MultipleExtensions", "/test/file.pkl.backup", false},
		}

		for _, tc := range testCases {
			tc := tc // Capture loop variable
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				if tc.isValid {
					// For valid files, create them first
					err := afero.WriteFile(fs, tc.filePath, []byte("test content"), 0644)
					require.NoError(t, err)

					err = EditPkl(fs, ctx, tc.filePath, logger)
					// Should fail at editor command execution stage, not extension validation
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "editor command failed")
				} else {
					// For invalid extensions, should fail at extension check
					err := EditPkl(fs, ctx, tc.filePath, logger)
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "does not have a .pkl extension")
				}
			})
		}
	})

	t.Run("DifferentFilesystemTypes", func(t *testing.T) {
		t.Parallel()

		// Test with OsFs (real filesystem) but with a non-existent file
		osFs := afero.NewOsFs()
		filePath := "/tmp/nonexistent_test_file.pkl"

		// Make sure the file doesn't exist
		_, err := os.Stat(filePath)
		if err == nil {
			// File exists, remove it for test
			os.Remove(filePath)
		}

		err = EditPkl(osFs, ctx, filePath, logger)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})
}
