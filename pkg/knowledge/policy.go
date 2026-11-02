// The policy values live in the knowledge asset set (policy.yaml). This file
// declares their shape and the accessor.

package knowledge

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/assets"
)

// policyItem is the policy item of the knowledge asset set.
const policyItem = "policy"

// policyFile is the policy: the limits and thresholds the knowledge graph uses.
type policyFile struct {
	MaxCodeFiles      int     `yaml:"max_code_files"`      // Go files read per repo; the rest are left out.
	MaxCardBody       int     `yaml:"max_card_body"`       // Body paragraphs of a card kept as its summary.
	ChunkChars        int     `yaml:"chunk_chars"`         // Characters of a note sent to the extraction model per request.
	MaxEntities       int     `yaml:"max_entities"`        // Entities kept from one extraction.
	MaxNameLen        int     `yaml:"max_name_len"`        // Longest entity name kept.
	MentionConfidence float64 `yaml:"mention_confidence"`  // Confidence of an entity the note states outright.
	AmbiguousBelow    float64 `yaml:"ambiguous_below"`     // An extracted link under this confidence is marked ambiguous.
	DefaultWorkers    int     `yaml:"default_workers"`     // Concurrent extraction requests when none is configured.
	MaxNoteSize       int64   `yaml:"max_note_size"`       // Notes larger than this many bytes are skipped (2 MiB).
	MaxSummary        int     `yaml:"max_summary"`         // Characters of a session note's summary line.
	MaxSlug           int     `yaml:"max_slug"`            // Longest file name slug of a session note.
	TitleWords        int     `yaml:"title_words"`         // Words of a session's first message used as its title.
	FollowUpWords     int     `yaml:"follow_up_words"`     // Words of a topic's follow-up question kept.
	RelatedShare      float64 `yaml:"related_share"`       // Share of a topic's related notes shown.
	ReportTop         int     `yaml:"report_top"`          // Entries of each top list in the graph report.
	CommunityTop      int     `yaml:"community_top"`       // Communities listed in the graph report.
	MissingTop        int     `yaml:"missing_top"`         // Missing links listed in the graph report.
	MarkerWindow      int     `yaml:"marker_window"`       // Bytes read around a report marker.
	TagEdgeWeight     float64 `yaml:"tag_edge_weight"`     // weight of a tag edge in recall, against 1 for a link
	AmbiguousWeight   float64 `yaml:"ambiguous_weight"`    // weight of an ambiguous edge in recall, times its confidence
	RecallMaxHops     int     `yaml:"recall_max_hops"`     // hops a recall walk takes from a matched note; a larger request is clamped to it
	RecallBudget      int     `yaml:"recall_budget"`       // tokens of recalled paths and summaries when a request gives none
	RecallMaxSeeds    int     `yaml:"recall_max_seeds"`    // best-matching notes a recall starts from
	RecallMaxHits     int     `yaml:"recall_max_hits"`     // notes a recall returns at most
	RecallHopDecay    float64 `yaml:"recall_hop_decay"`    // score kept per hop, before the edge weight and fan-out
	LabelMatchBoost   float64 `yaml:"label_match_boost"`   // multiplier of a query word that is in a note's label
	FullLabelBoost    float64 `yaml:"full_label_boost"`    // multiplier when a note's whole label appears in the query
	SessionNoteWeight float64 `yaml:"session_note_weight"` // weight of a session note in recall
	MemoryNoteWeight  float64 `yaml:"memory_note_weight"`  // weight of a project memory note in recall
	BytesPerToken     int     `yaml:"bytes_per_token"`     // bytes per token when a recalled note's cost is estimated
}

// policyValues loads the policy once; a missing or broken item is reported and
// the counts fall back to 1, so the graph still builds with a visible warning.
//
// policyCache holds the policy once it has loaded. A failed load is reported
// and not cached, so a later call with the assets in place reads them.
//
//nolint:gochecknoglobals // loaded once from the knowledge asset set
//nolint:gochecknoglobals // loaded once from the knowledge asset set
var (
	policyMu    sync.Mutex
	policyCache *policyFile
)

// loadPolicy reads the policy item of the knowledge asset set.
func loadPolicy() (policyFile, error) {
	data, err := assets.Open(knowledgeSet).ReadFile(knowledgeSet + "/" + policyItem + ".yaml")
	if err != nil {
		return policyFile{}, err
	}
	return parsePolicy(data)
}

// parsePolicy reads policy.yaml. The recall counts must be set: a file from
// before they existed is rejected, so the warning fallback applies instead of
// a recall that divides by zero.
func parsePolicy(data []byte) (policyFile, error) {
	var p policyFile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return p, err
	}
	for _, f := range []struct {
		key   string
		value int
	}{
		{"recall_max_hops", p.RecallMaxHops}, {"recall_budget", p.RecallBudget},
		{"recall_max_seeds", p.RecallMaxSeeds}, {"recall_max_hits", p.RecallMaxHits},
		{"bytes_per_token", p.BytesPerToken},
	} {
		if f.value <= 0 {
			return p, fmt.Errorf("%s must be above 0", f.key)
		}
	}
	return p, nil
}

// policy returns the knowledge graph's limits and thresholds.
func policy() policyFile {
	policyMu.Lock()
	defer policyMu.Unlock()
	if policyCache != nil {
		return *policyCache
	}
	p, err := loadPolicy()
	if err != nil {
		fmt.Fprintf(os.Stderr, "knowledge: policy: %v\n", err)
		return policyFile{
			MaxCodeFiles:   1,
			MaxCardBody:    1,
			ChunkChars:     1,
			MaxEntities:    1,
			MaxNameLen:     1,
			DefaultWorkers: 1,
			MaxNoteSize:    1,
			MaxSummary:     1,
			MaxSlug:        1,
			TitleWords:     1,
			FollowUpWords:  1,
			ReportTop:      1,
			CommunityTop:   1,
			MissingTop:     1,
			MarkerWindow:   1,
			RecallMaxHops:  1,
			RecallBudget:   1,
			RecallMaxSeeds: 1,
			RecallMaxHits:  1,
			BytesPerToken:  1,
		}
	}
	policyCache = &p
	return p
}
