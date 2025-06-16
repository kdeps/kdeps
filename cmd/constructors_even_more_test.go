package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCommandConstructorsUseStrings(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	dir := t.TempDir()
	logger := logging.NewTestLogger()

	constructors := []struct {
		name string
		cmd  func() string
	}{
		{"build", func() string { return NewBuildCommand(fs, ctx, dir, nil, logger).Use }},
		{"new", func() string { return NewAgentCommand(fs, ctx, dir, logger).Use }},
		{"package", func() string { return NewPackageCommand(fs, ctx, dir, nil, logger).Use }},
		{"run", func() string { return NewRunCommand(fs, ctx, dir, nil, logger).Use }},
		{"scaffold", func() string { return NewScaffoldCommand(fs, ctx, logger).Use }},
	}

	for _, c := range constructors {
		use := c.cmd()
		assert.NotEmpty(t, use, c.name)
	}
}
