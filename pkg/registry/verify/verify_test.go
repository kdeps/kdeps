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

package verify_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/registry/verify"
)

func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0600))
}

func TestVerifyDir_Clean(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "workflow.yaml", `
run:
  chat:
    model: ""
    apiKey: ""
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.False(t, result.HasErrors())
	assert.Empty(t, result.Findings)
}

func TestVerifyDir_HardcodedLLMApiKey(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "resource.yaml", `
run:
  chat:
    apiKey: sk-supersecret1234567890
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Findings, 1)
	assert.Equal(t, verify.SeverityError, result.Findings[0].Severity)
	assert.Contains(t, result.Findings[0].YAMLPath, "apiKey")
}

func TestVerifyDir_EnvExpressionAllowed(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "resource.yaml", `
run:
  chat:
    apiKey: env("OPENAI_API_KEY")
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.False(t, result.HasErrors())
	assert.Empty(t, result.Findings)
}

func TestVerifyDir_HardcodedModelWarning(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "resource.yaml", `
run:
  chat:
    model: gpt-4o
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.False(t, result.HasErrors())
	require.Len(t, result.Findings, 1)
	assert.Equal(t, verify.SeverityWarn, result.Findings[0].Severity)
}

func TestVerifyDir_BotToken(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "workflow.yaml", `
bot:
  discord:
    botToken: "my-real-discord-bot-token"
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Equal(t, verify.SeverityError, result.Findings[0].Severity)
	assert.Contains(t, result.Findings[0].YAMLPath, "botToken")
}

func TestVerifyDir_SlackTokens(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "workflow.yaml", `
bot:
  slack:
    botToken: "xoxb-realtoken"
    appToken: "xapp-realtoken"
    signingSecret: "abc123secret"
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Findings, 3)
}

func TestVerifyDir_WhatsAppCredentials(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "workflow.yaml", `
bot:
  whatsapp:
    accessToken: "EAABsbCS4IHABCDEF"
    webhookSecret: "wh-secret-xyz"
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Findings, 2)
}

func TestVerifyDir_HTTPAuthToken(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "resource.yaml", `
run:
  http:
    auth:
      type: bearer
      token: "hardcoded-bearer-token"
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
}

func TestVerifyDir_HTTPAuthPassword(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "resource.yaml", `
run:
  http:
    auth:
      type: basic
      password: "s3cr3t"
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
}

func TestVerifyDir_SearchWebApiKey(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "resource.yaml", `
run:
  searchweb:
    provider: brave
    apiKey: "BSA_real_key_12345"
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
}

func TestVerifyDir_TranscriberApiKey(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "resource.yaml", `
run:
  transcriber:
    online:
      provider: deepgram
      apiKey: "dg_real_api_key"
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
}

func TestVerifyDir_PlaceholderAllowed(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "resource.yaml", `
run:
  chat:
    apiKey: "<YOUR_API_KEY>"
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.False(t, result.HasErrors())
}

func TestVerifyDir_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
}

func TestVerifyDir_SkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()
	hiddenDir := filepath.Join(dir, ".git")
	require.NoError(t, os.Mkdir(hiddenDir, 0750))
	writeYAML(t, hiddenDir, "config.yaml", "token: realtoken123\n")
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
}

func TestResult_ErrorNilWhenNoFindings(t *testing.T) {
	r := verify.Result{}
	assert.Nil(t, r.Error())
}

func TestResult_ErrorMessageContainsAll(t *testing.T) {
	r := verify.Result{Findings: []verify.Finding{
		{File: "f.yaml", YAMLPath: "run.chat.apiKey", Severity: verify.SeverityError, Message: "hardcoded key"},
	}}
	err := r.Error()
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "pre-publish verification failed")
	assert.Contains(t, err.Error(), "f.yaml")
}

func TestVerifyDir_UnreadableDir(t *testing.T) {
	_, err := verify.Dir("/nonexistent/path/xyz123")
	assert.Error(t, err)
}

func TestVerifyDir_NonYAMLFilesIgnored(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("apiKey: real-key"), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "script.sh"), []byte("TOKEN=realtoken"), 0600))
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
}

func TestVerifyDir_SequenceNodeWithCredentials(t *testing.T) {
	dir := t.TempDir()
	// A sequence containing mapping nodes with credential fields.
	writeYAML(t, dir, "resource.yaml", `
items:
  - apiKey: "secret-in-sequence"
    model: gpt-4o
`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
}

func TestVerifyDir_MultipleResourceFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "resources"), 0750))
	writeYAML(t, dir, "workflow.yaml", `name: test`)
	writeYAML(t, filepath.Join(dir, "resources"), "r1.yaml", `run:
  chat:
    apiKey: sk-secret1`)
	writeYAML(t, filepath.Join(dir, "resources"), "r2.yaml", `run:
  http:
    auth:
      token: real-token`)
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.True(t, result.HasErrors())
	assert.Len(t, result.Findings, 2)
}

func TestVerifyDir_HasErrorsFalseOnWarnOnly(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, dir, "resource.yaml", "model: gpt-4o\n")
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.False(t, result.HasErrors())
	assert.Len(t, result.Findings, 1)
	assert.Equal(t, verify.SeverityWarn, result.Findings[0].Severity)
}

func TestVerifyDir_EmptyYAMLFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "empty.yaml"), []byte(""), 0600))
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
}

func TestVerifyDir_InvalidYAMLSkipped(t *testing.T) {
	dir := t.TempDir()
	// Jinja2 templates and other non-parseable YAML should not cause errors.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "template.yaml"), []byte("{% if x %}foo{% endif %}"), 0600))
	result, err := verify.Dir(dir)
	require.NoError(t, err)
	assert.Empty(t, result.Findings)
}
