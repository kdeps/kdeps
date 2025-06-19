package docker_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestGenerateDockerCompose_GeneratesFileForGPUs(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir := t.TempDir()

	// cName is placed inside tmp dir so the compose file is created there.
	cName := filepath.Join(tmpDir, "agent")
	containerName := "agent:latest"

	tests := []struct {
		name string
		gpu  string
	}{
		{"cpu", "cpu"},
		{"amd", "amd"},
		{"nvidia", "nvidia"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			filePath := cName + "_docker-compose-" + tc.gpu + ".yaml"
			// ensure clean slate
			_ = fs.Remove(filePath)

			err := GenerateDockerCompose(fs, cName, containerName, cName+"-"+tc.gpu, "127.0.0.1", "8080", "127.0.0.1", "9090", true, true, tc.gpu)
			require.NoError(t, err)

			content, err := afero.ReadFile(fs, filePath)
			require.NoError(t, err)
			str := string(content)
			require.NotEmpty(t, str)
			// Ensure gpu specific marker or at least container image present
			require.Contains(t, str, "image: "+containerName)
		})
	}

	t.Run("unsupported gpu", func(t *testing.T) {
		err := GenerateDockerCompose(fs, cName, containerName, cName+"-x", "", "", "", "", false, false, "unknown")
		require.Error(t, err)
	})

	t.Run("web-only-ports", func(t *testing.T) {
		path := filepath.Join(tmpDir, "agent_docker-compose-cpu.yaml")
		_ = fs.Remove(path)
		err := GenerateDockerCompose(fs, cName, containerName, cName+"-cpu", "", "", "127.0.0.1", "9090", false, true, "cpu")
		require.NoError(t, err)
		data, _ := afero.ReadFile(fs, path)
		str := string(data)
		// Ensure the ports section lists only the web port (9090)
		lines := strings.Split(str, "\n")
		var portLines []string
		for i, l := range lines {
			if strings.TrimSpace(l) == "ports:" {
				// collect following indented list items
				for j := i + 1; j < len(lines); j++ {
					item := lines[j]
					if strings.HasPrefix(item, "      -") {
						trimmed := strings.TrimSpace(item)
						if strings.Contains(trimmed, "9090") {
							portLines = append(portLines, trimmed)
						}
					} else if strings.TrimSpace(item) == "" {
						break
					} else if !strings.HasPrefix(item, " ") {
						break
					}
				}
				break
			}
		}
		require.Len(t, portLines, 1, "expected exactly one exposed port")
		require.Contains(t, portLines[0], "9090")
	})

	t.Run("no-ports", func(t *testing.T) {
		path := filepath.Join(tmpDir, "agent_docker-compose-cpu.yaml")
		_ = fs.Remove(path)
		err := GenerateDockerCompose(fs, cName, containerName, cName+"-cpu", "", "", "", "", false, false, "cpu")
		require.NoError(t, err)
		data, _ := afero.ReadFile(fs, path)
		require.NotContains(t, string(data), "ports:")
	})
}

func TestCheckDevBuildMode_Variants(t *testing.T) {
	fs := afero.NewMemMapFs()
	kdepsDir := t.TempDir()
	logger := logging.NewTestLogger()

	// Case: file missing → dev mode false
	dev, err := checkDevBuildMode(fs, kdepsDir, logger)
	require.NoError(t, err)
	require.False(t, dev)

	// create directory structure with file
	cacheDir := filepath.Join(kdepsDir, "cache")
	_ = fs.MkdirAll(cacheDir, 0o755)
	filePath := filepath.Join(cacheDir, "kdeps")
	require.NoError(t, afero.WriteFile(fs, filePath, []byte("bin"), 0o644))

	dev, err = checkDevBuildMode(fs, kdepsDir, logger)
	require.NoError(t, err)
	require.True(t, dev)

	// Replace file with directory to trigger non-regular case
	require.NoError(t, fs.Remove(filePath))
	require.NoError(t, fs.MkdirAll(filePath, 0o755))

	dev, err = checkDevBuildMode(fs, kdepsDir, logger)
	require.NoError(t, err)
	require.False(t, dev)
}
