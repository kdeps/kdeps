package docker

import "testing"

// TestGenerateDockerfileBranches exercises multiple flag combinations to hit
// the majority of conditional paths in generateDockerfile. We don't validate
// the entire output – only presence of a few sentinel strings that should
// appear when the corresponding branch executes. This drives a large number
// of statements for coverage without any external I/O.
func TestGenerateDockerfileBranches(t *testing.T) {
	baseArgs := struct {
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
	}{
		imageVersion:     "1.0",
		schemaVersion:    "v0",
		hostIP:           "0.0.0.0",
		ollamaPortNum:    "11434",
		kdepsHost:        "localhost",
		argsSection:      "ARG FOO=bar",
		envsSection:      "ENV BAR=baz",
		pkgSection:       "RUN echo pkgs",
		pythonPkgSection: "RUN echo py",
		condaPkgSection:  "RUN echo conda",
		anacondaVersion:  "2024.09-1",
		pklVersion:       "0.25.0",
		timezone:         "Etc/UTC",
		exposedPort:      "5000",
	}

	combos := []struct {
		installAnaconda bool
		devBuildMode    bool
		apiServerMode   bool
		useLatest       bool
		expectStrings   []string
	}{
		{false, false, false, false, []string{"ENV BAR=baz", "RUN echo pkgs"}},
		{true, false, false, false, []string{"/tmp/anaconda.sh", "RUN /bin/bash /tmp/anaconda.sh"}},
		{false, true, false, false, []string{"/cache/kdeps", "chmod a+x /bin/kdeps"}},
		{false, false, true, false, []string{"EXPOSE 5000"}},
		{true, true, true, true, []string{"latest", "cp /cache/pkl-linux-latest-amd64"}},
	}

	for i, c := range combos {
		df := generateDockerfile(
			baseArgs.imageVersion,
			baseArgs.schemaVersion,
			baseArgs.hostIP,
			baseArgs.ollamaPortNum,
			baseArgs.kdepsHost,
			baseArgs.argsSection,
			baseArgs.envsSection,
			baseArgs.pkgSection,
			baseArgs.pythonPkgSection,
			baseArgs.condaPkgSection,
			baseArgs.anacondaVersion,
			baseArgs.pklVersion,
			baseArgs.timezone,
			baseArgs.exposedPort,
			c.installAnaconda,
			c.devBuildMode,
			c.apiServerMode,
			c.useLatest,
		)
		for _, s := range c.expectStrings {
			if !strContains(df, s) {
				t.Fatalf("combo %d expected substring %q not found", i, s)
			}
		}
	}
}

// tiny helper – strings.Contains without importing strings multiple times.
func strContains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) &&
		func() bool {
			for i := 0; i+len(needle) <= len(haystack); i++ {
				if haystack[i:i+len(needle)] == needle {
					return true
				}
			}
			return false
		}())
}
