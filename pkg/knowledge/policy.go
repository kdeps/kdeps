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
	MaxCodeFiles      int     `yaml:"max_code_files"`     // Go files read per repo; the rest are left out.
	MaxCardBody       int     `yaml:"max_card_body"`      // Body paragraphs of a card kept as its summary.
	ChunkChars        int     `yaml:"chunk_chars"`        // Characters of a note sent to the extraction model per request.
	MaxEntities       int     `yaml:"max_entities"`       // Entities kept from one extraction.
	MaxNameLen        int     `yaml:"max_name_len"`       // Longest entity name kept.
	MentionConfidence float64 `yaml:"mention_confidence"` // Confidence of an entity the note states outright.
	AmbiguousBelow    float64 `yaml:"ambiguous_below"`    // An extracted link under this confidence is marked ambiguous.
	DefaultWorkers    int     `yaml:"default_workers"`    // Concurrent extraction requests when none is configured.
	MaxNoteSize       int64   `yaml:"max_note_size"`      // Notes larger than this many bytes are skipped (2 MiB).
	MaxSummary        int     `yaml:"max_summary"`        // Characters of a session note's summary line.
	MaxSlug           int     `yaml:"max_slug"`           // Longest file name slug of a session note.
	TitleWords        int     `yaml:"title_words"`        // Words of a session's first message used as its title.
	FollowUpWords     int     `yaml:"follow_up_words"`    // Words of a topic's follow-up question kept.
	RelatedShare      float64 `yaml:"related_share"`      // Share of a topic's related notes shown.
	ReportTop         int     `yaml:"report_top"`         // Entries of each top list in the graph report.
	CommunityTop      int     `yaml:"community_top"`      // Communities listed in the graph report.
	MissingTop        int     `yaml:"missing_top"`        // Missing links listed in the graph report.
	MarkerWindow      int     `yaml:"marker_window"`      // Bytes read around a report marker.
}

// policyValues loads the policy once; a missing or broken item is reported and
// the counts fall back to 1, so the graph still builds with a visible warning.
//nolint:gochecknoglobals // loaded once from the knowledge asset set
var policyValues = sync.OnceValue(func() policyFile {
	var p policyFile
	data, err := assets.Open(knowledgeSet).ReadFile(knowledgeSet + "/" + policyItem + ".yaml")
	if err == nil {
		err = yaml.Unmarshal(data, &p)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "knowledge: policy: %v\n", err)
		p = policyFile{MaxCodeFiles: 1, MaxCardBody: 1, ChunkChars: 1, MaxEntities: 1, MaxNameLen: 1, DefaultWorkers: 1, MaxNoteSize: 1, MaxSummary: 1, MaxSlug: 1, TitleWords: 1, FollowUpWords: 1, ReportTop: 1, CommunityTop: 1, MissingTop: 1, MarkerWindow: 1}
	}
	return p
})

// policy returns the knowledge graph's limits and thresholds.
func policy() policyFile { return policyValues() }
