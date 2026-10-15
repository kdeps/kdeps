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
	"sort"
	"strings"
	"time"
)

// Card fields every card carries in its frontmatter.
const (
	CardType    = "type"
	CardStatus  = "status"
	CardNext    = "next"
	CardUpdated = "updated"
)

// CardFields lists the required card fields in order.
//
//nolint:gochecknoglobals // fixed list
var CardFields = []string{CardType, CardStatus, CardNext, CardUpdated}

// CardDirs are the records folders that hold cards, one card per file.
//
//nolint:gochecknoglobals // fixed list
var CardDirs = []string{"records/projects", "records/clients"}

// linkCells is how many cells a links-table row needs: source and path.
const linkCells = 2

// RoutingFile is the workspace's routing table and rules.
const RoutingFile = "KDEPS.md"

// Link is one row of a card's links table: a source and where it lives.
type Link struct {
	Source string `json:"source"`
	Path   string `json:"path"`
}

// Card is a short routing record for a project or client: status and next
// step in frontmatter, then a table of where everything lives. It points;
// it never holds the story.
type Card struct {
	Name    string `json:"name"`
	File    string `json:"file"` // absolute path
	Type    string `json:"type"`
	Status  string `json:"status"`
	Next    string `json:"next"`
	Updated string `json:"updated"`
	Links   []Link `json:"links"`
	// BodyLines counts body lines outside the links table (a card that tells
	// a story fails the check).
	BodyLines int `json:"-"`
}

// Workspace is the folder holding KDEPS.md, the records and the schema.
type Workspace struct {
	Root string
}

// ErrNoCard is returned for a card name that has no file.
var ErrNoCard = errors.New("no such card")

// Cards returns every card of the workspace, sorted by name.
func (w Workspace) Cards() ([]*Card, error) {
	var cards []*Card
	for _, dir := range CardDirs {
		entries, err := os.ReadDir(filepath.Join(w.Root, filepath.FromSlash(dir)))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !strings.EqualFold(filepath.Ext(e.Name()), mdExt) {
				continue
			}
			c, readErr := ReadCard(filepath.Join(w.Root, filepath.FromSlash(dir), e.Name()))
			if readErr != nil {
				return nil, readErr
			}
			cards = append(cards, c)
		}
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].Name < cards[j].Name })
	return cards, nil
}

// Card returns the named card (case-insensitive).
func (w Workspace) Card(name string) (*Card, error) {
	cards, err := w.Cards()
	if err != nil {
		return nil, err
	}
	for _, c := range cards {
		if strings.EqualFold(c.Name, name) {
			return c, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNoCard, name)
}

// CardForDir returns the card whose repo (or code, path) link contains dir,
// or nil.
func (w Workspace) CardForDir(dir string) *Card {
	cards, err := w.Cards()
	if err != nil {
		return nil
	}
	dir = filepath.Clean(dir)
	var best *Card
	bestLen := 0
	for _, c := range cards {
		for _, l := range c.Links {
			switch strings.ToLower(l.Source) {
			case "repo", "code", "path", "folder":
			default:
				continue
			}
			p := ExpandPath(l.Path, w.Root)
			if (dir == p || strings.HasPrefix(dir, p+string(filepath.Separator))) && len(p) > bestLen {
				best, bestLen = c, len(p)
			}
		}
	}
	return best
}

// Link returns the path of the card's first link with source (case-
// insensitive), or "".
func (c *Card) Link(source string) string {
	for _, l := range c.Links {
		if strings.EqualFold(l.Source, source) {
			return l.Path
		}
	}
	return ""
}

// ReadCard parses a card file.
func ReadCard(file string) (*Card, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	meta, body := SplitFrontmatter(data)
	c := &Card{
		Name:    strings.TrimSuffix(filepath.Base(file), filepath.Ext(file)),
		File:    file,
		Type:    metaString(meta, CardType),
		Status:  metaString(meta, CardStatus),
		Next:    metaString(meta, CardNext),
		Updated: metaString(meta, CardUpdated),
	}
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case t == "", strings.HasPrefix(t, "#"):
		case strings.HasPrefix(t, "|"):
			cells := strings.Split(strings.Trim(t, "|"), "|")
			if len(cells) < linkCells {
				continue
			}
			src, p := strings.TrimSpace(cells[0]), strings.TrimSpace(cells[1])
			if strings.Trim(src, "-: ") == "" || strings.EqualFold(src, "source") {
				continue // separator or header row
			}
			c.Links = append(c.Links, Link{Source: src, Path: strings.Trim(p, "`")})
		default:
			c.BodyLines++
		}
	}
	return c, nil
}

// Update sets the card's status and/or next step (empty keeps the current
// value) and stamps updated with today's date. Other frontmatter lines and
// the body are kept as they are.
func (c *Card) Update(status, next string, now time.Time) error {
	data, err := os.ReadFile(c.File)
	if err != nil {
		return err
	}
	set := map[string]string{CardUpdated: now.Format(time.DateOnly)}
	if status != "" {
		set[CardStatus] = status
	}
	if next != "" {
		set[CardNext] = next
	}
	out, err := setFrontmatter(string(data), set, CardFields)
	if err != nil {
		return err
	}
	if innerErr := os.WriteFile(c.File, []byte(out), filePerm); innerErr != nil {
		return innerErr
	}
	updated, err := ReadCard(c.File)
	if err != nil {
		return err
	}
	*c = *updated
	return nil
}

// setFrontmatter replaces (or adds, in order) top-level frontmatter keys,
// creating the block when the file has none. Values are written as YAML
// double-quoted strings when they need it.
func setFrontmatter(text string, set map[string]string, order []string) (string, error) {
	var head []string
	body := text
	if strings.HasPrefix(text, frontDelim+"\n") {
		rest := text[len(frontDelim)+1:]
		end := strings.Index("\n"+rest, "\n"+frontDelim+"\n")
		if end < 0 {
			if !strings.HasSuffix(rest, "\n"+frontDelim) {
				return "", errors.New("unterminated frontmatter")
			}
			end = len(rest) - len(frontDelim)
		}
		if end > 0 {
			head = strings.Split(strings.TrimSuffix(rest[:end], "\n"), "\n")
		}
		body = strings.TrimPrefix(rest[end:], frontDelim)
		body = strings.TrimPrefix(body, "\n")
	}
	done := map[string]bool{}
	for i, line := range head {
		key, _, ok := strings.Cut(line, ":")
		if !ok || strings.HasPrefix(line, " ") {
			continue
		}
		if v, want := set[key]; want {
			head[i] = key + ": " + yamlScalar(v)
			done[key] = true
		}
	}
	for _, key := range order {
		if v, want := set[key]; want && !done[key] {
			head = append(head, key+": "+yamlScalar(v))
		}
	}
	return frontDelim + "\n" + strings.Join(head, "\n") + "\n" + frontDelim + "\n" + body, nil
}

// yamlScalar writes s as a plain scalar when safe, else double-quoted.
func yamlScalar(s string) string {
	if s == "" || strings.ContainsAny(s, ":#{}[],&*!|>'\"%@`\n") || strings.TrimSpace(s) != s {
		return fmt.Sprintf("%q", s)
	}
	return s
}
