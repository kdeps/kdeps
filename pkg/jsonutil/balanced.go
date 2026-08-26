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

// Package jsonutil provides small JSON-text helpers for parsing LLM output
// that is expected to contain JSON but isn't guaranteed to be well-formed or
// single-line.
package jsonutil

// ScanBalancedObject returns the index just past the closing brace that
// matches the opening brace at text[start], tracking depth across nested
// objects/arrays and skipping over brace-like characters inside quoted
// strings (respecting \" escapes) so a literal "{" or "}" in a string value
// never miscounts depth. ok is false if text[start] isn't '{' or the object
// is never closed.
//
// This exists because a fixed-depth regex assumption ("arguments" is one
// level deep, or a value never spans a newline) repeatedly broke on real LLM
// output that didn't match the assumed shape -- nested JSON objects, or
// pretty-printed multi-line JSON -- silently truncating or corrupting the
// parsed result instead of failing loudly.
func ScanBalancedObject(text string, start int) (int, bool) {
	if start >= len(text) || text[start] != '{' {
		return 0, false
	}
	var st jsonScanState
	for i := start; i < len(text); i++ {
		if st.step(text[i]) {
			return i + 1, true
		}
	}
	return 0, false
}

// jsonScanState tracks brace depth and quoted-string context while scanning
// one character at a time, kept separate from ScanBalancedObject's loop so
// each function stays simple on its own (a single function combining both
// pushed this package's average cyclomatic complexity over the repo's
// cyclop threshold).
type jsonScanState struct {
	depth    int
	inString bool
	escaped  bool
}

// step advances the scan by one character. It returns true when this
// character closed the outermost object (depth returned to zero).
func (s *jsonScanState) step(c byte) bool {
	switch {
	case s.escaped:
		s.escaped = false
	case s.inString:
		switch c {
		case '\\':
			s.escaped = true
		case '"':
			s.inString = false
		}
	case c == '"':
		s.inString = true
	case c == '{':
		s.depth++
	case c == '}':
		s.depth--
		return s.depth == 0
	}
	return false
}
