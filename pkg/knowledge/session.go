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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	maxSummary    = 120
	maxSlug       = 60
	noneText      = "(none)"
	titleWords    = 8
	boldTitleMark = "**"
)

// SessionSections are a session note's headings, in order.
//
//nolint:gochecknoglobals // fixed list
var SessionSections = []string{"Focus", "What was built", "Mistakes", "Decisions", "Wins"}

var (
	unsafeNameRe = regexp.MustCompile(`[^\p{L}\p{N} ._-]+`)
	// listMarkerRe matches a list item's marker and checkbox: "- ", "2. ", "- [x] ".
	listMarkerRe = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])?\s*(?:\[[ xX]\])?\s*`)
)

// SessionNote is what WriteSession wrote.
type SessionNote struct {
	File string // absolute path of the session note
	Card *Card  // the project's card
	New  bool
}

// WriteSession writes (or rewrites) the note for session id from a summary
// into the project's sessions/ folder, linked to the project's card. The
// summary's first next step becomes the card's next (its only home).
func WriteSession(card *Card, id, summary string, now time.Time) (*SessionNote, error) {
	sec := SummarySections(summary)
	focus := firstNonEmpty(sec["goal"], sec["focus"])
	if strings.TrimSpace(focus) == "" {
		return nil, errors.New("summary has no goal")
	}
	dir := filepath.Join(card.Dir(), SessionsFolder)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, err
	}
	line := oneLine(focus)
	sn := &SessionNote{Card: card}
	sn.File = findSession(dir, id)
	if sn.File == "" {
		sn.New = true
		sn.File = filepath.Join(dir, now.Format("2006-01-02 1504")+" "+slug(line)+mdExt)
	}
	if err := os.WriteFile(sn.File, []byte(sessionBody(id, card.Name, line, sec, now)), filePerm); err != nil {
		return nil, err
	}
	if next := firstItem(sec["next steps"]); next != "" {
		if err := card.Update("", next, now); err != nil {
			return nil, err
		}
	}
	return sn, nil
}

// WriteDecisions writes one note per item of a summary's Key Decisions into
// the project's decisions/ folder, linked to the session note and the card.
// A decision whose title already has a note is left alone (one home per
// fact). It returns the notes it wrote.
func WriteDecisions(card *Card, sessionFile, summary string, now time.Time) ([]string, error) {
	items := listItems(SummarySections(summary)["key decisions"])
	if len(items) == 0 {
		return nil, nil
	}
	dir := filepath.Join(card.Dir(), DecisionsFolder)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, err
	}
	session := strings.TrimSuffix(filepath.Base(sessionFile), mdExt)
	var written []string
	for _, item := range items {
		title := decisionTitle(item)
		file := filepath.Join(dir, slug(title)+mdExt)
		if _, err := os.Stat(file); err == nil {
			continue
		}
		summaryLine := strings.ReplaceAll(item, boldTitleMark, "")
		body := fmt.Sprintf("%s\ntype: decision\nstatus: done\nproject: %s\ncreated: %s\nsummary: %s\n"+
			"links: [%q, %q]\n%s\n\n# %s\n\n%s\n",
			frontDelim, yamlScalar(card.Name), now.Format(time.DateOnly), yamlScalar(oneLine(summaryLine)),
			"[["+session+"]]", "[["+card.Name+"]]", frontDelim, title, summaryLine)
		if err := os.WriteFile(file, []byte(body), filePerm); err != nil {
			return written, err
		}
		written = append(written, file)
	}
	return written, nil
}

// listItems is text's list items without their markers, skipping "(none)".
func listItems(text string) []string {
	var out []string
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(listMarkerRe.ReplaceAllString(l, ""))
		if l != "" && l != noneText {
			out = append(out, l)
		}
	}
	return out
}

// decisionTitle is a decision's bold lead ("**Monthly**: ...") or its
// first few words.
func decisionTitle(item string) string {
	if rest, ok := strings.CutPrefix(item, boldTitleMark); ok {
		if t, _, found := strings.Cut(rest, boldTitleMark); found && strings.TrimSpace(t) != "" {
			return slug(t)
		}
	}
	words := strings.Fields(strings.ReplaceAll(item, boldTitleMark, ""))
	if len(words) > titleWords {
		words = words[:titleWords]
	}
	return slug(strings.TrimRight(strings.Join(words, " "), ":;,."))
}

func sessionBody(id, project, summary string, sec map[string]string, now time.Time) string {
	var b strings.Builder
	b.WriteString(frontDelim + "\ntype: session\nstatus: done\n")
	fmt.Fprintf(&b, "project: %s\n", yamlScalar(project))
	fmt.Fprintf(
		&b,
		"session: %s\ncreated: %s\nsummary: %s\ntags: [session",
		yamlScalar(id),
		now.Format(time.DateOnly),
		yamlScalar(summary),
	)
	fmt.Fprintf(&b, ", %s]\nlinks: [%q]\n", yamlScalar(project), "[["+project+"]]")
	b.WriteString(frontDelim + "\n")
	body := map[string]string{
		"Focus":          sec["goal"],
		"What was built": firstNonEmpty(sec["done"], sec["progress"]),
		"Mistakes":       sec["mistakes"],
		"Decisions":      sec["key decisions"],
		"Wins":           sec["wins"],
	}
	for _, h := range SessionSections {
		text := strings.TrimSpace(body[h])
		if text == "" {
			text = noneText
		}
		fmt.Fprintf(&b, "\n## %s\n\n%s\n", h, text)
	}
	return b.String()
}

// SummarySections splits a compaction summary into its "##" and "###"
// sections, keyed by lowercase heading.
func SummarySections(summary string) map[string]string {
	out := map[string]string{}
	var cur []string
	key := ""
	flush := func() {
		if key != "" {
			out[key] = strings.TrimSpace(strings.Join(cur, "\n"))
		}
	}
	for _, line := range strings.Split(summary, "\n") {
		t := strings.TrimSpace(line)
		if h, ok := strings.CutPrefix(t, "### "); ok {
			flush()
			cur, key = nil, strings.ToLower(strings.TrimSpace(h))
			continue
		}
		if h, ok := strings.CutPrefix(t, "## "); ok {
			flush()
			cur, key = nil, strings.ToLower(strings.TrimSpace(h))
			continue
		}
		cur = append(cur, line)
	}
	flush()
	// "## Progress" holds its "###" parts too.
	if _, ok := out["progress"]; !ok || out["progress"] == "" {
		var parts []string
		for _, k := range []string{"done", "in progress"} {
			if s := strings.TrimSpace(out[k]); s != "" && s != noneText {
				parts = append(parts, s)
			}
		}
		out["progress"] = strings.Join(parts, "\n")
	}
	for k, v := range out {
		if v == noneText {
			out[k] = ""
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// oneLine is the first meaningful line of text, without list markers,
// capped for a summary.
func oneLine(text string) string {
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(listMarkerRe.ReplaceAllString(l, ""))
		l = strings.Trim(l, "*_ ")
		if l != "" {
			if r := []rune(l); len(r) > maxSummary {
				l = string(r[:maxSummary-3]) + "..."
			}
			return l
		}
	}
	return ""
}

// firstItem is the first list item of text, or "".
func firstItem(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return oneLine(text)
}

func slug(s string) string {
	s = strings.TrimSpace(unsafeNameRe.ReplaceAllString(s, ""))
	if r := []rune(s); len(r) > maxSlug {
		s = strings.TrimSpace(string(r[:maxSlug]))
	}
	if s == "" {
		s = "session"
	}
	return s
}

// findSession is the note of session id in dir, or "".
func findSession(dir, id string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), mdExt) {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(dir, e.Name()))
		if readErr != nil {
			continue
		}
		if meta, _ := SplitFrontmatter(data); metaString(meta, "session") == id {
			return filepath.Join(dir, e.Name())
		}
	}
	return ""
}

// runLog serializes AppendRun: a workflow's resources may run in parallel.
//
//nolint:gochecknoglobals // process-wide lock for one file per day
var runLog sync.Mutex

// AppendRun adds one workflow chat call to the project's run note for the
// day, sessions/<date> <project> runs.md, creating it on first use.
func AppendRun(card *Card, prompt, reply string, now time.Time) (string, error) {
	runLog.Lock()
	defer runLog.Unlock()
	dir := filepath.Join(card.Dir(), SessionsFolder)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return "", err
	}
	file := filepath.Join(dir, now.Format(time.DateOnly)+" "+card.Name+" runs"+mdExt)
	entry := fmt.Sprintf("- %s %s -> %s\n", now.Format("15:04:05"), oneLine(prompt), oneLine(reply))
	if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
		head := fmt.Sprintf("%s\ntype: session\nstatus: done\nproject: %s\nsession: %s\ncreated: %s\nsummary: %s\n"+
			"tags: [session, workflow, %s]\nlinks: [%q]\n%s\n\n## Runs\n\n",
			frontDelim, yamlScalar(card.Name), yamlScalar("runs-"+now.Format(time.DateOnly)),
			now.Format(time.DateOnly), yamlScalar(card.Name+" workflow runs on "+now.Format(time.DateOnly)),
			yamlScalar(card.Name), "[["+card.Name+"]]", frontDelim)
		return file, os.WriteFile(file, []byte(head+entry), filePerm)
	}
	f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, filePerm)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.WriteString(entry)
	return file, err
}
