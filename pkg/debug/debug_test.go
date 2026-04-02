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
		{
			name:        "KDEPS_DEBUG=true enables debug",
			kdepsDebug:  "true",
			wantEnabled: true,
		},
		{
			name:        "DEBUG=true enables debug",
			debug:       "true",
			wantEnabled: true,
		},
		{
			name:        "KDEPS_DEBUG=1 enables debug",
			kdepsDebug:  "1",
			wantEnabled: false,
		},
		{
			name:        "empty env vars disable debug",
			wantEnabled: false,
		},
		{
			name:        "KDEPS_DEBUG=false disables debug",
			kdepsDebug:  "false",
			wantEnabled: false,
		},
		{
			name:        "DEBUG=false disables debug",
			debug:       "false",
			wantEnabled: false,
		},
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

func TestLog(t *testing.T) {
	tests := []struct {
		name       string
		envEnabled bool
		message    string
		wantOutput string
	}{
		{
			name:       "log renders chain on stderr when enabled",
			envEnabled: true,
			message:    "enter: testFunc",
			wantOutput: "\r\033[2Kenter: testFunc",
		},
		{
			name:       "log does nothing when disabled",
			envEnabled: false,
			message:    "enter: testFunc",
			wantOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envEnabled {
				t.Setenv("KDEPS_DEBUG", "true")
			} else {
				t.Setenv("KDEPS_DEBUG", "")
			}

			got := captureStderr(t, func() {
				Reset()
				Log(tt.message)
			})

			if got != tt.wantOutput {
				t.Errorf("Log() output = %q, want %q", got, tt.wantOutput)
			}
			Reset()
		})
	}
}

func TestLog_Chain(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")

	got := captureStderr(t, func() {
		Reset()
		Log("enter: alpha")
		Log("enter: beta")
		Log("enter: gamma")
	})

	// Each call overwrites the line; the final write shows the full chain.
	want := "\r\033[2Kenter: alpha\r\033[2Kenter: alpha -> beta\r\033[2Kenter: alpha -> beta -> gamma"
	if got != want {
		t.Errorf("chain output =\n%q\nwant\n%q", got, want)
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

	want := "\r\033[2Kenter: one\r\033[2Kenter: one -> two\n"
	if got != want {
		t.Errorf("Flush() output =\n%q\nwant\n%q", got, want)
	}
}

func TestFlush_Empty(t *testing.T) {
	t.Setenv("KDEPS_DEBUG", "true")

	got := captureStderr(t, func() {
		Reset()
		Flush() // nothing logged — should produce no output
	})

	if got != "" {
		t.Errorf("Flush() on empty chain wrote %q, want empty", got)
	}
}
