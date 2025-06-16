package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewAddCommand_MetadataAndArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	cmd := NewAddCommand(fs, ctx, "/tmp/kdeps", logging.NewTestLogger())

	assert.Equal(t, "install [package]", cmd.Use)
	assert.Contains(t, cmd.Short, "Install")

	// missing arg should error
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}
