package logging

import "testing"

func TestLoggerWithAndOutput(t *testing.T) {
	base := NewTestLogger()
	child := base.With("k", "v")
	child.Info("hello")

	if out := child.GetOutput(); out == "" {
		t.Fatalf("expected output captured")
	}
}
