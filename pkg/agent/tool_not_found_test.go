package agent

import (
	"io"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/tools"
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
