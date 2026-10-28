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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathInVaultAndNoteRel(t *testing.T) {
	vault := filepath.Join(t.TempDir(), "notes")
	assert.True(t, PathInVault(vault, vault))
	assert.True(t, PathInVault(vault, filepath.Join(vault, "shared", "a.md")))
	assert.False(t, PathInVault(vault, t.TempDir()))
	assert.False(t, PathInVault("", vault))

	rel, ok := NoteRel(vault, filepath.Join(vault, "shared", "a.md"))
	require.True(t, ok)
	assert.Equal(t, "shared/a.md", rel)
	_, ok = NoteRel(vault, vault)
	assert.False(t, ok)
}

func TestLinksFrom_ReferenceTagAndStrongerEdge(t *testing.T) {
	vault := t.TempDir()
	ix := NewIndex(&Graph{
		Nodes: []*Node{
			{ID: "note:hub.md", Kind: KindNote, Note: "hub.md"},
			{ID: "note:linked.md", Kind: KindNote, Note: "linked.md"},
			{ID: "note:tagged.md", Kind: KindNote, Note: "tagged.md"},
			{ID: "tag:billing", Kind: KindTag, Label: "#billing"},
			{ID: "code:example.com/app", Kind: KindCode, Label: "example.com/app"},
		},
		Edges: []*Edge{
			{
				From: "note:hub.md", To: "note:linked.md", Relation: RelReferences,
				Provenance: ProvExtracted, Confidence: 1,
			},
			{
				From: "note:hub.md", To: "note:linked.md", Relation: RelMentions,
				Provenance: ProvInferred, Confidence: 0.4,
			},
			{
				From: "note:hub.md", To: "tag:billing", Relation: RelTagged,
				Provenance: ProvExtracted, Confidence: 1,
			},
			{
				From: "note:tagged.md", To: "tag:billing", Relation: RelTagged,
				Provenance: ProvExtracted, Confidence: 1,
			},
			{
				From: "note:hub.md", To: "code:example.com/app", Relation: RelDependsOn,
				Provenance: ProvExtracted, Confidence: 1,
			},
		},
	})

	links := ix.LinksFrom(vault, filepath.Join(vault, "hub.md"))
	require.Len(t, links, 2)
	byPath := map[string]FileLink{}
	for _, link := range links {
		byPath[link.Path] = link
	}
	linked := byPath[filepath.Join(vault, "linked.md")]
	assert.Equal(t, 1.0, linked.Weight, "the extracted reference beats the inferred edge")
	assert.Equal(t, RelReferences, linked.Relation)
	assert.Equal(t, "hub.md --references--> linked.md", linked.Chain)

	tagged := byPath[filepath.Join(vault, "tagged.md")]
	assert.InDelta(t, tagEdgeWeight, tagged.Weight, 0.0001)
	assert.Equal(t, RelTagged, tagged.Relation)
	assert.Equal(t, "hub.md --tagged--> billing --tagged--> tagged.md", tagged.Chain)

	assert.False(t, ix.HasNote(vault, filepath.Join(vault, "missing.md")))
	assert.True(t, ix.HasNote(vault, filepath.Join(vault, "hub.md")))
	assert.Nil(t, ix.LinksFrom(vault, filepath.Join(t.TempDir(), "hub.md")))
	assert.Nil(t, (*Index)(nil).LinksFrom(vault, filepath.Join(vault, "hub.md")))
}
