package resolver

import (
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	pklExec "github.com/kdeps/schema/gen/exec"
	pklHTTP "github.com/kdeps/schema/gen/http"
	pklLLM "github.com/kdeps/schema/gen/llm"
	pklPython "github.com/kdeps/schema/gen/python"
)

func TestGetResourceTimestamp_SuccessPaths(t *testing.T) {
	ts := &pkl.Duration{Value: 123, Unit: pkl.Second}
	resID := "res"

	// Exec
	execImpl := &pklExec.ExecImpl{Resources: &map[string]*pklExec.ResourceExec{resID: {Timestamp: ts}}}
	if got, _ := getResourceTimestamp(resID, execImpl); got != ts {
		t.Errorf("exec timestamp mismatch")
	}

	// Python
	pyImpl := &pklPython.PythonImpl{Resources: &map[string]*pklPython.ResourcePython{resID: {Timestamp: ts}}}
	if got, _ := getResourceTimestamp(resID, pyImpl); got != ts {
		t.Errorf("python timestamp mismatch")
	}

	// LLM
	llmImpl := &pklLLM.LLMImpl{Resources: &map[string]*pklLLM.ResourceChat{resID: {Timestamp: ts}}}
	if got, _ := getResourceTimestamp(resID, llmImpl); got != ts {
		t.Errorf("llm timestamp mismatch")
	}

	// HTTP
	httpImpl := &pklHTTP.HTTPImpl{Resources: &map[string]*pklHTTP.ResourceHTTPClient{resID: {Timestamp: ts}}}
	if got, _ := getResourceTimestamp(resID, httpImpl); got != ts {
		t.Errorf("http timestamp mismatch")
	}
}

func TestGetResourceTimestamp_Errors(t *testing.T) {
	ts := &pkl.Duration{Value: 1, Unit: pkl.Second}
	execImpl := &pklExec.ExecImpl{Resources: &map[string]*pklExec.ResourceExec{"id": {Timestamp: ts}}}

	if _, err := getResourceTimestamp("missing", execImpl); err == nil {
		t.Errorf("expected error for missing resource id")
	}

	// nil timestamp
	execImpl2 := &pklExec.ExecImpl{Resources: &map[string]*pklExec.ResourceExec{"id": {Timestamp: nil}}}
	if _, err := getResourceTimestamp("id", execImpl2); err == nil {
		t.Errorf("expected error for nil timestamp")
	}

	// unknown type
	if _, err := getResourceTimestamp("id", 42); err == nil {
		t.Errorf("expected error for unknown type")
	}
}

func TestFormatDuration_Simple(t *testing.T) {
	cases := []struct {
		d        time.Duration
		expected string
	}{
		{3 * time.Second, "3s"},
		{2*time.Minute + 5*time.Second, "2m 5s"},
		{1*time.Hour + 10*time.Minute + 30*time.Second, "1h 10m 30s"},
		{0, "0s"},
	}
	for _, c := range cases {
		got := formatDuration(c.d)
		if got != c.expected {
			t.Errorf("formatDuration(%v) = %q, want %q", c.d, got, c.expected)
		}
	}
}

func TestFormatDurationExtra(t *testing.T) {
	cases := []struct {
		dur  time.Duration
		want string
	}{
		{time.Second * 5, "5s"},
		{time.Minute*2 + time.Second*10, "2m 10s"},
		{time.Hour*1 + time.Minute*3 + time.Second*4, "1h 3m 4s"},
	}

	for _, c := range cases {
		got := formatDuration(c.dur)
		if got != c.want {
			t.Errorf("formatDuration(%v) = %s, want %s", c.dur, got, c.want)
		}
	}
}
