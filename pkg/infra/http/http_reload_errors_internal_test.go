package http

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHotReloadErrors_WrapMessage(t *testing.T) {
	t.Parallel()
	base := errors.New("base error")
	cases := []struct {
		fn      func(error) error
		wantMsg string
	}{
		{hotReloadWatchWorkflowFailed, "watch workflow"},
		{hotReloadPreprocessFailed, "preprocess"},
		{hotReloadParseFailed, "parse workflow"},
		{hotReloadResolvePathFailed, "resolve workflow path"},
		{workflowParserSchemaValidatorFailed, "schema validator"},
		{uploadInfrastructureCreateFailed, "file store"},
		{managementMkdirWorkflowDirFailed, "workflow directory"},
		{packageInvalidGzipError, "gzip"},
		{packageResolveDestDirFailed, "destination directory"},
	}
	for _, tc := range cases {
		err := tc.fn(base)
		assert.Error(t, err, "fn for %q should return error", tc.wantMsg)
		assert.Contains(t, err.Error(), "base error", "should wrap original error")
	}
}
