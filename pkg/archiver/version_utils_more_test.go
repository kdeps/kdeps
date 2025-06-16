package archiver

import (
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCompareVersionsAndGetLatest(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("compareVersions", func(t *testing.T) {
		versions := []string{"1.0.0", "2.3.4", "2.10.0", "0.9.9"}
		latest := compareVersions(versions, logger)
		assert.Equal(t, "2.3.4", latest)
	})

	t.Run("GetLatestVersion", func(t *testing.T) {
		fs := afero.NewOsFs()
		tmpDir := t.TempDir()
		logger := logging.NewTestLogger()

		// create version dirs
		for _, v := range []string{"0.1.0", "1.2.3", "1.2.10"} {
			assert.NoError(t, fs.MkdirAll(filepath.Join(tmpDir, v), 0o755))
		}
		latest, err := GetLatestVersion(tmpDir, logger)
		assert.NoError(t, err)
		assert.Equal(t, "1.2.3", latest)

		emptyDir := filepath.Join(tmpDir, "empty")
		assert.NoError(t, fs.MkdirAll(emptyDir, 0o755))
		_, err = GetLatestVersion(emptyDir, logger)
		assert.Error(t, err)
	})
}
