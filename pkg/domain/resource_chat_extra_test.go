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

package domain

import "testing"

func TestIsSystemSentinel(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"system", true},
		{"System", true},
		{"  SYSTEM  ", true},
		{"", false},
		{"anthropic", false},
		{"systemx", false},
	}
	for _, c := range cases {
		if got := IsSystemSentinel(c.in); got != c.want {
			t.Errorf("IsSystemSentinel(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
