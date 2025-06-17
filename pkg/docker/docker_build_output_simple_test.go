package docker

import (
	"bytes"
	"testing"
)

func TestPrintDockerBuildOutputSimple(t *testing.T) {
	successLog := bytes.NewBufferString(`{"stream":"Step 1/2 : FROM alpine\n"}\n{"stream":" ---> 123abc\n"}\n`)
	if err := printDockerBuildOutput(successLog); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Error case should propagate the message
	errBuf := bytes.NewBufferString(`{"error":"build failed"}`)
	if err := printDockerBuildOutput(errBuf); err == nil {
		t.Fatalf("expected error not returned")
	}
}
