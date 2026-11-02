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
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	kgstore "github.com/kdeps/kartographer/store"

	kgknowledge "github.com/kdeps/kartographer/knowledge"
)

// Node kinds and edge relations of the graph.
const (
	KindNote = kgknowledge.KindNote
	KindTag  = kgknowledge.KindTag
	KindCode = "code"

	RelReferences = kgknowledge.RelReferences
	RelRelatesTo  = kgknowledge.RelRelatesTo
	RelTagged     = kgknowledge.RelTagged
	RelDependsOn  = kgknowledge.RelDependsOn

	// ProvExtracted marks an edge written in the source (a link, an import).
	ProvExtracted = "extracted"
	// ProvInferred marks an edge a model guessed, with a confidence.
	ProvInferred = "inferred"
	// ProvAmbiguous marks an edge a model could not settle.
	ProvAmbiguous = "ambiguous"

	graphFile    = "graph.json"
	cacheFile    = "cache.json"
	manifestFile = "manifest.json"
)

// ManifestEntry is the filesystem state used for a cheap vault freshness check.
type ManifestEntry struct {
	Size    int64 `json:"size"`
	ModTime int64 `json:"modTime"`
}

// Manifest records every markdown note that contributes to the graph.
type Manifest map[string]ManifestEntry

// CurrentManifest scans note metadata without parsing note contents.
func CurrentManifest(v Vault) (Manifest, error) {
	manifest := Manifest{}
	err := filepath.WalkDir(v.Root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.HasPrefix(d.Name(), ".") && p != v.Root {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(p), mdExt) {
			return nil
		}
		info, ok := indexableNote(d)
		if !ok {
			return nil
		}
		rel, relErr := filepath.Rel(v.Root, p)
		if relErr != nil {
			return relErr
		}
		if filepath.ToSlash(rel) == reportFile && ours(p, reportMarker) {
			return nil
		}
		manifest[filepath.ToSlash(rel)] = ManifestEntry{
			Size: info.Size(), ModTime: info.ModTime().UnixNano(),
		}
		return nil
	})
	return manifest, err
}

// indexableNote reports file metadata for a markdown note small enough to index.
// Unreadable and oversized notes are omitted so one bad file does not fail the scan.
func indexableNote(d os.DirEntry) (os.FileInfo, bool) {
	info, err := d.Info()
	if err != nil || info.Size() > policy().MaxNoteSize {
		return nil, false
	}
	return info, true
}

// NeedsBuild reports whether notes were added, changed, renamed, or deleted
// since the last successful graph build.
func NeedsBuild(v Vault, dir string) (bool, error) {
	current, err := CurrentManifest(v)
	if err != nil {
		return false, err
	}
	stored := Manifest{}
	// A missing or unreadable manifest means the graph must be rebuilt.
	if readErr := kgstore.ReadJSON(filepath.Join(dir, manifestFile), &stored); readErr != nil {
		return true, nil //nolint:nilerr // an unreadable manifest is a reason to rebuild, not an error
	}
	return !reflect.DeepEqual(current, stored), nil
}

// graphMu serializes builds. Session startup, workflow recall, vault search,
// and skill linking can all refresh the same graph.
//
//nolint:gochecknoglobals // one build at a time for the process
var graphMu sync.Mutex

// Ensure brings the structural graph current for vault. It does not run
// model extraction. A nil context leaves the graph untouched.
func Ensure(ctx context.Context, vault string) error {
	if ctx == nil || strings.TrimSpace(vault) == "" {
		return nil
	}
	_, _, err := CheckUpdate(ctx, Vault{Root: vault}, GraphDir(vault), time.Now())
	return err
}

// CheckUpdate incrementally rebuilds a stale graph and reports whether it changed.
func CheckUpdate(
	ctx context.Context,
	v Vault,
	dir string,
	now time.Time,
) (bool, BuildStats, error) {
	graphMu.Lock()
	defer graphMu.Unlock()
	stale, err := NeedsBuild(v, dir)
	if err != nil || !stale {
		return false, BuildStats{}, err
	}
	_, stats, err := buildGraph(ctx, v, dir, now)
	return err == nil, stats, err
}

// Node is one thing the graph knows: a note, a tag, a code package.
type Node = kgknowledge.Node

// Edge is one relation between two nodes.
type Edge = kgknowledge.Edge

// Graph is the vault's knowledge graph: Kartographer's nodes and edges, with
// the vault's own build record.
type Graph struct {
	kgknowledge.Graph
	Vault        string    `json:"vault"`
	Built        time.Time `json:"built"`
	ExtractModel string    `json:"extract_model,omitempty"`
	ExtractedAt  time.Time `json:"extracted_at,omitempty"`
}

// BuildStats reports what a build read.
type BuildStats struct {
	Notes, Changed, CodeFiles, Nodes, Edges, Communities int
}

// cacheEntry is one parsed file, reused while its hash is unchanged.
type cacheEntry struct {
	Hash       string   `json:"hash"`
	Note       *Note    `json:"note,omitempty"`
	Imports    []string `json:"imports,omitempty"`    // code files
	ImportLine []int    `json:"importLine,omitempty"` // 1-based line of each import
}

type buildCache struct {
	Version int                   `json:"version"`
	Files   map[string]cacheEntry `json:"files"`
}

// parserVersion changes whenever parsing changes, so older cached parses
// are re-read. 3 records the source line of each link and tag.
const parserVersion = 3

// builder accumulates a graph's nodes and edges, deduplicated.
type builder struct {
	kg      *kgknowledge.Builder
	missing map[string]bool
	cache   buildCache
	next    *buildCache
	st      *BuildStats
}

// Build parses the vault (and the Go code behind cards' repo links),
// reusing cached parses of unchanged files, and writes graph.json to dir.
// When dir is GraphDir, a graph left in <vault>/.index is read once and the
// owned files there, plus a generated report at the vault root, are removed.
// Callers that only need a current graph use CheckUpdate.
func Build(ctx context.Context, v Vault, dir string, now time.Time) (*Graph, BuildStats, error) {
	graphMu.Lock()
	defer graphMu.Unlock()
	return buildGraph(ctx, v, dir, now)
}

func buildGraph(ctx context.Context, v Vault, dir string, now time.Time) (*Graph, BuildStats, error) {
	var st BuildStats
	next := buildCache{Version: parserVersion, Files: map[string]cacheEntry{}}
	b := &builder{
		kg: kgknowledge.NewBuilder(), missing: map[string]bool{},
		cache: loadCache(dir), next: &next, st: &st,
	}
	notes, err := b.readNotes(v)
	if err != nil {
		return nil, st, err
	}
	resolve := kgknowledge.NewResolver(noteDocs(notes))
	for _, n := range notes {
		for _, m := range b.kg.AddDoc(noteDoc(n), resolve) {
			b.missing[m] = true
		}
	}
	b.addExtracted(v.Root, dir, notes, resolve)
	b.addCode(ctx, v)
	g := &Graph{Graph: *b.kg.Graph(), Vault: v.Root, Built: now.UTC()}
	for m := range b.missing {
		g.Missing = append(g.Missing, m)
	}
	sortGraph(g)
	applyExtractMeta(g, dir)
	g.Modularity = Communities(g)
	st.Nodes, st.Edges, st.Communities = len(g.Nodes), len(g.Edges), g.Communities
	if innerErr := kgstore.WriteJSON(filepath.Join(dir, cacheFile), next); innerErr != nil {
		return nil, st, innerErr
	}
	if innerErr := kgstore.WriteJSON(filepath.Join(dir, graphFile), g); innerErr != nil {
		return nil, st, innerErr
	}
	manifest, innerErr := CurrentManifest(v)
	if innerErr != nil {
		return nil, st, innerErr
	}
	if innerErr = kgstore.WriteJSON(filepath.Join(dir, manifestFile), manifest); innerErr != nil {
		return nil, st, innerErr
	}
	if innerErr = writeReport(g, g.Modularity, dir); innerErr != nil {
		return nil, st, innerErr
	}
	return g, st, nil
}

// readNotes walks the vault, reusing the cached parse of unchanged notes.
func (b *builder) readNotes(v Vault) ([]*Note, error) {
	notes, err := v.Walk()
	if err != nil {
		return nil, err
	}
	for i, n := range notes {
		b.st.Notes++
		key := "note:" + n.Path
		if e, ok := b.cache.Files[key]; ok && e.Hash == n.Hash && e.Note != nil {
			notes[i] = e.Note
		} else {
			b.st.Changed++
		}
		b.next.Files[key] = cacheEntry{Hash: n.Hash, Note: notes[i]}
	}
	return notes, nil
}

// sortGraph sorts the missing link targets; the builder sorts nodes and edges.
func sortGraph(g *Graph) {
	sort.Strings(g.Missing)
}

// Load reads a built graph from dir.
func Load(dir string) (*Graph, error) {
	var g Graph
	if err := kgstore.ReadJSON(filepath.Join(dir, graphFile), &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// GraphPath is the graph.json file of dir.
func GraphPath(dir string) string { return filepath.Join(dir, graphFile) }

func loadCache(dir string) buildCache {
	c := buildCache{Files: map[string]cacheEntry{}}
	_ = kgstore.ReadJSON(filepath.Join(dir, cacheFile), &c)
	if c.Files == nil || c.Version != parserVersion {
		c.Files = map[string]cacheEntry{}
	}
	return c
}

// applyExtractMeta records the extraction model and the latest success time.
func applyExtractMeta(g *Graph, dir string) {
	cache := loadExtractCache(dir)
	var model string
	var at time.Time
	for _, ex := range cache.Notes {
		if ex.Error != "" || ex.Model == "" {
			continue
		}
		if model == "" || ex.At.After(at) || (ex.At.Equal(at) && ex.Model > model) {
			model = ex.Model
			at = ex.At
		}
	}
	g.ExtractModel = model
	if !at.IsZero() {
		g.ExtractedAt = at.UTC()
	}
}

// noteDoc is the graph's view of one vault note.
func noteDoc(n *Note) kgknowledge.Doc {
	kind := ""
	if n.Type != "" {
		kind = strings.ToLower(n.Type)
	}
	return kgknowledge.Doc{
		Path: n.Path, Title: n.Title, Kind: kind, Summary: n.Summary, Tags: n.Tags, Aliases: n.Aliases,
		Links: n.Links, MetaLinks: n.MetaLinks, MDLinks: n.MDLinks,
		LinkLine: n.LinkLine, MetaLine: n.MetaLine, MDLine: n.MDLine, TagLine: n.TagLine,
	}
}

func noteDocs(notes []*Note) []kgknowledge.Doc {
	docs := make([]kgknowledge.Doc, 0, len(notes))
	for _, n := range notes {
		docs = append(docs, noteDoc(n))
	}
	return docs
}
