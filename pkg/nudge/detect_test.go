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

// Package bodylimit checks a request body against its provider's size limit
// before a model call, in both modes.
package nudge_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/nudge"
)

func TestAdmitsFailure(t *testing.T) {
	assert.True(t, nudge.AdmitsFailure("I was not able to make that edit; old_string never matched."))
	assert.True(t, nudge.AdmitsFailure("The command did not finish."))
	assert.False(t, nudge.AdmitsFailure("I can't emit a real executable invoke."))
	assert.False(t, nudge.AdmitsFailure("Done. The file has been updated."))
}

func TestSelfLimit(t *testing.T) {
	yes := []string{
		"I don't actually have access to any kdeps tools",
		"I can't perform that warm-up",
		"I cannot do that",
		"I can't actually execute",
		"I can't emit a real executable invoke",
		"I don't have a way to call the tool",
		"bash_exec is not present in the tool list",
		"I'm unable to read the repository",
		"I am not able to run commands",
		"As an AI I cannot access your files",
		"That is beyond my capabilities",
		"I won't be able to list the files",
		"I can\u2019t emit an invoke",
		"Sorry, that's not something I can help with.",
		"I'm afraid that isn't possible from here.",
		"My apologies, I lack the ability to read your files.",
		"Unfortunately I am limited to answering in text.",
		"I don't think I can list those files.",
		"You'll have to run that command yourself.",
		"I recommend you do this manually on your machine.",
		"There is no filesystem available to me.",
		"This environment has no shell.",
		"I'm restricted from running commands.",
		"I am not permitted to access the workspace.",
		"That's outside what I can do.",
		"I wasn't built to execute code.",
		"I'm not sure the tools are available for that.",
		"It looks like tool use isn't available in this session.",
		"I don't support shell access.",
		"That request is out of scope for me.",
		"I'm only able to discuss the code, not change it.",
		"The best I can do is describe the command for you to paste.",
		"Sorry, no can do.",
		"I lack permission to read the workspace.",
		"Filesystem access isn't part of my setup.",
		"I operate without tools in this chat.",
		"Please run ls locally; I have nothing to execute with.",
		"I must decline. I shouldn't touch your files.",
		"This won't work because the tools aren't here.",
		"I'd love to help, but my hands are tied.",
		"Let me explain how you could do it instead of me doing it.",
		"I apologize, but that exceeds what this assistant is allowed to do.",
		"Hmm, I don't see a way to read that file.",
		"There isn't a tool for reading files.",
		"My environment doesn't include a terminal.",
		"I can describe the change, but applying it is on you.",
		"Afraid I have to stop here.",
		"I am not designed to run shell commands.",
		"The sandbox I have is empty, so your repository is not visible.",
		"The session directory and the repo are not accessible in this execution environment.",
		"The session directory is unavailable in this execution environment.",
		"The workspace is not available in this chat.",
		"The repository is unavailable to the tools in this execution environment.",
	}
	for _, s := range yes {
		assert.Truef(t, nudge.SelfLimit(s), "%q", s)
	}
	no := []string{
		"",
		"Listed the files.",
		"I don't know.",
		"The tests failed in TestSearchLocal.",
		"I will not delete the production database or run rm -rf.",
		"I can't do that. I will not run rm -rf on the disk.",
		"I refuse to force-push main.",
		"I won't drop the production database.",
		"I will not overwrite the production database.",
		"We're temporarily unable to respond to this volume of requests. Please try again later.",
		"Sorry for the wait. The title is kdeps.",
		"Done. I updated main.go.",
		"Unfortunately I don't think this is going to work, but here is the file list: README.md.",
		"I don't know which file you mean. Name it and I will read it.",
	}
	for _, s := range no {
		assert.Falsef(t, nudge.SelfLimit(s), "%q", s)
	}
}

func TestAck(t *testing.T) {
	yes := []string{
		"The limit I named was not real. The tool call succeeded, and I should have continued from its output instead of treating the work as blocked.",
		"The limit I named was not real. I should have continued.",
		"You are right about the limit you named. I will call the tool now.",
		"Understood. I should call the tool.",
	}
	for _, s := range yes {
		assert.Truef(t, nudge.Ack(s), "%q", s)
	}
	long := "The limit I named was not real, but the remaining work is the analysis: " +
		strings.Repeat("session 15 event 36, ", 40)
	no := []string{
		"",
		"Listed the files.",
		"You're right, the bug is in loop.go:2414. The fix is to check the budget first.",
		"I don't know which file you mean. Name it and I will read it.",
		long,
		"The limit I named was not real.\n```go\nfunc f() {}\n```",
	}
	for _, s := range no {
		assert.Falsef(t, nudge.Ack(s), "%q", s)
	}
}

func TestSelfLimit_CouldNotAccessEnvironment(t *testing.T) {
	assert.True(t, nudge.SelfLimit("I couldn't complete the fix because `calc.py` is not accessible "+
		"in the available project environment. Only `notes.md` is present."))
}

func TestUnbackedClaim(t *testing.T) {
	for _, s := range []string{
		"Created: notes.md", "I've remembered it.", "I fixed the add function.", "Done: all set",
		"Created [notes.md](https://example.invalid/notes.md) containing exactly: hello",
		"Remembered: keep `calc.py` dependency-free.",
		"Got it. I'll remember that the project decision is to keep calc.py dependency-free.",
		"## Test result\n\nCommand run:\n\n```bash\npython3 -m pytest -q\n```",
		"`pytest` exited with **code 5**, which means no tests were collected.",
		"I can't show it because running:\n\n```bash\ngit diff --stat\n```\n\nreturned:\n\n```text\nwarning",
		"This decision has been stored as a project preference.",
	} {
		assert.Truef(t, nudge.UnbackedClaim(s), "%q", s)
	}
	for _, s := range []string{
		"What would you like me to create?",
		"The add function subtracts instead of adding.",
		"Use git status to check.",
	} {
		assert.Falsef(t, nudge.UnbackedClaim(s), "%q", s)
	}
}

func TestAcknowledgesFailure(t *testing.T) {
	assert.True(t, nudge.AcknowledgesFailure("The edit failed: old_string not found."))
	assert.True(t, nudge.AcknowledgesFailure("I was unable to write the file."))
	assert.False(t, nudge.AcknowledgesFailure("Done. The file is updated."))
}
