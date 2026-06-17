package http

import (
	"context"
	"errors"
	"log/slog"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPSchemeURL(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "http://127.0.0.1:8080", httpSchemeURL("127.0.0.1:8080"))
}

func TestWriteStatusOK(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	writeStatusOK(w)
	assert.Equal(t, stdhttp.StatusOK, w.Code)
}

func TestShutdownHTTPServerIfRunning_Nil(t *testing.T) {
	t.Parallel()
	err := shutdownHTTPServerIfRunning(context.Background(), nil, slog.Default(), "test")
	assert.NoError(t, err)
}

func TestLogInvalidTrustedProxies_Empty(t *testing.T) {
	t.Parallel()
	logInvalidTrustedProxies(slog.Default(), nil)
}

func TestLogInvalidTrustedProxies_NilLogger(t *testing.T) {
	t.Parallel()
	logInvalidTrustedProxies(nil, []string{"bad-entry"})
}

func TestLogManagementAPIError_NilLogger(t *testing.T) {
	t.Parallel()
	logManagementAPIError(nil, 500, "message")
}

func TestLogManagementAPIError_WithLogger(t *testing.T) {
	t.Parallel()
	logManagementAPIError(slog.Default(), stdhttp.StatusInternalServerError, "test error")
}

func TestLogHotReloadSetupWarning_NoPanic(t *testing.T) {
	t.Parallel()
	logHotReloadSetupWarning(slog.Default(), errors.New("setup failed"))
}

func TestLogWorkflowPathResolveWarning_NoPanic(t *testing.T) {
	t.Parallel()
	logWorkflowPathResolveWarning(slog.Default(), "/tmp/workflow.yaml", errors.New("resolve failed"))
}

func TestLogOptionalWatchFailure_NoPanic(t *testing.T) {
	t.Parallel()
	logOptionalWatchFailure(slog.Default(), "/tmp/resources", errors.New("watch failed"))
}

func TestLogUploadCleanupFailure_NoPanic(t *testing.T) {
	t.Parallel()
	logUploadCleanupFailure(slog.Default(), "upload-123", errors.New("cleanup failed"))
}
