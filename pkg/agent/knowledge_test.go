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

package agent

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/knowledge"
	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

// knowledgeLoop returns a loop with a sample vault whose card covers the
// test's working directory, and its graph built.
func knowledgeLoop(t *testing.T) (*Loop, string) {
	t.Helper()
	t.Setenv("KDEPS_DATA_DIR", t.TempDir())
	vault := t.TempDir()
	wd := t.TempDir()
	t.Chdir(wd)
	files := map[string]string{
		"records/projects/app.md":          "---\ntype: project\nstatus: active\nnext: ship\nupdated: 2026-10-01\n---\n| Source | Path |\n|---|---|\n| repo | " + wd + " |\n",
		"Projects/app/Billing decision.md": "---\ntype: decision\nsummary: Bill monthly.\n---\nSee [[Stripe gotcha]].\n",
		"Resources/Stripe gotcha.md":       "---\ntype: gotcha\nsummary: Webhooks retry for 3 days.\n---\n",
		"KDEPS.md":                         "# Workspace routing\n",
	}
	for p, c := range files {
		abs := filepath.Join(vault, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o750))
		require.NoError(t, os.WriteFile(abs, []byte(c), 0o600))
	}
	l := &Loop{registry: kdepstools.NewRegistry(), session: NewSession(10)}
	l.knowledge = newKnowledgeState(t.Context(), &config.KnowledgeConfig{Vault: vault})
	require.NotNil(t, l.knowledge)
	require.NoError(t, l.knowledge.build())
	l.registerKnowledgeTools()
	t.Cleanup(l.knowledge.waitBuilds)
	return l, vault
}

func TestNewKnowledgeState(t *testing.T) {
	t.Setenv("KDEPS_DATA_DIR", t.TempDir())
	assert.Nil(t, newKnowledgeState(t.Context(), nil), "library and test loops have no vault")
	off := false
	assert.Nil(t, newKnowledgeState(t.Context(), &config.KnowledgeConfig{Enabled: &off}))
	def := newKnowledgeState(t.Context(), &config.KnowledgeConfig{Vault: "  "})
	require.NotNil(t, def)
	assert.Equal(t, knowledge.DefaultVault(), def.vault.Root, "blank means the default vault")
	assert.DirExists(t, filepath.Join(def.vault.Root, "records", "projects"), "a new vault is set up")
	assert.FileExists(t, filepath.Join(def.vault.Root, "KDEPS.md"))
	t.Cleanup(func() { _ = os.RemoveAll(def.vault.Root) })
	k := newKnowledgeState(t.Context(), &config.KnowledgeConfig{Vault: "/v", Workspace: "/w", SessionNotes: &off})
	assert.Equal(t, "/v", k.vault.Root)
	assert.Equal(t, "/w", k.ws.Root)
	assert.False(t, k.sessionNotes())
	assert.True(t, newKnowledgeState(t.Context(), &config.KnowledgeConfig{Vault: "/v"}).sessionNotes())
	assert.Nil(t, k.idx(), "no graph yet")
	assert.Empty(t, k.recall("billing").Hits)
}

func TestKnowledgeRecallAndPreamble(t *testing.T) {
	l, vault := knowledgeLoop(t)
	var out bytes.Buffer
	item := l.knowledgeRecallItem("why do we bill monthly", &out)
	require.NotNil(t, item)
	assert.Equal(t, "system", item.Role)
	assert.Contains(t, item.Prompt, filepath.Join(vault, "Projects", "app", "Billing decision.md"))
	assert.Regexp(t, `recall: \d notes \(~\d+ tok\)`, out.String())
	assert.Nil(t, l.knowledgeRecallItem("  ", nil))
	assert.Nil(t, l.knowledgeRecallItem("zzzunrelated", nil))
	rec, root, ok := l.LastRecall()
	assert.True(t, ok)
	assert.Equal(t, vault, root)
	assert.Empty(t, rec.Hits, "the last recall found nothing")

	pre := l.knowledgePreamble()
	assert.Contains(t, pre, "<knowledge-vault>")
	assert.Contains(t, pre, "| app | project | active | ship | 2026-10-01 |")
	assert.Contains(t, pre, `belongs to card "app"`)
	assert.Empty(t, (&Loop{}).knowledgePreamble())
	assert.Nil(t, (&Loop{}).knowledgeRecallItem("x", nil))
	_, _, ok = (&Loop{}).LastRecall()
	assert.False(t, ok)

	assert.Contains(
		t,
		discoverInstructions(t.TempDir(), vault),
		"# Workspace routing",
		"the workspace KDEPS.md is loaded",
	)
}

func TestKnowledgeTools(t *testing.T) {
	l, vault := knowledgeLoop(t)
	call := func(name string, args map[string]any) (string, error) {
		tool := l.registry.Get(name)
		require.NotNil(t, tool, name)
		return tool.Execute(args)
	}
	out, err := call("knowledge_recall", map[string]any{"query": "stripe webhooks"})
	require.NoError(t, err)
	assert.Contains(t, out, "Stripe gotcha.md")
	out, err = call("knowledge_recall", map[string]any{"query": "zzz"})
	require.NoError(t, err)
	assert.Contains(t, out, "No notes")
	_, err = call("knowledge_recall", map[string]any{})
	require.Error(t, err)

	out, err = call("knowledge_card_get", map[string]any{})
	require.NoError(t, err)
	assert.Contains(t, out, "card: app")
	_, err = call("knowledge_card_get", map[string]any{"name": "nope"})
	require.Error(t, err)

	_, err = call("knowledge_card_update", map[string]any{"name": "app"})
	require.Error(t, err)
	out, err = call("knowledge_card_update", map[string]any{"name": "app", "status": "paused", "next": "review"})
	require.NoError(t, err)
	assert.Contains(t, out, "status: paused")
	assert.Contains(t, out, "next: review")

	_, err = call("knowledge_note_save", map[string]any{"title": "x"})
	require.Error(t, err)
	_, err = call("knowledge_note_save", map[string]any{"title": "a/b", "summary": "s", "body": "b"})
	require.Error(t, err)
	_, err = call("knowledge_note_save", map[string]any{"title": "Stripe gotcha", "summary": "s", "body": "b"})
	require.ErrorContains(t, err, "one home per fact")
	_, err = call("knowledge_note_save", map[string]any{"title": "x", "summary": "s", "body": "b", "folder": "../out"})
	require.ErrorContains(t, err, "outside the vault")
	out, err = call("knowledge_note_save", map[string]any{
		"title": "Retry budget", "summary": "Retries stop after 3.", "body": "See [[Stripe gotcha]].",
		"type": "gotcha", "tags": "billing, #ops", "links": "Stripe gotcha", "folder": "areas",
	})
	require.NoError(t, err)
	file := filepath.Join(vault, "Areas", "Retry budget.md")
	assert.Contains(t, out, file)
	data, _ := os.ReadFile(file)
	note := knowledge.ParseNote("Areas/Retry budget.md", data)
	assert.Equal(t, "gotcha", note.Type)
	assert.Equal(t, []string{"billing", "ops"}, note.Tags)
	assert.Equal(t, []string{"Stripe gotcha"}, note.MetaLinks)
	_, err = call(
		"knowledge_note_save",
		map[string]any{"title": "Plain", "summary": "s", "body": "b", "folder": "Sub/Dir"},
	)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(vault, "Sub", "Dir", "Plain.md"))

	t.Chdir(t.TempDir())
	_, err = call("knowledge_card_get", map[string]any{})
	require.ErrorContains(t, err, "no card covers")

	(&Loop{}).registerKnowledgeTools() // no registry, no vault: nothing to do
}

func TestKnowledgeSessionNotes(t *testing.T) {
	l, vault := knowledgeLoop(t)
	assert.False(t, l.SessionNoteDue(), "no turns yet")
	l.session.Append("hi", "hello")
	assert.True(t, l.SessionNoteDue())

	var out bytes.Buffer
	l.writeSessionNote("## Goal\nShip recall\n## Next Steps\n1. docs\n", &out)
	assert.Contains(t, out.String(), "knowledge: session note")
	assert.False(t, l.SessionNoteDue(), "the note covers the turns so far")
	matches, _ := filepath.Glob(filepath.Join(vault, "Projects", "app", "sessions", "*.md"))
	require.Len(t, matches, 1)
	data, _ := os.ReadFile(matches[0])
	assert.Contains(t, string(data), "session: unsaved-")
	card, _ := l.knowledge.ws.Card("app")
	assert.Equal(t, "docs", card.Next)

	out.Reset()
	l.writeSessionNote("no goal here", &out)
	assert.Contains(t, out.String(), "not written")
	l.writeSessionNote("", nil)
	(&Loop{}).writeSessionNote("## Goal\nx\n", nil)

	l.WriteSessionNoteNow(context.Background(), &out) // not due: no summary call

	id := l.sessionNoteID()
	assert.Equal(t, id, l.sessionNoteID(), "stable for the loop")
	l.knowledge.waitBuilds()
}

func TestCmdRecall(t *testing.T) {
	r := &REPL{loop: &Loop{}}
	out := captureStdout(t, func() { r.cmdRecall(nil) })
	assert.Contains(t, out, "knowledge.enabled: false")

	l, _ := knowledgeLoop(t)
	r = &REPL{loop: l}
	out = captureStdout(t, func() { r.cmdRecall(nil) })
	assert.Contains(t, out, "Nothing recalled")
	out = captureStdout(t, func() { r.cmdRecall([]string{"stripe", "webhooks"}) })
	assert.Contains(t, out, "Stripe gotcha.md")
	assert.Regexp(t, `\d notes, ~\d+ tokens`, out)
	out = captureStdout(t, func() { r.cmdRecall([]string{"bill", "monthly"}) })
	assert.True(t, strings.Contains(out, "hop from") || strings.Contains(out, "Billing decision"))
}

func TestBuildChatConfig_CarriesRecall(t *testing.T) {
	l, _ := knowledgeLoop(t)
	cfg := l.buildChatConfig(context.Background(), "why do we bill monthly", "preamble")
	var found bool
	for _, it := range cfg.Scenario {
		if strings.HasPrefix(it.Prompt, knowledgeRecallMarker) {
			found = true
		}
	}
	assert.True(t, found, "recall rides on the turn, not the cached preamble")
}

func TestWriteSessionNoteOnExit_NotDue(t *testing.T) {
	r := &REPL{loop: &Loop{}, loopCtx: context.Background()}
	out := captureStdout(t, r.writeSessionNoteOnExit)
	assert.Empty(t, out)
}

func TestKnowledgeToggle(t *testing.T) {
	l, _ := knowledgeLoop(t)
	assert.True(t, l.KnowledgeEnabled())
	l.session.Append("a", "b")
	l.SetKnowledge(false)
	assert.False(t, l.KnowledgeEnabled())
	assert.Empty(t, l.knowledgePreamble())
	assert.Nil(t, l.knowledgeRecallItem("bill monthly", nil))
	assert.False(t, l.SessionNoteDue())
	_, err := l.registry.Get("knowledge_recall").Execute(map[string]any{"query": "x"})
	require.ErrorIs(t, err, errKnowledgeOff)
	for _, name := range []string{"knowledge_card_get", "knowledge_card_update", "knowledge_note_save"} {
		_, err = l.registry.Get(name).Execute(map[string]any{})
		require.ErrorIs(t, err, errKnowledgeOff, name)
	}
	l.SetKnowledge(true)
	assert.True(t, l.SessionNoteDue())
	assert.False(t, (&Loop{}).KnowledgeEnabled())
}

func TestCmdKnowledge(t *testing.T) {
	out := captureStdout(t, func() { (&REPL{loop: &Loop{}}).cmdKnowledge(nil) })
	assert.Contains(t, out, "knowledge.enabled: false")

	l, vault := knowledgeLoop(t)
	var saved ToolTuning
	r := &REPL{loop: l}
	r.SetSaveTuningFn(func(tt ToolTuning) error { saved = tt; return nil })
	out = captureStdout(t, func() { r.cmdKnowledge(nil) })
	assert.Contains(t, out, "knowledge vault is on")
	assert.Contains(t, out, vault)
	out = captureStdout(t, func() { r.cmdKnowledge([]string{"off"}) })
	assert.Contains(t, out, "knowledge vault off")
	assert.True(t, saved.KnowledgeOff, "persisted")
	out = captureStdout(t, func() { r.cmdKnowledge(nil) })
	assert.Contains(t, out, "off - /knowledge on")
	out = captureStdout(t, func() { r.cmdRecall(nil) })
	assert.Contains(t, out, "is off (/knowledge on)")
	out = captureStdout(t, func() { r.cmdKnowledge([]string{"on"}) })
	assert.Contains(t, out, "knowledge vault on")
	assert.False(t, saved.KnowledgeOff)
	r.cmdKnowledge([]string{"bogus"})

	r.applyToolTuning(ToolTuning{KnowledgeOff: true})
	assert.False(t, l.KnowledgeEnabled(), "restored from the saved settings")
	l.knowledge.waitBuilds()
}
