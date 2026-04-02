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

func TestLog(t *testing.T) {
	tests := []struct {
		name       string
		envEnabled bool
		message    string
		wantOutput string
	}{
		{
			name:       "log writes to stderr when enabled",
			envEnabled: true,
			message:    "test message",
			wantOutput: "test message\n",
		},
		{
			name:       "log does nothing when disabled",
			envEnabled: false,
			message:    "test message",
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

			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w //nolint:reassign

			Log(tt.message)

			w.Close()
			os.Stderr = oldStderr //nolint:reassign

			var buf bytes.Buffer
			io.Copy(&buf, r)
			got := buf.String()

			if got != tt.wantOutput {
				t.Errorf("Log() output = %q, want %q", got, tt.wantOutput)
			}
		})
	}
}
