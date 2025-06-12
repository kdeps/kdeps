package resolver

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestValidateAndEnsureResponseFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &DependencyResolver{
		Fs:                 fs,
		ResponsePklFile:    "/tmp/response.pkl",
		ResponseTargetFile: "/tmp/response.json",
		Logger:             logging.NewTestLogger(),
		Context:            context.Background(),
	}

	t.Run("ValidatePKLExtension_Success", func(t *testing.T) {
		require.NoError(t, dr.validatePklFileExtension())
	})

	t.Run("ValidatePKLExtension_Error", func(t *testing.T) {
		bad := &DependencyResolver{ResponsePklFile: "/tmp/file.txt"}
		err := bad.validatePklFileExtension()
		require.Error(t, err)
	})

	t.Run("EnsureTargetFileRemoved", func(t *testing.T) {
		// create the target file
		require.NoError(t, afero.WriteFile(fs, dr.ResponseTargetFile, []byte("x"), 0o644))
		// file should exist
		exists, _ := afero.Exists(fs, dr.ResponseTargetFile)
		require.True(t, exists)
		// call
		require.NoError(t, dr.ensureResponseTargetFileNotExists())
		// after call file should be gone
		exists, _ = afero.Exists(fs, dr.ResponseTargetFile)
		require.False(t, exists)
	})
}
