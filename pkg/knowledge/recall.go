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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	kgknowledge "github.com/kdeps/kartographer/knowledge"
	kword "github.com/kdeps/kartographer/text"
	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v3/pkg/assets"
)

// Recall results and the index come from the kartographer knowledge package;
// kdeps supplies the policy (word rules, tuning, source weights) and the prompt format.

// PathStep is one edge on the path that reached a recalled note.
type PathStep = kgknowledge.PathStep

// Hit is one recalled note.
type Hit = kgknowledge.Hit

// Recall is the outcome of one recall: the notes worth reading for a request.
type Recall = kgknowledge.Recall

// Index answers recalls over a loaded graph.
type Index = kgknowledge.Index

// Recall rules are the "recall" item of the "knowledge" asset set: the word
// lengths a query word must reach, and the words it ignores.
const (
	knowledgeSet = "knowledge"
	recallItem   = "recall"
)

// recallRulesFile mirrors recall.yaml; stop and task are its word lists as sets.
type recallRulesFile struct {
	MinWordLength int      `yaml:"min_word_length"`
	MinStemLength int      `yaml:"min_stem_length"`
	Stopwords     []string `yaml:"stopwords"`
	TaskWords     []string `yaml:"task_words"`
	stop, task    map[string]bool
}

//nolint:gochecknoglobals // loaded once from the knowledge asset set
var (
	recallRulesItem     = &yamlItem[recallRulesFile]{name: recallItem, parse: parseRecallRules}
	recallRulesReported sync.Once
)

// recallRules reads recall.yaml on first use (installing it when a profile
// left it out). A missing or broken file turns word filtering off (every word
// of length one or more is kept) and says so once.
//
//kdeps:flow knowledge: a knowledge-graph recall query arrives > its words are split and stemmed > [the recall rules set the word lengths and stopwords] > matching nodes are scored > the best notes fill the budget
func recallRules() recallRulesFile {
	r, err := recallRulesItem.get()
	if err != nil {
		recallRulesReported.Do(func() { fmt.Fprintf(os.Stderr, "knowledge: recall rules: %v\n", err) })
		return recallRulesFile{MinWordLength: 1, MinStemLength: 1}
	}
	return r
}

func parseRecallRules(data []byte) (recallRulesFile, error) {
	var r recallRulesFile
	if err := yaml.Unmarshal(data, &r); err != nil {
		return r, err
	}
	r.stop, r.task = kword.Set(r.Stopwords), kword.Set(r.TaskWords)
	return r, nil
}

//nolint:gochecknoinits // registers the knowledge asset set
func init() {
	assets.Register(knowledgeSet, knowledgeSpec())
}

// knowledgeSpec is the knowledge asset set: each item is checked by its own
// parser before install replaces a working file.
func knowledgeSpec() assets.Spec {
	return assets.Spec{
		Validate: func(data []byte, name string) error {
			switch name {
			case policyItem:
				_, err := parsePolicy(data)
				return err
			case vaultItem:
				_, err := parseLayout(data)
				return err
			case checkItem:
				_, err := parseCheckRules(data)
				return err
			case notesItem:
				_, err := parseNoteRules(data)
				return err
			case extractItem:
				_, err := parseExtractRules(data)
				return err
			case searchItem:
				_, err := parseSearchRules(data)
				return err
			case promptsItem:
				_, err := parsePrompts(data)
				return err
			}
			_, err := parseRecallRules(data)
			return err
		},
		Required: []string{
			recallItem,
			policyItem,
			vaultItem,
			checkItem,
			notesItem,
			extractItem,
			searchItem,
			promptsItem,
		},
	}
}

// wordOptions is the recall rules' policy for keyword words.
func wordOptions() kword.Options {
	r := recallRules()
	return kword.Options{MinWordLength: r.MinWordLength, MinStemLength: r.MinStemLength, Stopwords: r.stop}
}

func words(s string) []string { return kword.Words(s, wordOptions()) }

// recallTuning is the recall numbers of the policy.
func recallTuning() kgknowledge.Tuning {
	p := policy()
	return kgknowledge.Tuning{
		MaxHops:         p.RecallMaxHops,
		DefaultBudget:   p.RecallBudget,
		MaxSeeds:        p.RecallMaxSeeds,
		MaxHits:         p.RecallMaxHits,
		BytesPerToken:   p.BytesPerToken,
		HopDecay:        p.RecallHopDecay,
		LabelMatchBoost: p.LabelMatchBoost,
		FullLabelBoost:  p.FullLabelBoost,
		TagEdgeWeight:   p.TagEdgeWeight,
		AmbiguousWeight: p.AmbiguousWeight,
	}
}

// DefaultRecall is the hops and token budget a recall uses when a request gives none.
func DefaultRecall() (int, int) {
	p := policy()
	return p.RecallMaxHops, p.RecallBudget
}

// NewIndex prepares g for recall.
func NewIndex(g *Graph) *Index {
	return kgknowledge.NewIndex(&g.Graph, kgknowledge.Options{
		Words:      wordOptions(),
		Tuning:     recallTuning(),
		NodeWeight: sourceWeight,
	})
}

// sourceWeight ranks notes kdeps writes itself below the ones people curate:
// a session note is titled with its prompts and a memory entry is short, so
// both would otherwise out-match a decision or gotcha on the same words.
func sourceWeight(n *Node) float64 {
	p := policy()
	switch {
	case isSessionKind(n.Kind):
		return p.SessionNoteWeight
	case isMemoryNote(n.Note):
		return p.MemoryNoteWeight
	}
	return 1
}

// isSessionKind reports a node of the kind notes.yaml gives session notes.
func isSessionKind(kind string) bool {
	r, err := notesRules()
	return err == nil && kind == r.Session.Kind
}

// edgeWeight is the strength of an edge in recall, for community detection and
// link ranking.
func edgeWeight(e *Edge) float64 { return kgknowledge.EdgeWeight(e, recallTuning()) }

// FormatRecall renders a recall for the model: absolute paths it can read, each
// with its summary and how it was reached. Empty when nothing was recalled.
func FormatRecall(r Recall, vault string) string {
	if len(r.Hits) == 0 {
		return ""
	}
	type hit struct{ Path, Summary, Chain, Via string }
	hits := make([]hit, 0, len(r.Hits))
	for _, h := range r.Hits {
		hits = append(hits, hit{filepath.Join(vault, filepath.FromSlash(h.Note)), h.Summary, h.Chain(), h.Via})
	}
	return Prompt(PromptRecall, map[string]any{"Hits": hits})
}

//nolint:gochecknoglobals // process-wide cache of loaded graphs
var shared = struct {
	sync.Mutex
	m map[string]sharedIndex
}{m: map[string]sharedIndex{}}

type sharedIndex struct {
	ix  *Index
	mod time.Time
}

// SharedIndex returns the recall index of the graph built in dir, loaded
// once and reloaded when graph.json changes; nil when there is no graph.
func SharedIndex(dir string) *Index {
	info, err := os.Stat(GraphPath(dir))
	if err != nil {
		return nil
	}
	shared.Lock()
	defer shared.Unlock()
	if c, ok := shared.m[dir]; ok && !info.ModTime().After(c.mod) {
		return c.ix
	}
	g, err := Load(dir)
	if err != nil {
		return shared.m[dir].ix
	}
	ix := NewIndex(g)
	shared.m[dir] = sharedIndex{ix: ix, mod: info.ModTime()}
	return ix
}
