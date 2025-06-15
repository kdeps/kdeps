package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/assert"
)

func TestGenerateParamsSectionAdditional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prefix   string
		items    map[string]string
		expected []string // substrings expected in result
	}{
		{
			name:   "with values",
			prefix: "ARG",
			items: map[string]string{
				"FOO": "bar",
				"BAZ": "",
			},
			expected: []string{`ARG FOO="bar"`, `ARG BAZ`},
		},
		{
			name:     "empty map",
			prefix:   "ENV",
			items:    map[string]string{},
			expected: []string{""},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := generateParamsSection(tc.prefix, tc.items)
			for _, want := range tc.expected {
				assert.Contains(t, got, want)
			}
		})
	}
}

func TestGenerateDockerfile_Minimal(t *testing.T) {
	t.Parallel()

	// Build a minimal Dockerfile using generateDockerfile. Only verify that
	// critical dynamic pieces make their way into the output template. A full
	// semantic diff is unnecessary and would be brittle.
	schemaVersion := schema.SchemaVersion(context.Background())

	df := generateDockerfile(
		"1.0",            // imageVersion
		schemaVersion,    // schemaVersion
		"127.0.0.1",      // hostIP
		"11435",          // ollamaPort
		"127.0.0.1:3000", // kdepsHost
		"",               // argsSection
		"",               // envsSection
		"",               // pkgSection
		"",               // pythonPkgSection
		"",               // condaPkgSection
		"2024.10-1",      // anacondaVersion
		"0.28.1",         // pklVersion
		"UTC",            // timezone
		"",               // exposedPort
		false,            // installAnaconda
		false,            // devBuildMode
		false,            // apiServerMode
		false,            // useLatest
	)

	// Quick smoke-test assertions.
	assert.Contains(t, df, "FROM ollama/ollama:1.0")
	assert.Contains(t, df, "ENV SCHEMA_VERSION="+schemaVersion)
	assert.Contains(t, df, "ENV KDEPS_HOST=127.0.0.1:3000")
	// No ports should be exposed because apiServerMode == false && exposedPort == ""
	assert.NotContains(t, df, "EXPOSE")
}

func TestPrintDockerBuildOutput_Extra(t *testing.T) {
	t.Parallel()

	// 1. Happy-path: mixed JSON stream lines and raw text.
	lines := []string{
		marshal(t, BuildLine{Stream: "Step 1/2 : FROM scratch\n"}),
		marshal(t, BuildLine{Stream: " ---> Using cache\n"}),
		"non-json-line should be echoed as-is", // raw
	}
	reader := bytes.NewBufferString(strings.Join(lines, "\n"))
	err := printDockerBuildOutput(reader)
	assert.NoError(t, err)

	// 2. Error path: JSON line with an error field should surface.
	errLines := []string{marshal(t, BuildLine{Error: "boom"})}
	errReader := bytes.NewBufferString(strings.Join(errLines, "\n"))
	err = printDockerBuildOutput(errReader)
	assert.ErrorContains(t, err, "boom")
}

// marshal is a tiny helper that converts a BuildLine to its JSON string
// representation and fails the test immediately upon error.
func marshal(t *testing.T, bl BuildLine) string {
	t.Helper()
	data, err := json.Marshal(bl)
	if err != nil {
		t.Fatalf("failed to marshal build line: %v", err)
	}
	return string(data)
}
