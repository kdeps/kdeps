package resource

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestLoadResource_Error(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewTestLogger()
	_, err := LoadResource(ctx, "/tmp/doesnotexist.pkl", logger)
	assert.Error(t, err)
}
