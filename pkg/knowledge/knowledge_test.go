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
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeVault writes files (vault-relative path -> content) under a temp dir.
func writeVault(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, c := range files {
		abs := filepath.Join(root, filepath.FromSlash(p))
		require.NoError(t, os.MkdirAll(filepath.Dir(abs), 0o750))
		require.NoError(t, os.WriteFile(abs, []byte(c), 0o600))
	}
	return root
}

func TestParseNote(t *testing.T) {
	n := ParseNote("3-Resources/Graph recall.md", []byte("\ufeff---\ntype: concept\nstatus: draft\n"+
		"summary: Walk the graph.\ntags: [knowledge, \"#graph\"]\naliases: recall\nlinks: [\"[[Hub|the hub]]\", plain]\n---\n"+
		"See [[One home#rule|one home]] and [doc](../x/Doc.md#a) and [web](https://x.io/a.md).\n"+
		"```\n[[In code]]\n```\nand `[[inline]]` too.\n"))
	assert.Equal(t, "Graph recall", n.Title)
	assert.Equal(t, "concept", n.Type)
	assert.Equal(t, "draft", n.Status)
	assert.Equal(t, "Walk the graph.", n.Summary)
	assert.Equal(t, []string{"knowledge", "graph"}, n.Tags)
	assert.Equal(t, []string{"recall"}, n.Aliases)
	assert.Equal(t, []string{"Hub", "plain"}, n.MetaLinks)
	assert.Equal(t, []string{"One home"}, n.Links, "links in code are not links")
	assert.Equal(t, []string{"../x/Doc.md"}, n.MDLinks)
	assert.Len(t, n.Hash, hashPrefix*2)

	plain := ParseNote("a.md", []byte("# no frontmatter [[x]]"))
	assert.Empty(t, plain.Meta)
	assert.Equal(t, []string{"x"}, plain.Links)
	bad := ParseNote("b.md", []byte("---\n: [bad\n---\nbody"))
	assert.Empty(t, bad.Meta)
	assert.Equal(t, "body", strings.TrimSpace(bad.Body))
	open := ParseNote("c.md", []byte("---\ntype: x\nno end"))
	assert.Empty(t, open.Meta)
	num := ParseNote("d.md", []byte("---\nstatus: 3\nsummary: [a, b]\n---\n"))
	assert.Equal(t, "3", num.Status)
	assert.Contains(t, num.Summary, "a")
}

func TestVaultFolderAndWalk(t *testing.T) {
	root := writeVault(t, map[string]string{
		"0-Inbox/a.md":           "a",
		"1. Projects/x/b.md":     "b",
		"Archive/c.md":           "c",
		".obsidian/hidden.md":    "h",
		"3_Resources/d.MD":       "d",
		"notes.txt":              "not a note",
		"Resources-but-not/e.md": "e",
		"3_Resources/.hidden.md": "h",
	})
	v := Vault{Root: root}
	assert.Equal(t, "0-Inbox", v.Folder(RoleInbox))
	assert.Equal(t, "1. Projects", v.Folder(RoleProjects))
	assert.Equal(t, "Archive", v.Folder(RoleArchives))
	assert.Equal(t, "3_Resources", v.Folder(RoleResources))
	assert.Equal(t, "Areas", v.Folder(RoleAreas), "default when missing")
	notes, err := v.Walk()
	require.NoError(t, err)
	var paths []string
	for _, n := range notes {
		paths = append(paths, n.Path)
	}
	assert.Equal(
		t,
		[]string{"0-Inbox/a.md", "1. Projects/x/b.md", "3_Resources/d.MD", "Archive/c.md", "Resources-but-not/e.md"},
		paths,
	)
	_, err = Vault{Root: filepath.Join(root, "missing")}.Walk()
	require.Error(t, err)
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, "x"), ExpandPath("~/x", "/base"))
	assert.Equal(t, filepath.Clean("/base/y"), ExpandPath("y", "/base"))
	assert.Equal(t, filepath.Clean("/abs"), ExpandPath("/abs/", "/base"))
}

const cardText = `---
type: project
status: active
next: ship it
updated: 2026-10-01
owner: me
---
| Source | Path |
|---|---|
| repo | ` + "`REPO`" + ` |
| vault | Projects/kdeps |
`

func TestCards(t *testing.T) {
	repo := t.TempDir()
	root := writeVault(t, map[string]string{
		"records/projects/kdeps.md":  strings.Replace(cardText, "REPO", repo, 1),
		"records/clients/acme.md":    "---\ntype: client\n---\nA story line.\nAnother.\nThird.\nFourth.\n| x |\n",
		"records/projects/notes.txt": "skip",
	})
	w := Workspace{Root: root}
	cards, err := w.Cards()
	require.NoError(t, err)
	require.Len(t, cards, 2)
	c, err := w.Card("KDEPS")
	require.NoError(t, err)
	assert.Equal(t, "project", c.Type)
	assert.Equal(t, "ship it", c.Next)
	assert.Equal(t, []Link{{"repo", repo}, {"vault", "Projects/kdeps"}}, c.Links)
	assert.Equal(t, "Projects/kdeps", c.Link("Vault"))
	assert.Empty(t, c.Link("nope"))
	acme, _ := w.Card("acme")
	assert.Equal(t, 4, acme.BodyLines)
	_, err = w.Card("nope")
	require.ErrorIs(t, err, ErrNoCard)

	assert.Equal(t, "kdeps", w.CardForDir(filepath.Join(repo, "pkg", "x")).Name)
	assert.Equal(t, "kdeps", w.CardForDir(repo).Name)
	assert.Nil(t, w.CardForDir(t.TempDir()))

	now := time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, c.Update("paused", "wait: for review", now))
	assert.Equal(t, "paused", c.Status)
	assert.Equal(t, "wait: for review", c.Next)
	assert.Equal(t, "2026-10-15", c.Updated)
	data, _ := os.ReadFile(c.File)
	assert.Contains(t, string(data), "owner: me", "other fields are kept")
	assert.Contains(t, string(data), "| vault | Projects/kdeps |", "the body is kept")
	require.NoError(t, c.Update("", "", now))
	assert.Equal(t, "paused", c.Status, "empty keeps the value")

	require.NoError(t, acme.Update("active", "", now))
	assert.Equal(t, "active", acme.Status, "a missing field is added")

	bare := filepath.Join(root, "records/clients/bare.md")
	require.NoError(t, os.WriteFile(bare, []byte("no frontmatter\n"), 0o600))
	b, err := ReadCard(bare)
	require.NoError(t, err)
	require.NoError(t, b.Update("new", "first", now))
	assert.Equal(t, "new", b.Status)

	broken := filepath.Join(root, "records/clients/broken.md")
	require.NoError(t, os.WriteFile(broken, []byte("---\ntype: x\nnever closed\n"), 0o600))
	bc, err := ReadCard(broken)
	require.NoError(t, err)
	require.Error(t, bc.Update("a", "", now))
	_, err = ReadCard(filepath.Join(root, "none.md"))
	require.Error(t, err)

	assert.Equal(t, "plain", yamlScalar("plain"))
	assert.Equal(t, `"a: b"`, yamlScalar("a: b"))
	assert.Equal(t, `""`, yamlScalar(""))
}

func TestCheck(t *testing.T) {
	root := writeVault(t, map[string]string{
		"records/projects/kdeps.md": "---\ntype: project\nstatus: active\n---\n| repo | x |\n",
		"Projects/kdeps/a.md":       "---\nsummary: Same fact.\nnext: do it\n---\n[[Missing]] [[b]]",
		"Projects/kdeps/b.md":       "---\nsummary: same fact.\n---\n",
		"Resources/no summary.md":   "body",
		"Inbox/quick.md":            "no summary needed in the inbox",
		"top.md":                    "no summary needed at the root",
	})
	findings, err := Check(Vault{Root: root}, Workspace{Root: root})
	require.NoError(t, err)
	var got []string
	for _, f := range findings {
		rel, _ := filepath.Rel(root, f.File)
		got = append(got, filepath.ToSlash(rel)+": "+f.Problem)
	}
	joined := strings.Join(got, "\n")
	assert.Contains(t, joined, "records/projects/kdeps.md: card has no next")
	assert.Contains(t, joined, "records/projects/kdeps.md: card has no updated")
	assert.Contains(t, joined, "Projects/kdeps/a.md: has a next step; it belongs in the project card")
	assert.Contains(t, joined, "Projects/kdeps/a.md: links to [[Missing]]")
	assert.Contains(t, joined, "has the same summary as")
	assert.Contains(t, joined, "Resources/no summary.md: has no summary")
	assert.NotContains(t, joined, "Inbox/quick.md")
	assert.NotContains(t, joined, "top.md")

	_, err = Check(Vault{Root: filepath.Join(root, "missing")}, Workspace{Root: filepath.Join(root, "missing")})
	require.Error(t, err)
}
