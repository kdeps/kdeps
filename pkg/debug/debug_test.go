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

package debug

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestEnabled(t *testing.T) {
	tests := []struct {
		name        string
		kdepsDebug  string
		debug       string
		wantEnabled bool
	}{
		{name: "KDEPS_DEBUG=true enables debug", kdepsDebug: "true", wantEnabled: true},
		{name: "DEBUG=true enables debug", debug: "true", wantEnabled: true},
		{name: "KDEPS_DEBUG=1 does not enable debug", kdepsDebug: "1", wantEnabled: false},
		{name: "empty env vars disable debug", wantEnabled: false},
		{name: "KDEPS_DEBUG=false disables debug", kdepsDebug: "false", wantEnabled: false},
		{name: "DEBUG=false disables debug", debug: "false", wantEnabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KDEPS_DEBUG", tt.kdepsDebug)
			t.Setenv("DEBUG", tt.debug)
			got := Enabled()
			if got != tt.wantEnabled {
				t.Errorf("Enabled() = %v, want %v", got, tt.wantEnabled)
			}
		})
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w //nolint:reassign
	fn()
	w.Close()
	os.Stderr = oldStderr //nolint:reassign
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestLog_Disabled(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "")
	got := captureStderr(t, func() {
		Reset()
		Log("enter: anything")
	})
	if got != "" {
		t.Errorf("expected no output when disabled, got %q", got)
	}
	Reset()
}

func TestLog_FirstItem(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		Log("enter: alpha")
	})
	// First item prints the label in parens, no arrow yet.
	want := "(alpha)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	Reset()
}

func TestLog_ChainWithinGroup(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		Log("enter: alpha")
		Log("enter: beta")
		Log("enter: gamma")
	})
	// Each subsequent call in the group overwrites the line.
	want := "(alpha)" +
		"\r\033[2K(alpha) beta" +
		"\r\033[2K(alpha) beta -> gamma"
	if got != want {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
	Reset()
}

func TestLog_GroupBoundary(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		// Fill one complete group (5 items).
		for _, name := range []string{"a", "b", "c", "d", "e"} {
			Log("enter: " + name)
		}
		// Sixth item starts a new group.
		Log("enter: f")
	})

	// Group 1 builds up: (a), (a) b, (a) b->c, (a) b->c->d, (a) b->c->d->e
	// Then \n to end group 1, then (f) on the new line.
	want := "(a)" +
		"\r\033[2K(a) b" +
		"\r\033[2K(a) b -> c" +
		"\r\033[2K(a) b -> c -> d" +
		"\r\033[2K(a) b -> c -> d -> e" +
		"\n(f)"
	if got != want {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
	Reset()
}

func TestFlush(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		Log("enter: one")
		Log("enter: two")
		Flush()
	})
	want := "(one)" +
		"\r\033[2K(one) two" +
		"\n"
	if got != want {
		t.Errorf("got\n%q\nwant\n%q", got, want)
	}
}

func TestFlush_Empty(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		Flush()
	})
	if got != "" {
		t.Errorf("Flush() on empty chain wrote %q, want empty", got)
	}
}
