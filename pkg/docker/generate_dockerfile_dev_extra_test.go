package docker

import (
	"strings"
	"testing"
)

func TestGenerateDockerfile_DevBuildAndAPIServer(t *testing.T) {
	df := generateDockerfile(
		"1.2.3",              // image version
		"2.0",                // schema version
		"0.0.0.0",            // host IP
		"11434",              // ollama port
		"0.0.0.0:11434",      // kdeps host
		"ARG SAMPLE=1",       // args section
		"ENV FOO=bar",        // envs section
		"RUN apt-get update", // pkg section
		"RUN pip install x",  // python section
		"",                   // conda pkg section
		"2024.01-1",          // anaconda version
		"0.28.1",             // pkl version
		"UTC",                // timezone
		"8080",               // expose port
		false,                // installAnaconda
		true,                 // devBuildMode (exercise branch)
		true,                 // apiServerMode (expose port branch)
		false,                // useLatest
	)

	if !has(df, "cp /cache/kdeps /bin/kdeps") {
		t.Fatalf("expected dev build copy line")
	}
	if !has(df, "EXPOSE 8080") {
		t.Fatalf("expected expose port line")
	}
}

// small helper to avoid importing strings each time
func has(haystack, needle string) bool { return strings.Contains(haystack, needle) }
