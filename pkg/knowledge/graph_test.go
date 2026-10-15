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
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildNow() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }

func sampleVault(t *testing.T) (Vault, string) {
	t.Helper()
	repo := t.TempDir()
	require.NoError(
		t,
		os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module example.com/app\n\ngo 1.26\n"), 0o600),
	)
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "store"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "vendor", "x"), 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(repo, "main.go"),
		[]byte(
			"package main\n\nimport (\n\t\"fmt\"\n\t\"example.com/app/store\"\n)\n\nfunc main() { fmt.Println(store.X) }\n",
		),
		0o600,
	))
	require.NoError(
		t,
		os.WriteFile(filepath.Join(repo, "store", "store.go"), []byte("package store\n\nconst X = 1\n"), 0o600),
	)
	require.NoError(t, os.WriteFile(filepath.Join(repo, "vendor", "x", "x.go"), []byte("package x\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "broken.go"), []byte("not go"), 0o600))
	root := writeVault(t, map[string]string{
		"projects/app/app.md":            "---\ntype: project\nstatus: active\nnext: x\nupdated: 2026-10-01\n---\n| Source | Path |\n|---|---|\n| repo | " + repo + " |\n",
		"projects/app/notes/Overview.md": "---\ntype: note\nsummary: Overview of app\n---\n[[Billing decision]]\n",
		"projects/app/decisions/Billing decision.md": "---\ntype: decision\nsummary: Bill monthly, not per call.\n" +
			"tags: [billing]\nlinks: [\"[[Stripe gotcha]]\"]\n---\nWhy: churn. See [notes](../../../shared/Pricing.md).\n",
		"shared/Stripe gotcha.md":               "---\ntype: gotcha\nsummary: Webhooks retry for 3 days.\ntags: [billing]\n---\n",
		"shared/Pricing.md":                     "---\nsummary: Pricing tiers.\naliases: [tiers]\n---\n[[Nowhere]]\n",
		"archive/old/notes/Billing decision.md": "---\nsummary: Old copy, far away.\n---\n",
	})
	return Vault{Root: root}, repo
}

func TestBuild_GraphAndCache(t *testing.T) {
	v, _ := sampleVault(t)
	dir := t.TempDir()
	g, st, err := Build(t.Context(), v, dir, buildNow())
	require.NoError(t, err)
	assert.Equal(t, 6, st.Notes)
	assert.Equal(t, 6, st.Changed)
	assert.Equal(t, 3, st.CodeFiles, "vendor is skipped")
	edges := map[string]string{}
	for _, e := range g.Edges {
		edges[e.From+" -> "+e.To] = e.Relation
		assert.Equal(t, ProvExtracted, e.Provenance)
	}
	assert.Equal(
		t,
		RelReferences,
		edges["note:projects/app/notes/Overview.md -> note:projects/app/decisions/Billing decision.md"],
		"the closest of two same-titled notes wins",
	)
	assert.Equal(
		t,
		RelRelatesTo,
		edges["note:projects/app/decisions/Billing decision.md -> note:shared/Stripe gotcha.md"],
	)
	assert.Equal(t, RelReferences, edges["note:projects/app/decisions/Billing decision.md -> note:shared/Pricing.md"])
	assert.Equal(t, RelTagged, edges["note:shared/Stripe gotcha.md -> tag:billing"])
	assert.Equal(t, RelDependsOn, edges["code:example.com/app -> code:example.com/app/store"])
	assert.Equal(t, RelReferences, edges["note:projects/app/app.md -> code:example.com/app"])
	assert.Equal(t, []string{"Nowhere"}, g.Missing)

	loaded, err := Load(dir)
	require.NoError(t, err)
	assert.Len(t, loaded.Nodes, len(g.Nodes))
	assert.FileExists(t, GraphPath(dir))

	_, st, err = Build(t.Context(), v, dir, buildNow())
	require.NoError(t, err)
	assert.Zero(t, st.Changed, "unchanged notes come from the cache")
	require.NoError(
		t,
		os.WriteFile(filepath.Join(v.Root, "shared/Pricing.md"), []byte("---\nsummary: New tiers.\n---\n"), 0o600),
	)
	_, st, err = Build(t.Context(), v, dir, buildNow())
	require.NoError(t, err)
	assert.Equal(t, 1, st.Changed)

	_, err = Load(t.TempDir())
	require.Error(t, err)
	_, _, err = Build(t.Context(), Vault{Root: filepath.Join(v.Root, "missing")}, dir, buildNow())
	require.Error(t, err)
}

func TestResolver(t *testing.T) {
	r := newResolver([]*Note{
		{Path: "A/Note.md", Title: "Note"},
		{Path: "B/C/Note.md", Title: "Note", Aliases: []string{"alias"}},
		{Path: "B/Other.md", Title: "Other"},
	})
	assert.Equal(t, "A/Note.md", r.wiki("A/Note", "x.md"))
	assert.Equal(t, "B/C/Note.md", r.wiki("C/Note.md", "x.md"), "path suffix")
	assert.Equal(t, "B/C/Note.md", r.wiki("Note", "B/C/x.md"), "closest")
	assert.Equal(t, "B/C/Note.md", r.wiki("Alias", "x.md"))
	assert.Equal(t, "B/Other.md", r.wiki("Z/Other", "x.md"), "unknown folder falls back to the title")
	assert.Empty(t, r.wiki("", "x.md"))
	assert.Empty(t, r.wiki("Missing", "x.md"))
	assert.Equal(t, "B/Other.md", r.relative("../Other.md", "B/C/x.md"))
	assert.Empty(t, r.relative("nope.md", "x.md"))
}

func TestRecall(t *testing.T) {
	v, _ := sampleVault(t)
	g, _, err := Build(t.Context(), v, t.TempDir(), buildNow())
	require.NoError(t, err)
	ix := NewIndex(g)

	r := ix.Recall("why do we bill monthly?", 0, 0)
	require.NotEmpty(t, r.Hits)
	assert.Equal(t, "projects/app/decisions/Billing decision.md", r.Hits[0].Note, "the decision itself first")
	var notes []string
	for _, h := range r.Hits {
		notes = append(notes, h.Note)
	}
	assert.Contains(t, notes, "shared/Stripe gotcha.md", "a linked gotcha comes along")
	assert.Positive(t, r.Tokens)

	text := r.Format(v.Root)
	assert.True(t, strings.HasPrefix(text, "<knowledge-recall>"))
	assert.Contains(t, text, filepath.Join(v.Root, "projects", "app", "decisions", "Billing decision.md"))
	assert.Contains(t, text, "Bill monthly, not per call.")
	assert.Contains(t, text, "(linked from")

	tiny := ix.Recall("billing", 1, 5)
	assert.LessOrEqual(t, tiny.Tokens, 5)

	none := ix.Recall("the and for", 2, 100)
	assert.Empty(t, none.Hits)
	assert.Empty(t, none.Format(v.Root))

	assert.NotEmpty(t, ix.Recall("tiers", 1, 100).Hits, "aliases match")
}

func TestRecall_EdgeWeights(t *testing.T) {
	assert.InDelta(t, 1.0, edgeWeight(&Edge{Provenance: ProvExtracted}), 1e-9)
	assert.InDelta(t, 0.4, edgeWeight(&Edge{Provenance: ProvInferred, Confidence: 0.4}), 1e-9)
	assert.InDelta(t, ambiguousWeight, edgeWeight(&Edge{Provenance: ProvAmbiguous}), 1e-9)
	assert.InDelta(t, tagEdgeWeight, edgeWeight(&Edge{Provenance: ProvExtracted, Relation: RelTagged}), 1e-9)
	assert.Contains(t, words("parseNoteFrontmatter graph_recall"), "frontmatter")
	assert.Contains(t, words("parseNoteFrontmatter graph_recall"), "recall")
}

func TestInit(t *testing.T) {
	root := writeVault(t, map[string]string{"projects/keep.md": "x", "KDEPS.md": "mine"})
	created, err := Init(Vault{Root: root})
	require.NoError(t, err)
	for _, d := range []string{"shared", "inbox", "archive"} {
		assert.DirExists(t, filepath.Join(root, d))
		assert.Contains(t, created, filepath.Join(root, d))
	}
	assert.NotContains(t, created, filepath.Join(root, "projects"), "existing folders are kept")
	assert.FileExists(t, filepath.Join(root, "schema.md"))
	data, _ := os.ReadFile(filepath.Join(root, "KDEPS.md"))
	assert.Equal(t, "mine", string(data), "existing files are never changed")
	assert.NotContains(t, created, filepath.Join(root, "KDEPS.md"))

	again, err := Init(Vault{Root: root})
	require.NoError(t, err)
	assert.Empty(t, again)
}

const summaryText = `## Goal
- Ship knowledge recall for kdeps

## Constraints & Preferences
(none)

## Progress
### Done
- [x] graph build
### In Progress
- [ ] docs

## Key Decisions
- **Louvain first**: simpler

## Mistakes
- cache kept stale parses

## Wins
(none)

## Next Steps
1. write the docs
2. publish

## Critical Context
- none
`

func TestWriteSession(t *testing.T) {
	v, repo := sampleVault(t)
	card := v.CardForDir(filepath.Join(repo, "store"))
	require.NotNil(t, card)
	sn, err := WriteSession(card, "s1", summaryText, buildNow())
	require.NoError(t, err)
	assert.True(t, sn.New)
	assert.Equal(t, "app", sn.Card.Name)
	assert.Equal(t, filepath.Join(v.Root, "projects", "app", "sessions"), filepath.Dir(sn.File))
	data, err := os.ReadFile(sn.File)
	require.NoError(t, err)
	note := string(data)
	assert.Contains(t, note, "type: session")
	assert.Contains(t, note, "session: s1")
	assert.Contains(t, note, "project: app")
	assert.Contains(t, note, `links: ["[[app]]"]`, "the session links its project card")
	assert.Contains(t, note, "summary: Ship knowledge recall for kdeps")
	assert.Contains(t, note, "## What was built\n\n- [x] graph build")
	assert.Contains(t, note, "## Mistakes\n\n- cache kept stale parses")
	assert.Contains(t, note, "## Decisions\n\n- **Louvain first**")
	assert.Contains(t, note, "## Wins\n\n(none)")
	assert.NotContains(t, note, "write the docs", "the next step lives in the card")
	got, _ := v.Card("app")
	assert.Equal(t, "write the docs", got.Next)

	again, err := WriteSession(
		card,
		"s1",
		strings.Replace(summaryText, "graph build", "graph build v2", 1),
		buildNow().Add(time.Hour),
	)
	require.NoError(t, err)
	assert.False(t, again.New)
	assert.Equal(t, sn.File, again.File, "one note per session, rewritten in place")

	_, err = WriteSession(card, "s3", "## Goal\n(none)\n", buildNow())
	require.Error(t, err)
}

func TestWriteDecisions(t *testing.T) {
	v, repo := sampleVault(t)
	card := v.CardForDir(repo)
	require.NotNil(t, card)
	summary := "## Goal\nx\n\n## Key Decisions\n- **Louvain first**: simpler than Leiden\n" +
		"- keep the cache per note hash because rebuilds were slow\n- (none)\n"
	session := filepath.Join(card.Dir(), "sessions", "2026-10-15 0900 x.md")
	written, err := WriteDecisions(card, session, summary, buildNow())
	require.NoError(t, err)
	require.Len(t, written, 2)
	data, _ := os.ReadFile(filepath.Join(card.Dir(), "decisions", "Louvain first.md"))
	note := string(data)
	assert.Contains(t, note, "type: decision")
	assert.Contains(t, note, `summary: "Louvain first: simpler than Leiden"`)
	assert.Contains(t, note, `links: ["[[2026-10-15 0900 x]]", "[[app]]"]`)
	assert.FileExists(t, filepath.Join(card.Dir(), "decisions", "keep the cache per note hash because rebuilds.md"),
		"no bold lead: the first words")

	again, err := WriteDecisions(card, "s.md", summary, buildNow())
	require.NoError(t, err)
	assert.Empty(t, again, "a decision with a note already is left alone (one home per fact)")
	none, err := WriteDecisions(card, "s.md", "## Goal\nx\n", buildNow())
	require.NoError(t, err)
	assert.Empty(t, none)
}

func TestEnsureProject(t *testing.T) {
	v := Vault{Root: t.TempDir()}
	plain := filepath.Join(t.TempDir(), "notes app")
	require.NoError(t, os.MkdirAll(plain, 0o750))
	c, created, err := v.EnsureProject(plain, buildNow())
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, "notes app", c.Name)
	assert.Equal(t, filepath.Join(v.Root, "projects", "notes app", "notes app.md"), c.File)
	assert.Equal(t, "project", c.Type)
	assert.Equal(t, "active", c.Status)
	assert.Equal(t, "(none)", c.Next)
	assert.Equal(t, []Link{{SourceFolder, plain}}, c.Links)
	findings, err := Check(v)
	require.NoError(t, err)
	assert.Empty(t, findings, "a created project passes the check")

	same, created, err := v.EnsureProject(filepath.Join(plain, "sub"), buildNow())
	require.NoError(t, err)
	assert.False(t, created, "a subfolder belongs to its project")
	assert.Equal(t, c.File, same.File)

	repo := filepath.Join(t.TempDir(), "svc")
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0o750))
	require.NoError(t, os.MkdirAll(filepath.Join(repo, "cmd", "x"), 0o750))
	r, _, err := v.EnsureProject(filepath.Join(repo, "cmd", "x"), buildNow())
	require.NoError(t, err)
	assert.Equal(t, "svc", r.Name, "a git repository is named by its root")
	assert.Equal(t, []Link{{SourceRepo, repo}}, r.Links)

	other := filepath.Join(t.TempDir(), "elsewhere", "svc")
	require.NoError(t, os.MkdirAll(other, 0o750))
	o, created, err := v.EnsureProject(other, buildNow())
	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, "elsewhere-svc", o.Name, "a taken name gets the parent folder in front")

	require.NoError(t, v.SetStatus(r, StatusArchived, "", buildNow()))
	assert.True(t, r.Archived())
	assert.Equal(t, filepath.Join(v.Root, "archive", "svc", "svc.md"), r.File)
	assert.NoDirExists(t, filepath.Join(v.Root, "projects", "svc"))
	back, created, err := v.EnsureProject(repo, buildNow())
	require.NoError(t, err)
	assert.False(t, created)
	assert.True(t, back.Archived(), "an archived project is found where it is")
	require.NoError(t, v.Reopen(back, buildNow()))
	require.NoError(t, v.Reopen(back, buildNow()), "reopening an active project does nothing")
	assert.False(t, back.Archived(), "reopening brings it back")
	assert.Equal(t, "active", back.Status)
	assert.FileExists(t, filepath.Join(v.Root, "projects", "svc", "svc.md"))

	assert.True(t, IsCardPath("projects/svc/svc.md"))
	assert.True(t, IsCardPath("archive/svc/svc.md"))
	assert.False(t, IsCardPath("projects/svc/notes/svc.md"))
	assert.Equal(t, "svc", ProjectOf("projects/svc/memory/x.md"))
	assert.Empty(t, ProjectOf("shared/x.md"))
	assert.Equal(t, "root", safeName("///"))
}

func TestAppendRun(t *testing.T) {
	v := Vault{Root: t.TempDir()}
	wf := t.TempDir()
	card, _, err := v.CreateProject("invoices", SourceWorkflow, wf, buildNow())
	require.NoError(t, err)
	file, err := AppendRun(card, "summarize\nthe doc", "a summary", buildNow())
	require.NoError(t, err)
	_, err = AppendRun(card, "second", "map[answer:ok]", buildNow().Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(card.Dir(), "sessions", "2026-10-15 invoices runs.md"), file)
	data, _ := os.ReadFile(file)
	note := string(data)
	assert.Contains(t, note, "type: session")
	assert.Contains(t, note, "- 09:00:00 summarize -> a summary\n")
	assert.Contains(t, note, "- 09:01:00 second -> map[answer:ok]\n")
	assert.Equal(t, card.File, v.CardForDir(wf).File, "the workflow folder maps to its project")
}

func TestSummaryHelpers(t *testing.T) {
	sec := SummarySections("## Goal\nA\n## Progress\nmade progress\n## Wins\n(none)\n")
	assert.Equal(t, "made progress", sec["progress"])
	assert.Empty(t, sec["wins"])
	assert.Equal(t, "Fix the bug", oneLine("\n- [ ] **Fix the bug**\n"))
	assert.Len(t, []rune(oneLine(strings.Repeat("x", 300))), maxSummary)
	assert.Empty(t, firstItem("  "))
	assert.Equal(t, "session", slug("///"))
	assert.Len(t, []rune(slug(strings.Repeat("y", 100))), maxSlug)
}

func TestCode_GitIgnoredFilesSkipped(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo := t.TempDir()
	for p, c := range map[string]string{
		"go.mod":            "module example.com/g\n",
		"main.go":           "package main\n",
		".gitignore":        "private/\n",
		"private/secret.go": "package secret\n",
		"lib/lib.go":        "package lib\n",
		"vendor/v/v.go":     "package v\n",
	} {
		abs := filepath.Join(repo, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o750))
		require.NoError(t, os.WriteFile(abs, []byte(c), 0o600))
	}
	require.NoError(t, exec.Command("git", "-C", repo, "init", "-q").Run())
	files := gitGoFiles(t.Context(), repo)
	var rel []string
	for _, f := range files {
		r, _ := filepath.Rel(repo, f)
		rel = append(rel, filepath.ToSlash(r))
	}
	assert.ElementsMatch(t, []string{"main.go", "lib/lib.go"}, rel, "ignored and vendored files are skipped")
	assert.Empty(t, gitGoFiles(t.Context(), t.TempDir()), "not a git work tree, or ignored by an enclosing one")
	assert.Len(t, walkGoFiles(repo), 3, "without git: everything outside vendor")
}

func TestRecall_CuratedNotesOutrankAutoWritten(t *testing.T) {
	g := &Graph{Nodes: []*Node{
		{
			ID: "note:projects/app/sessions/s1.md", Label: "why do we bill monthly",
			Kind: "session", Note: "projects/app/sessions/s1.md",
		},
		{
			ID: "note:projects/app/memory/fact-billing.md", Label: "fact-billing",
			Kind: "fact", Note: "projects/app/memory/fact-billing.md", Summary: "bill monthly",
		},
		{
			ID: "note:projects/app/decisions/Billing decision.md", Label: "Billing decision",
			Kind: "decision", Note: "projects/app/decisions/Billing decision.md", Summary: "Bill monthly, not per call",
		},
	}}
	ix := NewIndex(g)
	r := ix.Recall("why do we bill monthly", 1, 0)
	var notes []string
	for _, h := range r.Hits {
		notes = append(notes, h.Note)
	}
	require.Contains(
		t,
		notes,
		"projects/app/decisions/Billing decision.md",
		"the curated decision is recalled with the session",
	)
	assert.Less(
		t,
		slices.Index(notes, "projects/app/decisions/Billing decision.md"),
		slices.Index(notes, "projects/app/memory/fact-billing.md"),
		"a curated note outranks a memory entry on the same words",
	)

	r = ix.Recall("why do we bill monthly", 1, 0, "projects/app/sessions/s1.md")
	for _, h := range r.Hits {
		assert.NotEqual(t, "projects/app/sessions/s1.md", h.Note, "excluded notes are never recalled")
	}
	assert.InDelta(t, 1.0, sourceWeight(&Node{Kind: "decision", Note: "x.md"}), 1e-9)
	assert.InDelta(t, sessionNoteWeight, sourceWeight(&Node{Kind: "session"}), 1e-9)
	assert.InDelta(t, memoryNoteWeight, sourceWeight(&Node{Note: "projects/app/memory/x.md"}), 1e-9)
	for in, want := range map[string]string{
		"billing": "bill", "invoices": "invoice", "deployed": "deploy", "bus": "bus", "process": "process", "notes": "note",
	} {
		assert.Equal(t, want, stem(in), in)
	}
}

func TestGitRoot_Ceilings(t *testing.T) {
	outer := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(outer, ".git"), 0o750))
	inner := filepath.Join(outer, "a", "b")
	require.NoError(t, os.MkdirAll(inner, 0o750))
	t.Setenv("GIT_CEILING_DIRECTORIES", "")
	assert.Equal(t, outer, gitRoot(inner))
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Join(outer, "a"))
	assert.Empty(t, gitRoot(inner), "git does not look into a ceiling directory")
	assert.Equal(t, outer, gitRoot(outer), "the start folder itself is always checked")
	t.Setenv("GIT_CEILING_DIRECTORIES", "")
	t.Setenv("HOME", outer)
	t.Setenv("USERPROFILE", outer)
	assert.Empty(t, gitRoot(inner), "a home kept in git does not name the folders under it")
	assert.Equal(t, outer, gitRoot(outer), "working in the home folder itself still finds it")
}

func TestCopyTree(t *testing.T) {
	from := writeVault(t, map[string]string{"a.md": "A", "sub/b.md": "B"})
	to := filepath.Join(t.TempDir(), "copy")
	require.NoError(t, copyTree(from, to))
	data, err := os.ReadFile(filepath.Join(to, "sub", "b.md"))
	require.NoError(t, err)
	assert.Equal(t, "B", string(data))
	require.Error(t, copyTree(filepath.Join(from, "missing"), to))
}
