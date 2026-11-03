package tools

import "context"

// ModelArgs is args without the keys the agent loop adds for its own use: _ctx
// (the turn's context), _bg_ch (a <-chan struct{}) and _interactive (terminal
// only). They are not the model's arguments and cannot be encoded as JSON, so
// a tool that sends its arguments on (to the kdeps API, to a code-graph tool)
// sends ModelArgs(args); any other context value is left out too.
func ModelArgs(args map[string]any) map[string]any {
	out := make(map[string]any, len(args))
	for k, v := range args {
		if k == "_ctx" || k == "_bg_ch" || k == "_interactive" {
			continue
		}
		if _, isCtx := v.(context.Context); isCtx {
			continue
		}
		out[k] = v
	}
	return out
}
