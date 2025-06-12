package resolver

import (
	"testing"

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
