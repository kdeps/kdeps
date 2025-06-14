package resolver

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
)

func TestEncodePythonEnv(t *testing.T) {
	dr := &DependencyResolver{Logger: logging.GetLogger()}

	env := map[string]string{"A": "alpha", "B": "beta"}
	encoded := dr.encodePythonEnv(&env)
	if encoded == nil || len(*encoded) != 2 {
		t.Fatalf("expected 2 encoded entries")
	}
	if (*encoded)["A"] == "alpha" {
		t.Errorf("value A not encoded")
	}
}

func TestEncodePythonOutputs(t *testing.T) {
	dr := &DependencyResolver{}
	stderr := "some err"
	stdout := "some out"
	e1, e2 := dr.encodePythonOutputs(&stderr, &stdout)
	if *e1 == stderr || *e2 == stdout {
		t.Errorf("outputs not encoded: %s %s", *e1, *e2)
	}

	// nil pass-through
	n1, n2 := dr.encodePythonOutputs(nil, nil)
	if n1 != nil || n2 != nil {
		t.Errorf("expected nil return for nil inputs")
	}
}

func TestEncodePythonStderrStdoutFormatting(t *testing.T) {
	dr := &DependencyResolver{}
	msg := "line1\nline2"
	got := dr.encodePythonStderr(&msg)
	if len(got) == 0 || got[0] != ' ' {
		t.Errorf("unexpected format: %s", got)
	}
	got2 := dr.encodePythonStdout(nil)
	if got2 != "    stdout = \"\"\n" {
		t.Errorf("unexpected default stdout: %s", got2)
	}
}
