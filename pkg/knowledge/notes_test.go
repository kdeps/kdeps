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

package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func shippedNoteRules(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "assets", knowledgeSet, notesItem+".yaml"))
	require.NoError(t, err)
	return string(data)
}

func TestParseNoteRules_RejectsBadFiles(t *testing.T) {
	good := shippedNoteRules(t)
	_, goodErr := parseNoteRules([]byte(good))
	require.NoError(t, goodErr)
	cases := map[string]string{
		"missing template": strings.Replace(good, `runs_file: "{{.Created}} {{.Project}} runs"`, "", 1),
		"bad template":     strings.Replace(good, "- {{.Time}} {{.Prompt}}", "- {{.Time {{.Prompt}}", 1),
		"no sections":      strings.Replace(good, "session_sections:", "session_sections: []\nunused:", 1),
		"no next":          strings.Replace(good, "next: next steps", "next: \"\"", 1),
		"broken YAML":      "summary: [",
	}
	for name, text := range cases {
		require.NotEqual(t, good, text, name+": the fixture must change the file")
		_, err := parseNoteRules([]byte(text))
		assert.Error(t, err, name)
	}
}

func TestNoteRules_CardUsesTheLayoutKeys(t *testing.T) {
	r, err := parseNoteRules([]byte(shippedNoteRules(t)))
	require.NoError(t, err)
	keys := Layout().Card
	keys.Type, keys.Next = "kind", "todo"
	keys.LinksHeader = []string{"Kind", "Where"}
	body, err := r.render(tmplCard, map[string]any{
		"Keys": keys, "None": r.None, "Updated": "2026-10-15", "Source": "repo", "Place": "/p",
	})
	require.NoError(t, err)
	assert.Contains(t, body, "kind: project\n", "the card's type key comes from vault.yaml")
	assert.Contains(t, body, "todo: (none)\n")
	assert.Contains(t, body, "| Kind | Where |\n", "the places table header comes from vault.yaml")
}

func TestSummarySections_FollowNotesYAML(t *testing.T) {
	r, err := parseNoteRules([]byte(strings.Replace(shippedNoteRules(t), `none: "(none)"`, `none: "n/a"`, 1)))
	require.NoError(t, err)
	sec := summarySections("## Goal\nship\n## Done\nbuilt it\n## Wins\nn/a\n", r)
	assert.Equal(t, "built it", sec["progress"], "progress is filled from done")
	assert.Empty(t, sec["wins"], "the placeholder from notes.yaml reads as empty")
	assert.Equal(t, []string{"a"}, listItems("- a\n- n/a\n", r.None))
}

func TestKnowledgeSpec_ValidatesTheNotesItem(t *testing.T) {
	spec := knowledgeSpec()
	require.Contains(t, spec.Required, notesItem)
	require.NoError(t, spec.Validate([]byte(shippedNoteRules(t)), notesItem))
	require.Error(t, spec.Validate([]byte("summary: {}\n"), notesItem))
}

func TestNoteRules_SessionKeysAreWrittenAndReadBack(t *testing.T) {
	text := strings.Replace(shippedNoteRules(t), "id_key: session", "id_key: sid", 1)
	text = strings.Replace(text, "commits_key: commits", "commits_key: range", 1)
	r, err := parseNoteRules([]byte(text))
	require.NoError(t, err)

	body, err := sessionBody(r, "s-9", "app", "[[app]]", "x", map[string]string{"goal": "x"},
		buildNow(), GitRange{From: "a1", To: "b2"})
	require.NoError(t, err)
	assert.Contains(t, body, "\nsid: s-9\n")
	assert.Contains(t, body, "\nrange: a1..b2\n")

	dir := t.TempDir()
	file := filepath.Join(dir, "note.md")
	require.NoError(t, os.WriteFile(file, []byte(body), 0o600))
	assert.Equal(t, file, findSession(dir, "s-9", r), "the reader finds the note by the renamed key")
	assert.Equal(t, "a1", recordedGitStart(file, r))
}

func TestParseNoteRules_RejectsAMissingSessionBlock(t *testing.T) {
	text := strings.Replace(shippedNoteRules(t), "  kind: session  #", "  kind: \"\"  #", 1)
	_, err := parseNoteRules([]byte(text))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session.kind")
}

func TestSavedNoteText(t *testing.T) {
	now := buildNow()
	got, err := SavedNoteText(SavedNote{Type: "note", Title: "Retry", Summary: `say "hi"`, Body: "Body."}, now)
	require.NoError(t, err)
	assert.Equal(t, "---\ntype: note\nstatus: draft\ncreated: "+now.Format("2006-01-02")+
		"\nsummary: \"say \\\"hi\\\"\"\n---\n\n# Retry\n\nBody.\n", got)

	got, err = SavedNoteText(SavedNote{
		Type: "decision", Title: "T", Summary: "s", Body: "b",
		Tags: []string{"a", "b"}, Links: []string{"[[x/y]]", "[[z]]"},
	}, now)
	require.NoError(t, err)
	assert.Contains(t, got, "summary: \"s\"\ntags: [a, b]\nlinks: [\"[[x/y]]\", \"[[z]]\"]\n---\n")
}

func TestNoteLink_ResolvesLikeTheGraph(t *testing.T) {
	v, _ := sampleVault(t)
	notes, err := v.Walk()
	require.NoError(t, err)
	n := notes[0]
	want := WikiLink(v.Root, filepath.Join(v.Root, filepath.FromSlash(n.Path)))
	assert.Equal(t, want, NoteLink(v, notes, n.Title, ""), "by title")
	assert.Equal(
		t,
		want,
		NoteLink(v, notes, strings.ToUpper(strings.TrimSuffix(n.Path, ".md")), ""),
		"by path, any case",
	)
	assert.Equal(t, "[[no such note]]", NoteLink(v, notes, "no such note.md", ""))
}

func TestNoteRules_SessionFileTimeRequired(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "assets", "knowledge", "notes.yaml"))
	require.NoError(t, err)
	r, err := parseNoteRules(data)
	require.NoError(t, err)
	assert.Equal(t, "2006-01-02 1504", r.Session.FileTime)
	_, err = parseNoteRules([]byte(strings.Replace(string(data), "file_time:", "file_time_unused:", 1)))
	require.ErrorContains(t, err, "session.file_time")
}
