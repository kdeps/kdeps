package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

// A model's refusal record: every time the loop catches a model inventing a
// limit, describing a sandbox session, or claiming an action no tool ran, its
// own words are kept here, per backend/model, across sessions. The record is
// put in front of that model whenever it is used again (the
// model-refusal-record harness entry).
const (
	modelRecordsFile    = "model-records.json"
	refusalKindLimit    = "invented limit"
	refusalKindSandbox  = "sandbox story"
	refusalKindUnbacked = "claimed action, no tool ran"
	refusalTextMax      = 240
)

// modelRefusal is one caught refusal.
type modelRefusal struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
	At   string `json:"at"`
}

// modelRecord is one model's refusals: Count is every catch, Refusals the
// newest distinct ones (bounded by the model-record-limit event).
type modelRecord struct {
	Count    int            `json:"count"`
	Refusals []modelRefusal `json:"refusals"`
}

// sentenceEndRe is the end of a sentence: . ! or ? before whitespace.
var sentenceEndRe = regexp.MustCompile(`[.!?]\s`)

//nolint:gochecknoglobals // serializes the records file
var modelRecordsMu sync.Mutex

// modelRecordsPath is ~/.kdeps/model-records.json ("" without a home).
func modelRecordsPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".kdeps", modelRecordsFile)
}

func readModelRecords() map[string]*modelRecord {
	out := map[string]*modelRecord{}
	p := modelRecordsPath()
	if p == "" {
		return out
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return out
	}
	_ = json.Unmarshal(data, &out)
	return out
}

func writeModelRecords(recs map[string]*modelRecord) {
	p := modelRecordsPath()
	if p == "" {
		return
	}
	data, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		return
	}
	if os.MkdirAll(filepath.Dir(p), 0o750) == nil {
		_ = os.WriteFile(p, data, 0o600)
	}
}

// modelRecordKey names a model's record: backend/model.
func modelRecordKey(backend, model string) string {
	return strings.TrimSpace(backend) + "/" + strings.TrimSpace(model)
}

// refusalSentence is the first sentence of a refusal, cut to refusalTextMax.
func refusalSentence(content string) string {
	t := strings.Join(strings.Fields(sanitizeLoopArtifacts(content)), " ")
	if loc := sentenceEndRe.FindStringIndex(t); loc != nil {
		t = t[:loc[0]+1] // a "." inside a name (notes.md) is not an end
	}
	if len(t) > refusalTextMax {
		t = truncateValue(t, refusalTextMax)
	}
	return t
}

// recordModelRefusal adds a caught refusal to the current model's record.
func (l *Loop) recordModelRefusal(kind, content string) {
	text := refusalSentence(content)
	if text == "" || l.config.Model == "" {
		return
	}
	key := modelRecordKey(l.config.Backend, l.config.Model)
	modelRecordsMu.Lock()
	defer modelRecordsMu.Unlock()
	recs := readModelRecords()
	rec := recs[key]
	if rec == nil {
		rec = &modelRecord{}
		recs[key] = rec
	}
	rec.Count++
	rec.Refusals = slices.DeleteFunc(rec.Refusals, func(r modelRefusal) bool {
		return strings.EqualFold(r.Text, text)
	})
	rec.Refusals = append(rec.Refusals, modelRefusal{Kind: kind, Text: text, At: time.Now().UTC().Format(time.RFC3339)})
	if n := modelRecordLimit(); len(rec.Refusals) > n {
		rec.Refusals = rec.Refusals[len(rec.Refusals)-n:]
	}
	writeModelRecords(recs)
}

// modelRecordFor returns a model's record, or nil when it has none.
func modelRecordFor(backend, model string) *modelRecord {
	modelRecordsMu.Lock()
	defer modelRecordsMu.Unlock()
	rec := readModelRecords()[modelRecordKey(backend, model)]
	if rec == nil || len(rec.Refusals) == 0 {
		return nil
	}
	return rec
}

// clearModelRecord deletes a model's record and reports whether it had one.
func clearModelRecord(backend, model string) bool {
	modelRecordsMu.Lock()
	defer modelRecordsMu.Unlock()
	recs := readModelRecords()
	key := modelRecordKey(backend, model)
	if _, ok := recs[key]; !ok {
		return false
	}
	delete(recs, key)
	writeModelRecords(recs)
	return true
}

// modelRecordPreamble is the model-refusal-record harness entry rendered with
// the current model's record, or "" when the model has none.
func (l *Loop) modelRecordPreamble() string {
	rec := modelRecordFor(l.config.Backend, l.config.Model)
	if rec == nil {
		return ""
	}
	return harnessRender("model-refusal-record", struct {
		Model    string
		Count    int
		Refusals []modelRefusal
	}{modelRecordKey(l.config.Backend, l.config.Model), rec.Count, rec.Refusals})
}
