package docker

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestStartOllamaServerStubbed(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Function should return immediately and not panic.
	startOllamaServer(ctx, logger)
}
