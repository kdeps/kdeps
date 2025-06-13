package docker

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestPullModels_Error(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Provide some dummy model names; expect error as 'ollama' binary likely unavailable
	err := pullModels(ctx, []string{"nonexistent-model-1"}, logger)
	if err == nil {
		t.Fatalf("expected error when pulling models with missing binary")
	}
}
