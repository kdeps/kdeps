package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOnDemandWriter_FollowsRedirectedStderr(t *testing.T) {
	w := onDemandWriter(os.Stderr)

	orig := os.Stderr
	r, pw, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = pw                         //nolint:reassign // simulates the full-screen view taking stderr
	t.Cleanup(func() { os.Stderr = orig }) //nolint:reassign // restore

	_, err = w.Write([]byte("Installed x 1.0.0 (first use).\n"))
	require.NoError(t, err)
	require.NoError(t, pw.Close())
	var got bytes.Buffer
	_, _ = got.ReadFrom(r)
	assert.Equal(t, "Installed x 1.0.0 (first use).\n", got.String(),
		"the write reaches the stderr in place at write time")
}

func TestOnDemandWriter_KeepsOtherWriters(t *testing.T) {
	var buf bytes.Buffer
	assert.Same(t, &buf, onDemandWriter(&buf))
}
