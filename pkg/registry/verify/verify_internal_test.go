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

// Internal tests for unexported helpers in verify.go.
package verify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- looksLikeSecret ---

func TestLooksLikeSecret_EmptyString(t *testing.T) {
	assert.False(t, looksLikeSecret(""))
}

func TestLooksLikeSecret_EnvExpression(t *testing.T) {
	assert.False(t, looksLikeSecret(`env("OPENAI_API_KEY")`))
	assert.False(t, looksLikeSecret(`ENV("key")`))
}

func TestLooksLikeSecret_DollarBrace(t *testing.T) {
	assert.False(t, looksLikeSecret("${MY_VAR}"))
}

func TestLooksLikeSecret_DoubleBrace(t *testing.T) {
	assert.False(t, looksLikeSecret("{{ some_expression }}"))
}

func TestLooksLikeSecret_AngleBracketPlaceholder(t *testing.T) {
	assert.False(t, looksLikeSecret("<YOUR_KEY_HERE>"))
	assert.False(t, looksLikeSecret("<replace-me>"))
}

func TestLooksLikeSecret_YourPrefix(t *testing.T) {
	assert.False(t, looksLikeSecret("your_api_key"))
	assert.False(t, looksLikeSecret("your-token-here"))
}

func TestLooksLikeSecret_XxxPlaceholder(t *testing.T) {
	assert.False(t, looksLikeSecret("xxx"))
	assert.False(t, looksLikeSecret("XXXX"))
}

func TestLooksLikeSecret_ChangeMe(t *testing.T) {
	assert.False(t, looksLikeSecret("change-me"))
	assert.False(t, looksLikeSecret("change_me"))
}

func TestLooksLikeSecret_Placeholder(t *testing.T) {
	assert.False(t, looksLikeSecret("placeholder"))
	assert.False(t, looksLikeSecret("PLACEHOLDER"))
}

func TestLooksLikeSecret_Todo(t *testing.T) {
	assert.False(t, looksLikeSecret("TODO"))
	assert.False(t, looksLikeSecret("todo"))
}

func TestLooksLikeSecret_Ellipsis(t *testing.T) {
	assert.False(t, looksLikeSecret("..."))
}

func TestLooksLikeSecret_RealKey(t *testing.T) {
	assert.True(t, looksLikeSecret("sk-supersecret1234567890"))
	assert.True(t, looksLikeSecret("xoxb-slack-bot-token"))
	assert.True(t, looksLikeSecret("dg_real_deepgram_key"))
}

// --- joinPath ---

func TestJoinPath_Empty(t *testing.T) {
	assert.Equal(t, "key", joinPath("", "key"))
}

func TestJoinPath_WithParent(t *testing.T) {
	assert.Equal(t, "run.chat.apiKey", joinPath("run.chat", "apiKey"))
}

// --- Finding.String ---

func TestFinding_StringError(t *testing.T) {
	f := Finding{File: "f.yaml", YAMLPath: "run.chat.apiKey", Severity: SeverityError, Message: "bad key"}
	s := f.String()
	assert.Contains(t, s, "ERROR")
	assert.Contains(t, s, "f.yaml")
	assert.Contains(t, s, "bad key")
}

func TestFinding_StringWarn(t *testing.T) {
	f := Finding{File: "f.yaml", YAMLPath: "model", Severity: SeverityWarn, Message: "model hint"}
	s := f.String()
	assert.Contains(t, s, "WARN")
}

// --- credentialFields and modelFields ---

func TestCredentialFields_CaseSensitivity(t *testing.T) {
	// The map uses lowercase keys; verify all expected fields are present.
	expected := []string{"apikey", "password", "token", "bottoken", "apptoken",
		"signingsecret", "webhooksecret", "accesstoken"}
	for _, k := range expected {
		_, ok := credentialFields[k]
		assert.True(t, ok, "missing credential field: %s", k)
	}
}

func TestModelFields_HasModel(t *testing.T) {
	assert.True(t, modelFields["model"])
}
