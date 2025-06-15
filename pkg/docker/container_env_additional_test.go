package docker

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestLoadEnvFile_VariousCases(t *testing.T) {
	fs := afero.NewOsFs()
	dir := t.TempDir()

	t.Run("file-missing", func(t *testing.T) {
		envs, err := loadEnvFile(fs, filepath.Join(dir, "missing.env"))
		require.NoError(t, err)
		require.Nil(t, envs)
	})

	t.Run("valid-env", func(t *testing.T) {
		path := filepath.Join(dir, "good.env")
		content := "FOO=bar\nHELLO=world\n"
		require.NoError(t, afero.WriteFile(fs, path, []byte(content), 0o644))

		envs, err := loadEnvFile(fs, path)
		require.NoError(t, err)
		// Convert slice to joined string for easier contains checks irrespective of order.
		joined := strings.Join(envs, ",")
		require.Contains(t, joined, "FOO=bar")
		require.Contains(t, joined, "HELLO=world")
	})
}
