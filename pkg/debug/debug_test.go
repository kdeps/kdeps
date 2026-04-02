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
		instrument  string
		kdepsDebug  string
		debug       string
		wantEnabled bool
	}{
		{name: "KDEPS_INSTRUMENT=true enables instrumentation", instrument: "true", wantEnabled: true},
		{name: "KDEPS_DEBUG=true enables debug (legacy)", kdepsDebug: "true", wantEnabled: true},
		{name: "DEBUG=true enables debug (legacy)", debug: "true", wantEnabled: true},
		{name: "KDEPS_DEBUG=1 does not enable debug", kdepsDebug: "1", wantEnabled: false},
		{name: "empty env vars disable debug", wantEnabled: false},
		{name: "KDEPS_INSTRUMENT=false disables", instrument: "false", wantEnabled: false},
		{name: "KDEPS_DEBUG=false disables debug", kdepsDebug: "false", wantEnabled: false},
		{name: "DEBUG=false disables debug", debug: "false", wantEnabled: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KDEPS_INSTRUMENT", tt.instrument)
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

func TestRLE(t *testing.T) {
	tests := []struct {
		input []string
		want  string
	}{
		{[]string{"a", "b", "c"}, "a -> b -> c"},
		{[]string{"a", "a", "a"}, "a(3x)"},
		{[]string{"a", "b", "b", "c"}, "a -> b(2x) -> c"},
		{[]string{"a"}, "a"},
		{nil, ""},
	}
	for _, tt := range tests {
		got := rle(tt.input)
		if got != tt.want {
			t.Errorf("rle(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
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
	t.Setenv("KDEPS_DEBUG", "true")
	t.Setenv("NO_COLOR", "1")
	got := captureStderr(t, func() {
		Reset()
		Log("enter: alpha")
		Log("enter: beta")
	})
	if got != "" {
		t.Errorf("partial group should produce no output, got %q", got)
	}
	Reset()
}

func TestLog_CompleteGroup(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	t.Setenv("NO_COLOR", "1")
	got := captureStderr(t, func() {
		Reset()
		for _, name := range []string{"a", "b", "c", "d", "e"} {
			Log("enter: " + name)
		}
	})
	if want := "(a) b -> c -> d -> e\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	Reset()
}

func TestLog_AllSameCollapsed(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	t.Setenv("NO_COLOR", "1")
	got := captureStderr(t, func() {
		Reset()
		for range 5 {
			Log("enter: UnmarshalYAML")
		}
	})
	if want := "UnmarshalYAML(5x)\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	Reset()
}

func TestLog_PartialDuplicatesInChain(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	t.Setenv("NO_COLOR", "1")
	got := captureStderr(t, func() {
		Reset()
		for _, name := range []string{"ValidateResource", "validateRemote", "UnmarshalYAML", "UnmarshalYAML", "UnmarshalYAML"} {
			Log("enter: " + name)
		}
	})
	if want := "(ValidateResource) validateRemote -> UnmarshalYAML(3x)\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	Reset()
}

func TestFlush_WritesPartialGroup(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	t.Setenv("NO_COLOR", "1")
	got := captureStderr(t, func() {
		Reset()
		Log("enter: one")
		Log("enter: two")
		Flush()
	})
	if want := "(one) two\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFlush_SingleItem(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")
	t.Setenv("NO_COLOR", "1")
	got := captureStderr(t, func() {
		Reset()
		Log("enter: solo")
		Flush()
	})
	if want := "(solo)\n"; got != want {
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

func TestColorize_Enabled(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm")
	got := colorize("hello")
	if got != ansiCyan+"hello"+ansiReset {
		t.Errorf("colorize with color enabled = %q", got)
	}
}

func TestColorize_Disabled_NoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	got := colorize("hello")
	if got != "hello" {
		t.Errorf("colorize with NO_COLOR = %q, want plain", got)
	}
}

func TestColorize_Disabled_DumbTerm(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "dumb")
	got := colorize("hello")
	if got != "hello" {
		t.Errorf("colorize with TERM=dumb = %q, want plain", got)
	}
}
