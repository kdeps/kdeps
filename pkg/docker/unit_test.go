package docker_test

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestGenerateParamsSection_Unit tests the GenerateParamsSection function
func TestGenerateParamsSection_Unit(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		items    map[string]string
		expected []string // We'll check that all these strings are present
	}{
		{
			name:   "ARG parameters",
			prefix: "ARG",
			items: map[string]string{
				"NODE_ENV": "production",
				"VERSION":  "1.0.0",
				"DEBUG":    "",
			},
			expected: []string{
				"ARG NODE_ENV=\"production\"",
				"ARG VERSION=\"1.0.0\"",
				"ARG DEBUG",
			},
		},
		{
			name:   "ENV parameters",
			prefix: "ENV",
			items: map[string]string{
				"PATH":    "/usr/local/bin",
				"WORKDIR": "/app",
			},
			expected: []string{
				"ENV PATH=\"/usr/local/bin\"",
				"ENV WORKDIR=\"/app\"",
			},
		},
		{
			name:     "Empty items",
			prefix:   "ARG",
			items:    map[string]string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateParamsSection(tt.prefix, tt.items)

			for _, expected := range tt.expected {
				if expected == "" && len(tt.items) == 0 {
					assert.Empty(t, result)
				} else {
					assert.Contains(t, result, expected)
				}
			}
		})
	}
}

// TestGenerateDockerfile_Unit tests the GenerateDockerfile function
func TestGenerateDockerfile_Unit(t *testing.T) {
	tests := []struct {
		name             string
		imageVersion     string
		schemaVersion    string
		hostIP           string
		ollamaPortNum    string
		kdepsHost        string
		argsSection      string
		envsSection      string
		pkgSection       string
		pythonPkgSection string
		condaPkgSection  string
		anacondaVersion  string
		pklVersion       string
		timezone         string
		exposedPort      string
		installAnaconda  bool
		devBuildMode     bool
		apiServerMode    bool
		useLatest        bool
		expectContains   []string
	}{
		{
			name:             "Basic configuration",
			imageVersion:     "latest",
			schemaVersion:    "1.0.0",
			hostIP:           "127.0.0.1",
			ollamaPortNum:    "11434",
			kdepsHost:        "127.0.0.1:3000",
			argsSection:      "ARG TEST_ARG=value",
			envsSection:      "ENV TEST_ENV=value",
			pkgSection:       "RUN apt-get install -y curl",
			pythonPkgSection: "RUN pip install numpy",
			condaPkgSection:  "RUN conda install pandas",
			anacondaVersion:  "2024.10-1",
			pklVersion:       "0.28.1",
			timezone:         "UTC",
			exposedPort:      "3000",
			installAnaconda:  true,
			devBuildMode:     false,
			apiServerMode:    true,
			useLatest:        false,
			expectContains: []string{
				"FROM ollama/ollama:latest",
				"ENV SCHEMA_VERSION=1.0.0",
				"ENV OLLAMA_HOST=127.0.0.1:11434",
				"ENV KDEPS_HOST=127.0.0.1:3000",
				"ARG TEST_ARG=value",
				"ENV TEST_ENV=value",
				"COPY cache /cache",
				"ENV TZ=UTC",
				"RUN apt-get install -y curl",
				"RUN pip install numpy",
				"RUN conda install pandas",
				"EXPOSE 3000",
				"ENTRYPOINT [\"/bin/kdeps\"]",
			},
		},
		{
			name:             "Dev build mode without Anaconda",
			imageVersion:     "latest",
			schemaVersion:    "1.0.0",
			hostIP:           "0.0.0.0",
			ollamaPortNum:    "11434",
			kdepsHost:        "0.0.0.0:8080",
			argsSection:      "",
			envsSection:      "",
			pkgSection:       "",
			pythonPkgSection: "",
			condaPkgSection:  "",
			anacondaVersion:  "2024.10-1",
			pklVersion:       "0.28.1",
			timezone:         "America/New_York",
			exposedPort:      "",
			installAnaconda:  false,
			devBuildMode:     true,
			apiServerMode:    false,
			useLatest:        false,
			expectContains: []string{
				"FROM ollama/ollama:latest",
				"ENV TZ=America/New_York",
				"RUN cp /cache/kdeps /bin/kdeps", // Dev build mode
				"COPY workflow /agent/project",
			},
		},
		{
			name:             "Use latest versions",
			imageVersion:     "latest",
			schemaVersion:    "1.0.0",
			hostIP:           "127.0.0.1",
			ollamaPortNum:    "11434",
			kdepsHost:        "127.0.0.1:3000",
			argsSection:      "",
			envsSection:      "",
			pkgSection:       "",
			pythonPkgSection: "",
			condaPkgSection:  "",
			anacondaVersion:  "2024.10-1",
			pklVersion:       "0.28.1",
			timezone:         "UTC",
			exposedPort:      "",
			installAnaconda:  false,
			devBuildMode:     false,
			apiServerMode:    false,
			useLatest:        true, // This should override versions
			expectContains: []string{
				"pkl-linux-latest-amd64",
				"pkl-linux-latest-aarch64",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateDockerfile(
				tt.imageVersion,
				tt.schemaVersion,
				tt.hostIP,
				tt.ollamaPortNum,
				tt.kdepsHost,
				tt.argsSection,
				tt.envsSection,
				tt.pkgSection,
				tt.pythonPkgSection,
				tt.condaPkgSection,
				tt.anacondaVersion,
				tt.pklVersion,
				tt.timezone,
				tt.exposedPort,
				tt.installAnaconda,
				tt.devBuildMode,
				tt.apiServerMode,
				tt.useLatest,
			)

			assert.NotEmpty(t, result)

			for _, expected := range tt.expectContains {
				assert.Contains(t, result, expected, "Expected content not found: %s", expected)
			}
		})
	}
}

// TestCheckDevBuildMode_Unit tests the CheckDevBuildMode function
func TestCheckDevBuildMode_Unit(t *testing.T) {
	tests := []struct {
		name        string
		setupFS     func(fs afero.Fs, kdepsDir string)
		expectMode  bool
		expectError bool
	}{
		{
			name: "Kdeps binary exists",
			setupFS: func(fs afero.Fs, kdepsDir string) {
				downloadDir := filepath.Join(kdepsDir, "cache")
				fs.MkdirAll(downloadDir, 0o755)
				afero.WriteFile(fs, filepath.Join(downloadDir, "kdeps"), []byte("binary"), 0o755)
			},
			expectMode:  true,
			expectError: false,
		},
		{
			name: "Kdeps binary doesn't exist",
			setupFS: func(fs afero.Fs, kdepsDir string) {
				fs.MkdirAll(filepath.Join(kdepsDir, "cache"), 0o755)
				// Don't create the kdeps file
			},
			expectMode:  false,
			expectError: false,
		},
		{
			name: "Kdeps path is directory not file",
			setupFS: func(fs afero.Fs, kdepsDir string) {
				kdepsPath := filepath.Join(kdepsDir, "cache", "kdeps")
				fs.MkdirAll(kdepsPath, 0o755) // Create as directory
			},
			expectMode:  false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			logger := logging.NewTestLogger()
			kdepsDir := "/tmp/kdeps"

			tt.setupFS(fs, kdepsDir)

			mode, err := CheckDevBuildMode(fs, kdepsDir, logger)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectMode, mode)
		})
	}
}

// TestCopyFilesToRunDir_Unit tests the CopyFilesToRunDir function
func TestCopyFilesToRunDir_Unit(t *testing.T) {
	tests := []struct {
		name        string
		setupFS     func(fs afero.Fs, downloadDir string)
		expectError bool
		expectFiles []string
	}{
		{
			name: "Successful copy",
			setupFS: func(fs afero.Fs, downloadDir string) {
				fs.MkdirAll(downloadDir, 0o755)
				afero.WriteFile(fs, filepath.Join(downloadDir, "file1.txt"), []byte("content1"), 0o644)
				afero.WriteFile(fs, filepath.Join(downloadDir, "file2.txt"), []byte("content2"), 0o644)
			},
			expectError: false,
			expectFiles: []string{"file1.txt", "file2.txt"},
		},
		{
			name: "Download directory doesn't exist",
			setupFS: func(fs afero.Fs, downloadDir string) {
				// Don't create the directory
			},
			expectError: true,
			expectFiles: []string{},
		},
		{
			name: "Empty download directory",
			setupFS: func(fs afero.Fs, downloadDir string) {
				fs.MkdirAll(downloadDir, 0o755)
				// Create empty directory
			},
			expectError: false,
			expectFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			ctx := context.Background()
			logger := logging.NewTestLogger()

			downloadDir := "/tmp/downloads"
			runDir := "/tmp/run"

			tt.setupFS(fs, downloadDir)

			err := CopyFilesToRunDir(fs, ctx, downloadDir, runDir, logger)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check that cache directory was created
				cacheDir := filepath.Join(runDir, "cache")
				exists, _ := afero.DirExists(fs, cacheDir)
				assert.True(t, exists)

				// Check that expected files exist
				for _, expectedFile := range tt.expectFiles {
					expectedPath := filepath.Join(cacheDir, expectedFile)
					exists, _ := afero.Exists(fs, expectedPath)
					assert.True(t, exists, "Expected file %s to exist", expectedPath)
				}
			}
		})
	}
}
