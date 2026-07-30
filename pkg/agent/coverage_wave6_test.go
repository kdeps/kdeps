// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package agent

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

func TestRequestPlan_NoLocalServer(t *testing.T) {
	for _, backend := range []string{"", "file", "gguf"} {
		l := &Loop{config: Config{Backend: backend}}
		if got := requestPlan(l, "do something", false); got != nil {
			t.Fatalf("backend %q: %v", backend, got)
		}
	}
}

func TestRegisterGoalToolsAndSettleNoEnforcer(t *testing.T) {
	(*Loop)(nil).registerGoalTools()
	(&Loop{}).registerGoalTools()

	reg := kdepstools.NewRegistry()
	l := &Loop{registry: reg, config: Config{}}
	l.registerGoalTools()
	l.registerGoalTools() // second call no-op
	if reg.Get(toolNameTaskComplete) == nil {
		t.Fatal("task_complete not registered")
	}
	if reg.Get(toolNameTaskFail) == nil {
		t.Fatal("task_fail not registered")
	}
	msg, err := l.settleTask(map[string]any{"id": 1}, GoalTaskDone)
	if err == nil {
		t.Fatalf("expected error, got %q", msg)
	}
}

func TestSessionStore_Close(t *testing.T) {
	dir := t.TempDir()
	s := NewSessionStore(dir)
	s.SetCwd(filepath.Join(dir, "proj"))
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s2 := NewSessionStore(dir)
	s2.SetCwd(filepath.Join(dir, "proj"))
	_, _ = s2.List()
	if err := s2.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s2.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCmdSearch_UsageAndIndex(t *testing.T) {
	r := &REPL{
		loop: &Loop{config: Config{}, session: NewSession(0)},
		ctx:  context.Background(),
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	rd, wr, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatal(pipeErr)
	}
	os.Stdout = wr
	os.Stderr = wr

	if err := r.cmdSearch(nil); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldWD, _ := os.Getwd()
	if chErr := os.Chdir(dir); chErr != nil {
		t.Fatal(chErr)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })
	if err := r.cmdSearch([]string{"index"}); err != nil {
		t.Fatal(err)
	}
	_ = r.cmdSearch([]string{"main"})

	_ = wr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	_, _ = io.ReadAll(rd)
}
