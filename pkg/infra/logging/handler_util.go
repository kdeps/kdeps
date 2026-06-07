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

package logging

import (
	"os"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

func (h *PrettyHandler) colorize(color, text string) string {
	kdeps_debug.Log("enter: colorize")
	if h.opts.DisableColors {
		return text
	}
	return color + text + colorReset
}

// isTerminal checks if the file is a terminal.
func isTerminal(f *os.File) bool {
	kdeps_debug.Log("enter: isTerminal")
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}
