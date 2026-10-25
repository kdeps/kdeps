package llm

import (
	"time"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/instrument"
)

// recordWorkflowRequest records a chat resource's request in an instrumented
// session (workflow mode, and the agent loop's own engine calls such as prompt
// refinement): the agent loop's turns are recorded by the loop itself.
func recordWorkflowRequest(cfg *domain.ChatConfig, backend, model string, messages []map[string]interface{},
	tools []domain.Tool,
) {
	if instrument.Active() == nil || cfg == nil {
		return
	}
	names := make([]string, 0, len(tools))
	for _, t := range tools {
		names = append(names, t.Name)
	}
	instrument.Event("model_request", map[string]any{
		"source": "workflow", "backend": backend, "model": model, "messages": messages, "tools": names,
	})
}

// recordWorkflowResponse records what the model answered a chat resource.
func recordWorkflowResponse(backend, model string, response map[string]any, err error, took time.Duration) {
	if instrument.Active() == nil {
		return
	}
	fields := map[string]any{
		"source": "workflow", "backend": backend, "model": model, "duration_ms": took.Milliseconds(),
	}
	if msg, ok := response["message"].(map[string]any); ok {
		fields["content"] = msg["content"]
		if calls, has := msg["tool_calls"]; has {
			fields["tool_calls"] = calls
		}
	}
	if errMsg, ok := response["error"]; ok {
		fields["error"] = errMsg
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	instrument.Event("model_response", fields)
}
