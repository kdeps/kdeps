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

package api

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestBashExec_Echo(t *testing.T) {
	t.Setenv("KDEPS_RTK", "off")
	t.Setenv("KDEPS_ALLOW_BASH", "true")
	out, err := BashExec(t.Context(), map[string]any{argCommand: "echo hi"})
	if err != nil || out != "hi" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestBashExec_UnescapesHTML(t *testing.T) {
	t.Setenv("KDEPS_RTK", "off")
	t.Setenv("KDEPS_ALLOW_BASH", "true")
	out, err := BashExec(t.Context(), map[string]any{argCommand: "echo a &amp;&amp; echo b"})
	if err != nil || out != "a\nb" {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestBashExec_NilContext(t *testing.T) {
	var noCtx context.Context
	_, err := BashExec(noCtx, map[string]any{argCommand: "echo hi"})
	if err == nil || err.Error() != msgBashNil {
		t.Fatal(err)
	}
}

func TestBashExec_EmptyCommand(t *testing.T) {
	t.Setenv("KDEPS_ALLOW_BASH", "true")
	_, err := BashExec(t.Context(), map[string]any{argCommand: ""})
	if err == nil || err.Error() != msgBashRequired {
		t.Fatal(err)
	}
}

func TestBashExec_Disabled(t *testing.T) {
	t.Setenv("KDEPS_ALLOW_BASH", "false")
	_, err := BashExec(t.Context(), map[string]any{argCommand: "echo hi"})
	if err == nil || err.Error() != msgBashDisabled {
		t.Fatal(err)
	}
}

func TestBashExec_Blocked(t *testing.T) {
	t.Setenv("KDEPS_ALLOW_BASH", "true")
	t.Setenv("KDEPS_ALLOW_ROOT", "false")
	_, err := BashExec(t.Context(), map[string]any{argCommand: "ls /"})
	if err == nil || !strings.HasPrefix(err.Error(), "bash_exec: blocked:") {
		t.Fatal(err)
	}
}

func TestBashExec_StderrOnFailure(t *testing.T) {
	t.Setenv("KDEPS_RTK", "off")
	t.Setenv("KDEPS_ALLOW_BASH", "true")
	_, err := BashExec(t.Context(), map[string]any{argCommand: "echo 'err msg' >&2; exit 1"})
	if err == nil || !strings.Contains(err.Error(), "err msg") {
		t.Fatal(err)
	}
}

func TestBashExec_StderrOnSuccess(t *testing.T) {
	t.Setenv("KDEPS_RTK", "off")
	t.Setenv("KDEPS_ALLOW_BASH", "true")
	out, err := BashExec(t.Context(), map[string]any{argCommand: "echo 'warning' >&2; echo 'output'"})
	if err != nil || !strings.Contains(out, "output") || !strings.Contains(out, "warning") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestBashExec_Cancel(t *testing.T) {
	t.Setenv("KDEPS_RTK", "off")
	t.Setenv("KDEPS_ALLOW_BASH", "true")
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	out, err := BashExec(ctx, map[string]any{argCommand: "echo partial; sleep 10"})
	if err != nil || !strings.Contains(out, "[interrupted]") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestBashExec_AlreadyCancelled(t *testing.T) {
	t.Setenv("KDEPS_RTK", "off")
	t.Setenv("KDEPS_ALLOW_BASH", "true")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	out, err := BashExec(ctx, map[string]any{argCommand: "sleep 5"})
	if err != nil || !strings.Contains(out, "[interrupted]") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestBashExec_OutputWriter(t *testing.T) {
	t.Setenv("KDEPS_RTK", "off")
	t.Setenv("KDEPS_ALLOW_BASH", "true")
	var buf bytes.Buffer
	ctx := WithBashOpts(t.Context(), BashOpts{Output: &buf})
	out, err := BashExec(ctx, map[string]any{argCommand: "echo hi"})
	if err != nil || out != "hi" || !strings.Contains(buf.String(), "hi") {
		t.Fatalf("out=%q buf=%q err=%v", out, buf.String(), err)
	}
}

func TestWithBashOpts_NilContext(t *testing.T) {
	var noCtx context.Context
	if WithBashOpts(noCtx, BashOpts{}) != nil {
		t.Fatal("nil ctx")
	}
	if bashOpts(noCtx) != (BashOpts{}) {
		t.Fatal("opts")
	}
}

func TestBashExecResult(t *testing.T) {
	out, err := bashExecResult("output text", "", nil)
	if err != nil || !strings.Contains(out, "output text") {
		t.Fatalf("out=%q err=%v", out, err)
	}
	_, err = bashExecResult("", "error output", errors.New("exit code 1"))
	if err == nil || !strings.Contains(err.Error(), "exit code 1") || !strings.Contains(err.Error(), "stderr:") {
		t.Fatal(err)
	}
	_, err = bashExecResult("", "", errors.New("exit code 1"))
	if err == nil || strings.Contains(err.Error(), "stderr:") {
		t.Fatal(err)
	}
	out, err = bashExecResult("stdout text", "stderr text", nil)
	if err != nil || !strings.Contains(out, "stdout text") || !strings.Contains(out, "stderr: stderr text") {
		t.Fatalf("out=%q err=%v", out, err)
	}
}

func TestBashExecCancelText(t *testing.T) {
	got := bashExecResultText("some output", "") + "\n[interrupted]"
	if !strings.Contains(got, "some output") || !strings.Contains(got, "[interrupted]") {
		t.Fatal(got)
	}
	got = bashExecResultText("", "error detail") + "\n[interrupted]"
	if !strings.Contains(got, "stderr:") || !strings.Contains(got, "error detail") {
		t.Fatal(got)
	}
	got = bashExecResultText("stdout line", "stderr line") + "\n[interrupted]"
	if !strings.Contains(got, "stdout line") || !strings.Contains(got, "stderr line") {
		t.Fatal(got)
	}
}

func TestTruncateBashOutput(t *testing.T) {
	if truncateBashOutput("hello world") != "hello world" {
		t.Fatal("plain")
	}
	if sanitizeBashOutput("hello\x00world\x01\x02\t\nfoo") != "helloworld\t\nfoo" {
		t.Fatal("controls")
	}
	if sanitizeBashOutput("\t\n\r") != "\t\n\r" {
		t.Fatal("kept")
	}
	if sanitizeBashOutput("hello\ufff9world\ufffafoo\ufffbbar") != "helloworldfoobar" {
		t.Fatal("interlinear")
	}
	cr := truncateBashOutput("line1\rline2\rline3")
	if !strings.Contains(cr, "\r") {
		t.Fatal(cr)
	}
	lines := make([]string, bashOutputMaxLines)
	for i := range lines {
		lines[i] = "y"
	}
	exact := strings.Join(lines, "\n")
	if truncateBashOutput(exact) != exact {
		t.Fatal("exact line limit")
	}
	over := exact + "\nmore\nmore2"
	got := truncateBashOutput(over)
	if !strings.Contains(got, "[Output truncated:") || !strings.Contains(got, "showing first 2000") {
		t.Fatal(got)
	}
	big := "header\n" + strings.Repeat("a", 60*1024) + "\nfooter"
	got = truncateBashOutput(big)
	if !strings.Contains(got, "bytes total") || len(got) > bashOutputMaxBytes+200 {
		t.Fatalf("len %d", len(got))
	}
}

func TestBashExec_ReadOnlyBlocksWrite(t *testing.T) {
	t.Setenv("KDEPS_ALLOW_BASH", "true")
	t.Setenv("KDEPS_BASH_MODE", "read-only")
	t.Setenv("KDEPS_ALLOW_ROOT", "false")
	_, err := BashExec(t.Context(), map[string]any{argCommand: "touch newfile"})
	if err == nil || !strings.HasPrefix(err.Error(), "bash_exec: blocked:") {
		t.Fatal(err)
	}
}
