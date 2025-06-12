package resolver

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
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

func TestValidatePklFileExtension_Response(t *testing.T) {
	dr := &DependencyResolver{ResponsePklFile: "resp.pkl"}
	if err := dr.validatePklFileExtension(); err != nil {
		t.Errorf("expected .pkl to validate, got %v", err)
	}
	dr.ResponsePklFile = "bad.txt"
	if err := dr.validatePklFileExtension(); err == nil {
		t.Errorf("expected error for non-pkl extension")
	}
}

func TestDecodeErrorMessage_Handler(t *testing.T) {
	logger := logging.GetLogger()
	plain := "hello"
	enc := utils.EncodeValue(plain)
	if got := decodeErrorMessage(enc, logger); got != plain {
		t.Errorf("expected decoded value, got %s", got)
	}
	// non-base64 string passes through
	if got := decodeErrorMessage("not-encoded", logger); got != "not-encoded" {
		t.Errorf("expected passthrough, got %s", got)
	}
}
