package llm

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/instrument"
)

// A chat resource's call is recorded in an instrumented session (kdeps run
// --instrument recorded no model calls at all).
func TestRecordWorkflowRequestAndResponse(t *testing.T) {
	// Without a session nothing is recorded and nothing panics.
	recordWorkflowRequest(&domain.ChatConfig{}, "b", "m", nil, nil)
	recordWorkflowResponse("b", "m", nil, nil, 0)

	t.Setenv(instrument.RootEnv, t.TempDir())
	s, err := instrument.Start("wf", nil, time.Now())
	require.NoError(t, err)
	recordWorkflowRequest(&domain.ChatConfig{}, "openai", "gpt-x",
		[]map[string]interface{}{{"role": "user", "content": "hi"}}, []domain.Tool{{Name: "calc"}})
	recordWorkflowResponse("openai", "gpt-x",
		map[string]any{"message": map[string]any{"content": "hello", "tool_calls": []any{"c"}}}, nil, time.Millisecond)
	recordWorkflowResponse("openai", "gpt-x", map[string]any{"error": "boom"}, errors.New("failed"), 0)
	require.NoError(t, s.Close(nil))

	f, err := os.Open(filepath.Join(s.Dir, instrument.EventsFile))
	require.NoError(t, err)
	defer f.Close()
	var got []map[string]any
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var ev map[string]any
		require.NoError(t, json.Unmarshal(sc.Bytes(), &ev))
		if ev["source"] == "workflow" {
			got = append(got, ev)
		}
	}
	require.Len(t, got, 3)
	assert.Equal(t, "model_request", got[0]["kind"])
	assert.Equal(t, []any{"calc"}, got[0]["tools"])
	assert.Equal(t, "hello", got[1]["content"])
	assert.Equal(t, "failed", got[2]["error"])
}
