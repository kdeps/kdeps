package agent

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/assets"
)

// The memory store's rules are the "rules" item of the "memory" asset set.
const (
	memoryRulesSet  = "memory"
	memoryRulesItem = "rules"
)

// MemoryRules is how the memory store files what it saves: an entry's type
// from its key, the type it links to, which tool results it captures, the
// types a resumed session picks up from, the low-signal types pruned first,
// and what reads as a finished status
// or a resolved error.
type MemoryRules struct {
	Types         []memoryTypeRule  `yaml:"types"`
	DefaultType   string            `yaml:"default_type"`
	Parents       map[string]string `yaml:"parents"`
	CaptureTools  []string          `yaml:"capture_tools"`
	Resume        []string          `yaml:"resume"`
	LowSignal     []string          `yaml:"low_signal"`
	StatusDone    memoryMarkers     `yaml:"status_done"`
	ErrorResolved memoryMarkers     `yaml:"error_resolved"`
	ReplyCapture  struct {
		ActionVerbs []string `yaml:"action_verbs"`
		ActionMin   int      `yaml:"action_min"`
		ActionMax   int      `yaml:"action_max"`
		FileVerbs   []string `yaml:"file_verbs"`
	} `yaml:"reply_capture"`
	FocusStopwords []string `yaml:"focus_stopwords"`

	actionRe, fileRe *regexp.Regexp // built from ReplyCapture; nil captures nothing
	focusStop        map[string]bool
}

// verbAlternation is a regexp alternation of verbs, each matched literally.
func verbAlternation(verbs []string) string {
	quoted := make([]string, len(verbs))
	for i, v := range verbs {
		quoted[i] = regexp.QuoteMeta(v)
	}
	return strings.Join(quoted, "|")
}

// memoryTypeRule gives Type to a key that has Prefix, equals Equals, or
// contains one of Contains.
type memoryTypeRule struct {
	Prefix   string   `yaml:"prefix"`
	Equals   string   `yaml:"equals"`
	Contains []string `yaml:"contains"`
	Type     string   `yaml:"type"`
}

// memoryMarkers reads a value as a hit when it has one of Markers and none
// of Not.
type memoryMarkers struct {
	Not     []string `yaml:"not"`
	Markers []string `yaml:"markers"`
}

func (r memoryTypeRule) match(key string) bool {
	switch {
	case r.Prefix != "":
		return strings.HasPrefix(key, r.Prefix)
	case r.Equals != "":
		return key == r.Equals
	default:
		return containsAny(key, r.Contains...)
	}
}

func (m memoryMarkers) hit(value string) bool {
	v := strings.ToLower(value)
	toks := wordTokens(v)
	for _, neg := range m.Not {
		if markerHit(v, toks, neg) {
			return false
		}
	}
	for _, mk := range m.Markers {
		if markerHit(v, toks, mk) {
			return true
		}
	}
	return false
}

func parseMemoryRules(data []byte) (MemoryRules, error) {
	var r MemoryRules
	if err := yaml.Unmarshal(data, &r); err != nil {
		return MemoryRules{}, err
	}
	for i, t := range r.Types {
		if t.Type == "" || (t.Prefix == "" && t.Equals == "" && len(t.Contains) == 0) {
			return MemoryRules{}, fmt.Errorf("types[%d]: needs a type and a prefix, equals or contains", i)
		}
	}
	if r.DefaultType == "" {
		return MemoryRules{}, errors.New("needs a default_type")
	}
	if c := r.ReplyCapture; len(c.ActionVerbs) > 0 {
		if c.ActionMin < 0 || c.ActionMax < c.ActionMin {
			return MemoryRules{}, errors.New("reply_capture: action_min must be 0 or more and at most action_max")
		}
		r.actionRe = regexp.MustCompile(`(?i)(` + verbAlternation(c.ActionVerbs) + `)\s+(.+)`)
	}
	r.focusStop = make(map[string]bool, len(r.FocusStopwords))
	for _, w := range r.FocusStopwords {
		r.focusStop[strings.ToLower(w)] = true
	}
	if verbs := r.ReplyCapture.FileVerbs; len(verbs) > 0 {
		r.fileRe = regexp.MustCompile(
			`(?:` + verbAlternation(verbs) + `)\s+(?:file\s+)?` + "`" + `?([~/.][^\s` + "`" + `]+)` + "`" + `?`,
		)
	}
	return r, nil
}

//nolint:gochecknoglobals // loaded from the memory asset set; reloaded by ReloadAssets
var memoryRules MemoryRules

// initMemoryRules loads the memory rules. A missing or broken file leaves
// every entry a plain note with nothing auto-captured.
//
//kdeps:flow memory: the agent saves a memory or a tool returns > kdeps types the entry by its key and links it to its parent type > [the tool is in capture_tools] > its result is captured > the memory block flags open errors and resumes from unfinished work
func initMemoryRules() {
	data, err := assets.Open(memoryRulesSet).ReadFile(memoryRulesSet + "/" + memoryRulesItem + ".yaml")
	if err == nil {
		memoryRules, err = parseMemoryRules(data)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "memory: rules: %v\n", err)
		memoryRules = MemoryRules{DefaultType: memTypeNote}
	}
}

//nolint:gochecknoinits // registers and loads the memory rules asset set
func init() {
	assets.Register(memoryRulesSet, assets.Spec{
		Validate: func(data []byte, _ string) error {
			_, err := parseMemoryRules(data)
			return err
		},
		Required: []string{memoryRulesItem},
	})
	initMemoryRules()
}
