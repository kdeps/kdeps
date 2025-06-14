package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewPackageCommand_MetadataAndArgs(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}

	cmd := NewPackageCommand(fs, ctx, "/tmp/kdeps", env, logging.NewTestLogger())

	assert.Equal(t, "package [agent-dir]", cmd.Use)
	assert.Contains(t, strings.ToLower(cmd.Short), "package")

	// Execute with no args – expect error
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing args")
	}
}
