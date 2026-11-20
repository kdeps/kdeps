package agent

import (
	"io"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/tools"
)

func TestRunStreamToolCall_NotFoundListsOfferedTools(t *testing.T) {
	loop := freeModelLoop(t, "m", &modelListStreamer{text: "x"})
	loop.registry.Register(&tools.Tool{Name: "read_file"})
	loop.registry.Register(&tools.Tool{Name: "bash_exec"})

	tc := &domain.StreamedToolCall{Name: "search_local"}
	out := loop.runStreamToolCall(tc, io.Discard)

	if !strings.Contains(out, `tool \"search_local\" not found`) {
		t.Fatalf("missing not-found text: %s", out)
	}
	if !strings.Contains(out, "bash_exec") || !strings.Contains(out, "read_file") {
		t.Fatalf("offered tools not listed: %s", out)
	}
}

func TestRunStreamToolCall_BlankArgumentsRunTheTool(t *testing.T) {
	loop := freeModelLoop(t, "m", &modelListStreamer{text: "x"})
	loop.registry.Register(&tools.Tool{
		Name:       "list_files",
		Parameters: map[string]domain.ToolParam{},
		Execute:    func(map[string]any) (string, error) { return "a.txt", nil },
	})

	out := loop.runStreamToolCall(&domain.StreamedToolCall{Name: "list_files", Arguments: ""}, io.Discard)

	if !strings.Contains(out, "a.txt") {
		t.Fatalf("blank arguments did not run the tool: %s", out)
	}
}
