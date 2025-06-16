package docker

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateDockerfileContent(t *testing.T) {
	df := generateDockerfile(
		"10.1",          // imageVersion
		"v1",            // schemaVersion
		"127.0.0.1",     // hostIP
		"8000",          // ollamaPortNum
		"localhost",     // kdepsHost
		"ARG FOO=bar",   // argsSection
		"ENV BAR=baz",   // envsSection
		"# pkg section", // pkgSection
		"# python pkgs", // pythonPkgSection
		"# conda pkgs",  // condaPkgSection
		"2024.10-1",     // anacondaVersion
		"0.28.1",        // pklVersion
		"UTC",           // timezone
		"8080",          // exposedPort
		true,            // installAnaconda
		true,            // devBuildMode
		true,            // apiServerMode
		false,           // useLatest
	)

	// basic sanity checks on returned content
	assert.True(t, strings.Contains(df, "FROM ollama/ollama:10.1"))
	assert.True(t, strings.Contains(df, "ENV SCHEMA_VERSION=v1"))
	assert.True(t, strings.Contains(df, "EXPOSE 8080"))
	assert.True(t, strings.Contains(df, "ARG FOO=bar"))
	assert.True(t, strings.Contains(df, "ENV BAR=baz"))
}

// TestGenerateDockerfileBranchCoverage exercises additional parameter combinations
func TestGenerateDockerfileBranchCoverage(t *testing.T) {
	combos := []struct {
		installAnaconda bool
		devBuildMode    bool
		apiServerMode   bool
		useLatest       bool
	}{
		{false, false, false, true},
		{true, false, true, true},
		{false, true, false, false},
	}

	for _, c := range combos {
		df := generateDockerfile(
			"10.1",
			"v1",
			"127.0.0.1",
			"8000",
			"localhost",
			"",
			"",
			"",
			"",
			"",
			"2024.10-1",
			"0.28.1",
			"UTC",
			"8080",
			c.installAnaconda,
			c.devBuildMode,
			c.apiServerMode,
			c.useLatest,
		)
		// simple assertion to ensure function returns non-empty string
		assert.NotEmpty(t, df)
	}
}
