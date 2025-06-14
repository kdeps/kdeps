package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// captureOutput redirects stdout to a buffer and returns a restore func along
// with the buffer pointer.
func captureOutput() (*bytes.Buffer, func()) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	buf := &bytes.Buffer{}
	done := make(chan struct{})

	go func() {
		_, _ = io.Copy(buf, r)
		close(done)
	}()

	restore := func() {
		w.Close()
		<-done
		os.Stdout = old
	}
	return buf, restore
}

// TestScaffoldCommand_Happy creates two valid resources and asserts files are
// written under the expected paths.
func TestScaffoldCommand_Happy(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(fs, ctx, logger)

	agent := "myagent"
	args := []string{agent, "client", "exec"}

	// Capture output just in case (not strictly needed but keeps test quiet).
	_, restore := captureOutput()
	defer restore()

	cmd.Run(cmd, args)

	// Verify generated files exist.
	expected := []string{
		agent + "/resources/client.pkl",
		agent + "/resources/exec.pkl",
	}
	for _, path := range expected {
		if ok, _ := afero.Exists(fs, path); !ok {
			t.Fatalf("expected file %s to exist", path)
		}
	}

	_ = schema.SchemaVersion(ctx)
}

// TestScaffoldCommand_InvalidResource ensures invalid names are reported and
// not created.
func TestScaffoldCommand_InvalidResource(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	cmd := NewScaffoldCommand(fs, ctx, logger)
	agent := "badagent"

	buf, restore := captureOutput()
	defer restore()

	cmd.Run(cmd, []string{agent, "bogus"})

	// The bogus file should not be created.
	if ok, _ := afero.Exists(fs, agent+"/resources/bogus.pkl"); ok {
		t.Fatalf("unexpected file created for invalid resource")
	}

	_ = buf // output not asserted; just ensuring no panic

	_ = schema.SchemaVersion(ctx)
}
