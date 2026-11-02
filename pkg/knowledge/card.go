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
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/spf13/fileflow"

	kgnotes "github.com/kdeps/kartographer/notes"
)

// linkCells is how many cells a links-table row needs: source and path.
const linkCells = 2

// RoutingFile is the vault's routing table and rules.
const RoutingFile = "KDEPS.md"

// Link is one row of a card's links table: a source and where it lives.
type Link struct {
	Source string `json:"source"`
	Path   string `json:"path"`
}

// Card is a project's routing card, projects/<name>/<name>.md: its
// frontmatter is the project's status, its links table the project's places.
type Card struct {
	Name    string `json:"name"`
	File    string `json:"file"` // absolute path
	Type    string `json:"type"`
	Status  string `json:"status"`
	Next    string `json:"next"`
	Updated string `json:"updated"`
	Links   []Link `json:"links"`
	// BodyLines counts prose lines outside the links table and headings.
	BodyLines int `json:"-"`
}

// ErrNoCard is returned when a named card does not exist.
var ErrNoCard = errors.New("no such card")

// Dir is the card's project folder.
func (c *Card) Dir() string { return filepath.Dir(c.File) }

// Archived reports whether the project sits in archive/.
func (c *Card) Archived() bool {
	return filepath.Base(filepath.Dir(c.Dir())) == Layout().Folders.Archive
}

// Cards returns every project card, active and archived, sorted by name.
func (v Vault) Cards() ([]*Card, error) {
	var cards []*Card
	for _, top := range []string{Layout().Folders.Projects, Layout().Folders.Archive} {
		entries, err := os.ReadDir(filepath.Join(v.Root, top))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			file := filepath.Join(v.Root, top, e.Name(), e.Name()+mdExt)
			if _, statErr := os.Stat(file); statErr != nil {
				continue
			}
			c, readErr := ReadCard(file)
			if readErr != nil {
				return nil, readErr
			}
			cards = append(cards, c)
		}
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].Name < cards[j].Name })
	return cards, nil
}

// Card returns the card named name (case-insensitive).
func (v Vault) Card(name string) (*Card, error) {
	cards, err := v.Cards()
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

// CardForDir returns the card whose repo, folder or workflow link is dir or
// one of its parents (the deepest wins), or nil.
func (v Vault) CardForDir(dir string) *Card {
	cards, err := v.Cards()
	if err != nil {
		return nil
	}
	dir = filepath.Clean(dir)
	var best *Card
	bestLen := 0
	for _, c := range cards {
		for _, l := range c.Links {
			switch strings.ToLower(l.Source) {
			case Layout().Sources.Repo, Layout().Sources.Folder, Layout().Sources.Workflow:
			default:
				continue
			}
			p := ExpandPath(l.Path, v.Root)
			if (dir == p || strings.HasPrefix(dir, p+string(filepath.Separator))) && len(p) > bestLen {
				best, bestLen = c, len(p)
			}
		}
	}
	return best
}

// ProjectName names the project for dir, with the place and link source
// its card records: the git repository's root folder, else dir itself.
func ProjectName(dir string) (string, string, string) {
	if root := gitRoot(dir); root != "" {
		return safeName(filepath.Base(root)), root, Layout().Sources.Repo
	}
	return safeName(filepath.Base(dir)), filepath.Clean(dir), Layout().Sources.Folder
}

// gitRoot is the nearest folder at or above dir holding .git, or "". Like
// git, it does not look into GIT_CEILING_DIRECTORIES; it also stops below
// the home folder, so a home kept in git (dotfiles) does not swallow every
// folder under it.
func gitRoot(dir string) string {
	var stops []string
	for _, s := range filepath.SplitList(os.Getenv("GIT_CEILING_DIRECTORIES")) {
		if s != "" {
			stops = append(stops, filepath.Clean(s))
		}
	}
	home, _ := os.UserHomeDir()
	start := filepath.Clean(dir)
	for d := start; ; {
		if d != start && (d == home || slices.Contains(stops, d)) {
			return ""
		}
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		parent := filepath.Dir(d)
		if parent == d {
			return ""
		}
		d = parent
	}
}

// safeName makes a folder and note name from s.
func safeName(s string) string {
	s = strings.Trim(unsafeNameRe.ReplaceAllString(s, "-"), ". -")
	if s == "" {
		return "root"
	}
	return s
}

// EnsureProject returns dir's project card (archived ones included),
// creating projects/<name>/ and its card when no card covers dir.
func (v Vault) EnsureProject(dir string, now time.Time) (*Card, bool, error) {
	if c := v.CardForDir(dir); c != nil {
		return c, false, nil
	}
	name, place, source := v.NewProjectName(dir)
	return v.CreateProject(name, source, place, now)
}

// NewProjectName is the name, place and link source the project kdeps
// creates for dir gets: ProjectName's, with the parent folder's name in
// front when a card already has that name. Memory and the project use it
// alike, so they always land in the same folder.
func (v Vault) NewProjectName(dir string) (string, string, string) {
	name, place, source := ProjectName(dir)
	if _, err := v.Card(name); err == nil {
		name = safeName(filepath.Base(filepath.Dir(place))) + "-" + name
	}
	return name, place, source
}

// CreateProject creates projects/<name>/ with a card linking place under
// source. A name another place already uses gets the place's parent folder
// in front.
func (v Vault) CreateProject(name, source, place string, now time.Time) (*Card, bool, error) {
	if _, err := v.Card(name); err == nil {
		name = safeName(filepath.Base(filepath.Dir(place))) + "-" + name
	}
	dir := filepath.Join(v.Root, Layout().Folders.Projects, name)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return nil, false, err
	}
	body := fmt.Sprintf("%s\ntype: project\nstatus: active\nnext: %s\nupdated: %s\n%s\n"+
		"| Source | Path |\n|---|---|\n| %s | %s |\n",
		frontDelim, kgnotes.Scalar(noneText), now.Format(time.DateOnly), frontDelim, source, place)
	file := filepath.Join(dir, name+mdExt)
	if err := writeFileAtomic(file, body); err != nil {
		return nil, false, err
	}
	c, err := ReadCard(file)
	return c, true, err
}

// moveProject moves a card's whole project folder under top (projects or
// archive) and returns the card read from its new place.
func (v Vault) moveProject(c *Card, top string) (*Card, error) {
	dst := filepath.Join(v.Root, top, c.Name)
	if _, err := os.Stat(dst); err == nil {
		return nil, fmt.Errorf("%s already exists", dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), dirPerm); err != nil {
		return nil, err
	}
	if err := renameRetry(c.Dir(), dst); err != nil {
		return nil, err
	}
	return ReadCard(filepath.Join(dst, c.Name+mdExt))
}

// renameRetry moves a folder: a rename, retried for a few seconds (on
// Windows a folder cannot be renamed while a file in it is open - an
// indexer, a virus scanner), then a copy of every file and a removal of
// the original. When the original cannot be removed either, the copy is
// removed and the folder stays where it was.
func renameRetry(from, to string) error {
	var err error
	for range renameAttempts {
		if err = os.Rename(from, to); err == nil {
			return nil
		}
		time.Sleep(renameBackoff)
	}
	if copyErr := copyTree(from, to); copyErr != nil {
		_ = os.RemoveAll(to)
		return fmt.Errorf("move %s: %w (copy: %w)", from, err, copyErr)
	}
	if rmErr := os.RemoveAll(from); rmErr != nil {
		_ = os.RemoveAll(to)
		return fmt.Errorf("move %s: %w (remove: %w)", from, err, rmErr)
	}
	return nil
}

// copyTree copies every file under from to the same place under to.
func copyTree(from, to string) error {
	return filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		dst := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, dirPerm)
		}
		_, copyErr := fileflow.Copy(p, dst)
		return copyErr
	})
}

const (
	renameAttempts = 20
	renameBackoff  = 250 * time.Millisecond
)

// SetStatus updates the card and moves the project to archive/ when its
// status becomes archived, or back to projects/ when it stops being.
func (v Vault) SetStatus(c *Card, status, next string, now time.Time) error {
	if err := c.Update(status, next, now); err != nil {
		return err
	}
	archived := strings.EqualFold(c.Status, Layout().Card.Archived)
	if archived == c.Archived() {
		return nil
	}
	top := Layout().Folders.Projects
	if archived {
		top = Layout().Folders.Archive
	}
	moved, err := v.moveProject(c, top)
	if err != nil {
		return err
	}
	*c = *moved
	return nil
}

// Reopen moves an archived project back to projects/ with status active;
// an active project is left alone.
func (v Vault) Reopen(c *Card, now time.Time) error {
	if !c.Archived() {
		return nil
	}
	return v.SetStatus(c, "active", "", now)
}

// IsCardPath reports whether a vault-relative note path is a project card.
func IsCardPath(rel string) bool {
	parts := strings.Split(rel, "/")
	return len(parts) == 3 && (parts[0] == Layout().Folders.Projects || parts[0] == Layout().Folders.Archive) &&
		parts[2] == parts[1]+mdExt
}

// WikiLink is a link to file written with its full vault path, as kdeps
// writes every link: [[projects/app/sessions/2026-10-15 0900 x]]. A path that
// later moves (an archived project) still resolves by its title.
func WikiLink(vaultRoot, file string) string {
	rel, err := filepath.Rel(vaultRoot, file)
	if err != nil || strings.HasPrefix(rel, "..") {
		rel = filepath.Base(file)
	}
	return "[[" + strings.TrimSuffix(filepath.ToSlash(rel), mdExt) + "]]"
}

// VaultRoot is the vault a card lives in.
func (c *Card) VaultRoot() string { return filepath.Dir(filepath.Dir(c.Dir())) }

// SelfLink is the full-path link to the card.
func (c *Card) SelfLink() string { return WikiLink(c.VaultRoot(), c.File) }

// ProjectOf is the project a vault-relative note belongs to, or "".
func ProjectOf(rel string) string {
	parts := strings.Split(rel, "/")
	if len(parts) >= 3 && (parts[0] == Layout().Folders.Projects || parts[0] == Layout().Folders.Archive) {
		return parts[1]
	}
	return ""
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
		Type:    metaString(meta, Layout().Card.Type),
		Status:  metaString(meta, Layout().Card.Status),
		Next:    metaString(meta, Layout().Card.Next),
		Updated: metaString(meta, Layout().Card.Updated),
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

// AddLink adds a "| source | path |" row to the card's links table unless
// the card has that row already, and re-reads the card.
func (c *Card) AddLink(source, place string) error {
	for _, l := range c.Links {
		if strings.EqualFold(l.Source, source) && l.Path == place {
			return nil
		}
	}
	data, err := os.ReadFile(c.File)
	if err != nil {
		return err
	}
	text := strings.TrimRight(string(data), "\n")
	if len(c.Links) == 0 {
		text += "\n| Source | Path |\n|---|---|"
	}
	text += "\n| " + source + " | " + place + " |\n"
	if err = writeFileAtomic(c.File, text); err != nil {
		return err
	}
	updated, err := ReadCard(c.File)
	if err != nil {
		return err
	}
	*c = *updated
	return nil
}

// Places are the card's places on disk: its repo, folder and workflow rows.
func (c *Card) Places() []string {
	var out []string
	for _, l := range c.Links {
		switch strings.ToLower(l.Source) {
		case Layout().Sources.Repo, Layout().Sources.Folder, Layout().Sources.Workflow:
			out = append(out, l.Path)
		}
	}
	return out
}

// Update sets the card's status and/or next step (empty keeps the current
// value) and stamps updated with today's date. Other frontmatter lines and
// the body are kept as they are.
func (c *Card) Update(status, next string, now time.Time) error {
	data, err := os.ReadFile(c.File)
	if err != nil {
		return err
	}
	set := map[string]string{Layout().Card.Updated: now.Format(time.DateOnly)}
	if status != "" {
		set[Layout().Card.Status] = status
	}
	if next != "" {
		set[Layout().Card.Next] = next
	}
	out, err := kgnotes.EditFrontmatter(string(data), set, Layout().CardFields())
	if err != nil {
		return err
	}
	if innerErr := writeFileAtomic(c.File, out); innerErr != nil {
		return innerErr
	}
	updated, err := ReadCard(c.File)
	if err != nil {
		return err
	}
	*c = *updated
	return nil
}
