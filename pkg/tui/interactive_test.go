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

package tui

import "testing"

// The test runner's stdin is never an interactive terminal, so both wrappers
// must report false rather than panic on a closed/redirected fd.
func TestIsInteractive_FalseUnderTest(t *testing.T) {
	if isInteractive() {
		t.Fatal("isInteractive() = true under go test, want false")
	}
	if IsInteractive() {
		t.Fatal("IsInteractive() = true under go test, want false")
	}
}
