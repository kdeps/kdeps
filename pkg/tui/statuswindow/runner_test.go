package statuswindow

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunner_DrawsStatusAndLogsThenStops(t *testing.T) {
	var out bytes.Buffer
	r := Start(&out)
	r.Set("[turn 5.6k]", "[session sent 1k]", "⠋")
	r.Log("openrouter: refused, continuing")
	r.Stop()

	got := out.String()
	assert.Contains(t, got, "[turn 5.6k]")
	assert.Contains(t, got, "[session sent 1k]")
	assert.Contains(t, got, "openrouter: refused, continuing")
}

func TestRunner_StopWithoutUpdatesReturns(t *testing.T) {
	var out bytes.Buffer
	r := Start(&out)
	r.Stop()
	assert.False(t, strings.Contains(out.String(), "panic"))
}
