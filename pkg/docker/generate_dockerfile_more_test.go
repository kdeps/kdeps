package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGenerateDockerfileAdditionalCases exercises seldom-hit branches in generateDockerfile so that
// coverage reflects real-world usage scenarios.
func TestGenerateDockerfileAdditionalCases(t *testing.T) {
	t.Run("DevBuildModeWithLatestAndExpose", func(t *testing.T) {
		result := generateDockerfile(
			"v1.2.3",                      // imageVersion
			"2.0",                         // schemaVersion
			"0.0.0.0",                     // hostIP
			"9999",                        // ollamaPortNum
			"kdeps.example",               // kdepsHost
			"ARG SAMPLE=1",                // argsSection
			"ENV FOO=bar",                 // envsSection
			"RUN apt-get -y install curl", // pkgSection
			"RUN pip install pytest",      // pythonPkgSection
			"",                            // condaPkgSection (none)
			"2024.10-1",                   // anacondaVersion (overwritten by useLatest=true below)
			"0.28.1",                      // pklVersion   (ditto)
			"UTC",                         // timezone
			"8080",                        // exposedPort
			true,                          // installAnaconda
			true,                          // devBuildMode  – should copy local kdeps binary
			true,                          // apiServerMode – should add EXPOSE line
			true,                          // useLatest     – should convert version marks to "latest"
		)

		// Ensure dev build mode path is present.
		assert.Contains(t, result, "cp /cache/kdeps /bin/kdeps", "expected dev build mode copy command")
		// When useLatest==true we expect the placeholder 'latest' to appear in pkl download section.
		assert.Contains(t, result, "pkl-linux-latest", "expected latest pkl artifact reference")
		// installAnaconda==true should result in anaconda installer copy logic.
		assert.Contains(t, result, "anaconda-linux-latest", "expected latest anaconda artifact reference")
		// apiServerMode==true adds an EXPOSE directive for provided port(s).
		assert.Contains(t, result, "EXPOSE 8080", "expected expose directive present")
	})

	t.Run("NonDevNoAnaconda", func(t *testing.T) {
		result := generateDockerfile(
			"stable",    // imageVersion
			"1.1",       // schemaVersion
			"127.0.0.1", // hostIP
			"1234",      // ollamaPortNum
			"host:1234", // kdepsHost
			"",          // argsSection
			"",          // envsSection
			"",          // pkgSection
			"",          // pythonPkgSection
			"",          // condaPkgSection
			"2024.10-1", // anacondaVersion
			"0.28.1",    // pklVersion
			"UTC",       // timezone
			"",          // exposedPort (no api server)
			false,       // installAnaconda
			false,       // devBuildMode
			false,       // apiServerMode – no EXPOSE
			false,       // useLatest
		)

		// Non-dev build should use install script instead of local binary.
		assert.Contains(t, result, "raw.githubusercontent.com/kdeps/kdeps", "expected remote install script usage")
		// Should NOT contain cp of anaconda because installAnaconda==false.
		assert.NotContains(t, result, "anaconda-linux", "unexpected anaconda installation commands present")
		// Should not contain EXPOSE directive.
		assert.NotContains(t, result, "EXPOSE", "unexpected expose directive present")
	})
}
