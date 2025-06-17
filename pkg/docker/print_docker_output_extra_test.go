package docker

import (
	"strings"
	"testing"
)

func TestPrintDockerBuildOutputSuccess(t *testing.T) {
	logs := `{"stream":"Step 1/2 : FROM alpine\n"}\n{"stream":" ---\u003e 123abc\n"}\n`
	if err := printDockerBuildOutput(strings.NewReader(logs)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestPrintDockerBuildOutputError(t *testing.T) {
	logs := `{"error":"something bad"}`
	if err := printDockerBuildOutput(strings.NewReader(logs)); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
