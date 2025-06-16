package docker

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintDockerBuildOutput_Success(t *testing.T) {
	logs := []string{
		`{"stream":"Step 1/3 : FROM alpine"}`,
		`{"stream":" ---\u003e a0d0a0d0a0d0"}`,
		`{"stream":"Successfully built"}`,
	}
	rd := strings.NewReader(strings.Join(logs, "\n"))

	err := printDockerBuildOutput(rd)
	assert.NoError(t, err)
}

func TestPrintDockerBuildOutput_Error(t *testing.T) {
	logs := []string{
		`{"stream":"Step 1/1 : FROM alpine"}`,
		`{"error":"some docker build error"}`,
	}
	rd := strings.NewReader(strings.Join(logs, "\n"))

	err := printDockerBuildOutput(rd)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "some docker build error")
}

func TestPrintDockerBuildOutput_NonJSONLines(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("non json line\n")
	buf.WriteString("{\"stream\":\"ok\"}\n")
	buf.WriteString("another bad line\n")

	err := printDockerBuildOutput(&buf)
	assert.NoError(t, err)
}
