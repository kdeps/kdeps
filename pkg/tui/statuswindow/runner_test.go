package statuswindow

import (
	"bytes"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// syncBuffer is a bytes.Buffer the program writes to from its own goroutine.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestRunner_DrawsStatusAndLogsThenStops(t *testing.T) {
	var out syncBuffer
	r := Start(&out)
	r.Set("[turn 5.6k]", "[session sent 1k]", "⠋")
	r.Log("openrouter: refused, continuing")
	// A log line printed right before Stop can be dropped: Println runs as a
	// command, so the quit may be handled first. Wait for the line to print.
	require.Eventually(t, func() bool {
		return strings.Contains(out.String(), "openrouter: refused, continuing")
	}, 2*time.Second, 10*time.Millisecond)
	r.Stop()

	got := out.String()
	assert.Contains(t, got, "[turn 5.6k]")
	assert.Contains(t, got, "[session sent 1k]")
	assert.Contains(t, got, "openrouter: refused, continuing")
}

func TestRunner_StopWithoutUpdatesReturns(t *testing.T) {
	var out syncBuffer
	r := Start(&out)
	r.Stop()
	assert.False(t, strings.Contains(out.String(), "panic"))
}
