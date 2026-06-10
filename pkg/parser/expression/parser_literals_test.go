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

package expression

import (
	"testing"
)

func TestLooksLikeAuthToken_FinalReturnFalse(t *testing.T) {
	p := &Parser{}

	// "user@example.com" has >= 8 chars, contains "@" which is not in
	// containsInvalidChars check (only parens/quotes/brackets/braces/spaces),
	// so it passes through but fails ^[a-zA-Z0-9\-_\.]+$ regex
	if p.looksLikeAuthToken("user@example.com") {
		t.Error("expected false for email-like string")
	}

	// "value$with$special" similarly should return false
	if p.looksLikeAuthToken("value$with$special") {
		t.Error("expected false for string with $")
	}
}
