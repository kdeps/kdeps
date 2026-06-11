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

func TestResource_ResponseBlock(t *testing.T) {
	t.Parallel()

	if (&Resource{}).HasResponseBlock() {
		t.Fatal("empty resource should not have response block")
	}
	cfg := (&Resource{APIServer: &APIResponseConfig{Success: true}}).ResponseBlock()
	if cfg == nil || cfg.Success != true {
		t.Fatal("apiServer block should be returned")
	}
	if name := (&Resource{APIServer: &APIResponseConfig{}}).ResponseBlockEventName(); name != "apiServer" {
		t.Fatalf("event name = %q, want apiServer", name)
	}
	if name := (&Resource{APIResponse: &APIResponseConfig{}}).ResponseBlockEventName(); name != "apiResponse" {
		t.Fatalf("event name = %q, want apiResponse", name)
	}
}

func TestActionConfig_InlineResponseBlock(t *testing.T) {
	t.Parallel()

	if (&ActionConfig{APIServer: &APIResponseConfig{}}).InlineResponseBlock() == nil {
		t.Fatal("apiServer inline should be recognized")
	}
	if !IsRecognizedResourceActionKey("apiServer") {
		t.Fatal("apiServer should be recognized action key")
	}
}

func TestResource_IsResponseOnlyPrimary(t *testing.T) {
	t.Parallel()

	res := &Resource{
		Before:      []ActionConfig{{Chat: &ChatConfig{}}},
		APIResponse: &APIResponseConfig{Success: true},
	}
	if !res.IsResponseOnlyPrimary() {
		t.Fatal("apiResponse-only resource should be response-only primary")
	}
	if !res.HasInlineActions() {
		t.Fatal("resource with before should have inline actions")
	}

	combo := &Resource{
		Chat:        &ChatConfig{},
		APIResponse: &APIResponseConfig{},
	}
	if combo.IsResponseOnlyPrimary() {
		t.Fatal("chat + apiResponse is not response-only primary")
	}
}
