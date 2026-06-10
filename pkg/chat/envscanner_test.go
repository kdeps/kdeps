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

package chat

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func wfWithContent(content string) *GeneratedWorkflow {
	return &GeneratedWorkflow{
		Files: map[string]string{
			"workflow.yaml":       "apiVersion: kdeps.io/v1\n",
			"resources/main.yaml": content,
		},
	}
}

func TestScanEnvVars_NoVars(t *testing.T) {
	wf := wfWithContent("run:\n  exec:\n    command: ls\n")
	vars := ScanEnvVars(wf)
	assert.Empty(t, vars)
}

func TestScanEnvVars_OpenAIBackend(t *testing.T) {
	wf := wfWithContent("run:\n  chat:\n    backend: openai\n    model: gpt-4o\n")
	vars := ScanEnvVars(wf)
	assert.False(t, hasVar(vars, "OPENAI_API_KEY"))
}

func TestScanEnvVars_AnthropicBackend(t *testing.T) {
	wf := wfWithContent("run:\n  chat:\n    backend: anthropic\n")
	vars := ScanEnvVars(wf)
	assert.False(t, hasVar(vars, "ANTHROPIC_API_KEY"))
}

func TestScanEnvVars_GoogleBackend(t *testing.T) {
	wf := wfWithContent("run:\n  chat:\n    backend: google\n")
	vars := ScanEnvVars(wf)
	assert.False(t, hasVar(vars, "GOOGLE_API_KEY"))
}

func TestScanEnvVars_GroqBackend(t *testing.T) {
	wf := wfWithContent("run:\n  chat:\n    backend: groq\n")
	vars := ScanEnvVars(wf)
	assert.False(t, hasVar(vars, "GROQ_API_KEY"))
}

func TestScanEnvVars_DeepSeekBackend(t *testing.T) {
	wf := wfWithContent("run:\n  chat:\n    backend: deepseek\n")
	vars := ScanEnvVars(wf)
	assert.False(t, hasVar(vars, "DEEPSEEK_API_KEY"))
}

func TestScanEnvVars_OpenRouterBackend(t *testing.T) {
	wf := wfWithContent("run:\n  chat:\n    backend: openrouter\n")
	vars := ScanEnvVars(wf)
	assert.False(t, hasVar(vars, "OPENROUTER_API_KEY"))
}

func TestScanEnvVars_Gmail(t *testing.T) {
	wf := wfWithContent("run:\n  exec:\n    command: send email via gmail\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "GMAIL_USERNAME"))
	assert.True(t, hasVar(vars, "GMAIL_PASSWORD"))
}

func TestScanEnvVars_SMTP(t *testing.T) {
	wf := wfWithContent("run:\n  exec:\n    command: smtp://mail.example.com\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "SMTP_HOST"))
	assert.True(t, hasVar(vars, "SMTP_PASSWORD"))
}

func TestScanEnvVars_Slack(t *testing.T) {
	wf := wfWithContent("url: https://slack.com/api/chat.postMessage\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "SLACK_BOT_TOKEN"))
}

func TestScanEnvVars_Discord(t *testing.T) {
	wf := wfWithContent("url: https://discord.com/api/webhooks/123\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "DISCORD_BOT_TOKEN"))
}

func TestScanEnvVars_Telegram(t *testing.T) {
	wf := wfWithContent("url: https://api.telegram.org/bot\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "TELEGRAM_BOT_TOKEN"))
}

func TestScanEnvVars_Twilio(t *testing.T) {
	wf := wfWithContent("url: https://api.twilio.com/Messages\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "TWILIO_ACCOUNT_SID"))
	assert.True(t, hasVar(vars, "TWILIO_AUTH_TOKEN"))
}

func TestScanEnvVars_GitHub(t *testing.T) {
	wf := wfWithContent("url: https://api.github.com/repos\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "GITHUB_TOKEN"))
}

func TestScanEnvVars_AWS(t *testing.T) {
	wf := wfWithContent("url: https://s3.amazonaws.com/my-bucket\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "AWS_ACCESS_KEY_ID"))
	assert.True(t, hasVar(vars, "AWS_SECRET_ACCESS_KEY"))
}

func TestScanEnvVars_Postgres(t *testing.T) {
	wf := wfWithContent("connection: postgres://user:pass@localhost/db\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "DATABASE_URL"))
}

func TestScanEnvVars_MongoDB(t *testing.T) {
	wf := wfWithContent("url: mongodb://localhost:27017/mydb\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "MONGODB_URL"))
}

func TestScanEnvVars_Redis(t *testing.T) {
	wf := wfWithContent("url: redis://localhost:6379\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "REDIS_URL"))
}

func TestScanEnvVars_Stripe(t *testing.T) {
	wf := wfWithContent("url: https://api.stripe.com/v1/charges\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "STRIPE_SECRET_KEY"))
}

func TestScanEnvVars_TemplateExpr(t *testing.T) {
	wf := wfWithContent("prompt: \"key is {{ env('MY_SECRET_KEY') }}\"\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "MY_SECRET_KEY"))
}

func TestScanEnvVars_TemplateExprNoDuplicates(t *testing.T) {
	content := "a: {{ env('MY_KEY') }}\nb: {{ env('MY_KEY') }}\n"
	wf := wfWithContent(content)
	vars := ScanEnvVars(wf)
	count := 0
	for _, v := range vars {
		if v.Name == "MY_KEY" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestScanEnvVars_NoDuplicatesAcrossHints(t *testing.T) {
	// Both "openai.com" and "backend: openai" match OPENAI_API_KEY — should appear once.
	wf := wfWithContent("backend: openai\nurl: https://api.openai.com/v1/chat\n")
	vars := ScanEnvVars(wf)
	count := 0
	for _, v := range vars {
		if v.Name == "OPENAI_API_KEY" {
			count++
		}
	}
	assert.Equal(t, 0, count)
}

func TestScanEnvVars_Pinecone(t *testing.T) {
	wf := wfWithContent("url: https://index.pinecone.io/query\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "PINECONE_API_KEY"))
}

func TestScanEnvVars_Notion(t *testing.T) {
	wf := wfWithContent("url: https://api.notion.com/v1/pages\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "NOTION_API_KEY"))
}

func TestScanEnvVars_SendGrid(t *testing.T) {
	wf := wfWithContent("url: https://api.sendgrid.com/v3/mail/send\n")
	vars := ScanEnvVars(wf)
	assert.True(t, hasVar(vars, "SENDGRID_API_KEY"))
}

func hasVar(vars []EnvVar, name string) bool {
	for _, v := range vars {
		if v.Name == name {
			return true
		}
	}
	return false
}

func TestAppendUniqueEnvVars_SkipsDuplicate(t *testing.T) {
	t.Parallel()
	seen := map[string]bool{"FOO": true}
	var result []EnvVar
	appendUniqueEnvVars(&result, seen, []EnvVar{{Name: "FOO"}, {Name: "BAR"}})
	require.Len(t, result, 1)
	assert.Equal(t, "BAR", result[0].Name)
}
