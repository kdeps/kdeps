package environment

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestNewEnvironment_UnmarshalError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Set TIMEOUT to a non-numeric value so parsing into int fails.
	t.Setenv("TIMEOUT", "notanumber")
	t.Setenv("PWD", "/tmp")
	t.Setenv("ROOT_DIR", "/")
	t.Setenv("HOME", "/tmp")

	env, err := NewEnvironment(fs, nil)

	assert.Error(t, err)
	assert.Nil(t, env)
}
