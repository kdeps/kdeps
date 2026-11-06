package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCapToolResult_KeepsTheStartAndTheEnd(t *testing.T) {
	limit := maxToolResultBytes()
	var b strings.Builder
	b.WriteString("FIRST LINE\n")
	for b.Len() < limit*3 {
		b.WriteString("middle line of output\n")
	}
	b.WriteString("ok  	pkg/agent	3.1s\n")
	got := capToolResult(b.String())
	assert.True(t, strings.HasPrefix(got, "FIRST LINE\n"))
	assert.True(t, strings.HasSuffix(got, "ok  	pkg/agent	3.1s\n"), "the summary at the end survives")
	assert.Contains(t, got, "[tool result truncated:")
	assert.LessOrEqual(t, len(got), limit+120)
}

func TestShortenOldResults_CutsAllButTheRecentSteps(t *testing.T) {
	isolateEventsAndPresetsHome(t)
	limit := oldToolResultBytes()
	big := strings.Repeat("row\n", limit)
	h := []map[string]any{{"role": RoleUser, "content": "q"}}
	for _, id := range []string{"a", "b", "c", "d"} {
		h = append(h, rtAssistant(id, 0), map[string]any{"role": roleTool, "tool_call_id": id, "content": big})
	}
	shortenOldResults(h)
	for i, id := range []string{"a", "b", "c", "d"} {
		content, _ := h[2+2*i]["content"].(string)
		if id == "a" || id == "b" {
			assert.Contains(t, content, oldResultMarker, id)
			assert.Less(t, len(content), limit+120, id)
		} else {
			assert.Equal(t, big, content, "%s is one of the two most recent steps", id)
		}
	}
	first, _ := h[2]["content"].(string)
	shortenOldResults(h)
	again, _ := h[2]["content"].(string)
	assert.Equal(t, first, again, "a shortened result is not cut twice")
}

func TestShortenOldResults_LeavesFewStepsAlone(t *testing.T) {
	isolateEventsAndPresetsHome(t)
	big := strings.Repeat("x", oldToolResultBytes()*4)
	h := []map[string]any{
		{"role": RoleUser, "content": "q"},
		rtAssistant("a", 0), map[string]any{"role": roleTool, "content": big},
		rtAssistant("b", 0), map[string]any{"role": roleTool, "content": big},
	}
	shortenOldResults(h)
	assert.Equal(t, big, h[2]["content"])
}

func TestFormatForPromptCapped_ShortensAnOvershootingChain(t *testing.T) {
	isolateEventsAndPresetsHome(t)
	m := NewMemoryStore(t.TempDir())
	m.SetCwd("/Users/test/Projects/foo")
	long := strings.Repeat("detail ", 200)
	for _, k := range []string{"task:alpha", "task:beta", "task:gamma"} {
		require.NoError(t, m.Set(k, k+" "+long))
	}
	got := m.FormatForPromptCapped(50, "alpha beta gamma task", 0, 0, 0)
	require.NotEmpty(t, got)
	assert.Contains(t, got, "task:alpha")
	assert.NotContains(
		t,
		got,
		strings.Repeat("detail ", 30),
		"kept values are cut to a summary when the budget is overshot",
	)
}
