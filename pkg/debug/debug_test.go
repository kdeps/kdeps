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

func TestLog_PartialGroupNoOutput(t *testing.T) {
	// Fewer than maxChainLen calls should produce no output until Flush.
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		Log("enter: alpha")
		Log("enter: beta")
		Log("enter: gamma")
	})
	if got != "" {
		t.Errorf("partial group should not write output, got %q", got)
	}
	Reset()
}

func TestLog_CompleteGroupWritesLine(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		for _, name := range []string{"a", "b", "c", "d", "e"} {
			Log("enter: " + name)
		}
	})
	want := "(a) b -> c -> d -> e\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	Reset()
}

func TestLog_MultipleGroups(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		names := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
		for _, name := range names {
			Log("enter: " + name)
		}
	})
	want := "(a) b -> c -> d -> e\n(f) g -> h -> i -> j\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	Reset()
}

func TestFlush_WritesPartialGroup(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		Log("enter: one")
		Log("enter: two")
		Flush()
	})
	want := "(one) two\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFlush_SingleItem(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		Log("enter: solo")
		Flush()
	})
	want := "(solo)\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
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

func TestFlush_AfterCompleteGroup(t *testing.T) {
	// Flush after a full group already written should produce no extra output.
	t.Setenv("KDEPS_DEBUG", "true")
	got := captureStderr(t, func() {
		Reset()
		for _, name := range []string{"a", "b", "c", "d", "e"} {
			Log("enter: " + name)
		}
		Flush() // chain is already aligned; no partial remainder
	})
	want := "(a) b -> c -> d -> e\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
