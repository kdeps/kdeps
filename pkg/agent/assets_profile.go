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

package agent

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/assets"
)

// ProfileEnv picks the install profile without asking (first run, or a
// terminal-less one): a profile name, or a comma-separated list of sets.
const ProfileEnv = "KDEPS_PROFILE"

// customProfile names a hand-picked selection in the lock.
const customProfile = "custom"

// EnsureSets makes the sets a command needs available. On first run it asks
// which profile to install (in reads the answers; nil means no terminal,
// then $KDEPS_PROFILE or just the needed sets); afterwards a missing set is
// installed on its own. Without a network it fails naming the update to run.
func EnsureSets(ctx context.Context, w io.Writer, in io.Reader, need []string) error {
	if len(assets.MissingOf(need)) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, firstRunAssetTimeout*5) //nolint:mnd // picker + download
	defer cancel()
	root, c := assets.Root(), assets.NewClient()
	if !assets.Initialized(root) {
		if err := firstRunSelect(ctx, w, in, root, c); err != nil {
			return err
		}
	}
	if missing := assets.MissingOf(need); len(missing) > 0 {
		fmt.Fprintf(w, "Installing %s (needed by this command)...\n", strings.Join(missing, ", "))
		if _, err := assets.Update(ctx, root, c, missing, false); err != nil {
			return notInstalled(missing, err)
		}
		_ = assets.MarkInitialized(root)
	}
	ReloadAssets()
	if missing := assets.MissingOf(need); len(missing) > 0 {
		return notInstalled(missing, nil)
	}
	return nil
}

func notInstalled(sets []string, cause error) error {
	msg := fmt.Sprintf("kdeps assets %s are not installed; connect to the internet and run kdeps registry update %s",
		strings.Join(sets, ", "), strings.Join(sets, " "))
	if cause != nil {
		return fmt.Errorf("%s (%w)", msg, cause)
	}
	return errors.New(msg)
}

// firstRunSelect installs the profile chosen on first run.
func firstRunSelect(ctx context.Context, w io.Writer, in io.Reader, root string, c *assets.Client) error {
	name, targets, err := chooseProfile(ctx, w, in, c, os.Getenv(ProfileEnv))
	if err != nil || name == "" {
		return err
	}
	fmt.Fprintf(w, "Installing the %s profile...\n", name)
	changes, err := assets.Select(ctx, root, c, name, targets)
	if err != nil {
		return err
	}
	for _, ch := range changes {
		if ch.Action == assets.ActionFailed {
			return fmt.Errorf("could not download %s (%s); connect to the internet and run kdeps registry profile %s",
				ch.ID, ch.Note, name)
		}
	}
	return nil
}

// chooseProfile resolves the profile: from preset (a name or a set list),
// else by asking on in, else none (only the command's sets are installed).
func chooseProfile(
	ctx context.Context, w io.Writer, in io.Reader, c *assets.Client, preset string,
) (string, []string, error) {
	if preset == "" && in == nil {
		return "", nil, nil
	}
	idx, err := c.Index(ctx)
	if err != nil {
		return "", nil, err
	}
	if preset != "" {
		return resolveProfileChoice(preset, idx)
	}
	return pickProfile(w, bufio.NewReader(in), idx)
}

// resolveProfileChoice turns a profile name, or a comma-separated list of
// sets and set/name items, into what to install.
func resolveProfileChoice(preset string, idx assets.Index) (string, []string, error) {
	if p, err := assets.ProfileByName(preset); err == nil {
		return p.Name, p.Targets(idx), nil
	}
	sets := strings.Split(preset, ",")
	for i, s := range sets {
		sets[i] = strings.TrimSpace(s)
		set, _, _ := strings.Cut(sets[i], "/")
		if !slices.Contains(assets.Registered(), set) {
			_, err := assets.ProfileByName(preset)
			return "", nil, err
		}
	}
	return customProfile, withBase(sets), nil
}

// withBase puts the base sets first, once.
func withBase(sets []string) []string {
	out := slices.Clone(assets.BaseSets)
	for _, s := range sets {
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	return out
}

// pickProfile shows the profiles and reads the choice; custom continues
// with a set checklist.
func pickProfile(w io.Writer, r *bufio.Reader, idx assets.Index) (string, []string, error) {
	fmt.Fprintln(w, "\nWhat will you use kdeps for? (change it later with kdeps registry profile)")
	for i, p := range assets.Profiles {
		fmt.Fprintf(w, "  %d) %-10s %s\n", i+1, p.Name, p.Description)
	}
	fmt.Fprintf(w, "  %d) %-10s %s\n",
		len(assets.Profiles)+1, customProfile, "tick the asset sets, or single items, yourself")
	def := slices.Index(assets.ProfileNames(), "full") + 1
	for {
		answer := readLine(w, r, fmt.Sprintf("Choice [%d]: ", def))
		n := def
		if answer != "" {
			var err error
			if n, err = strconv.Atoi(answer); err != nil || n < 1 || n > len(assets.Profiles)+1 {
				fmt.Fprintln(w, "  pick a number from the list")
				continue
			}
		}
		if n == len(assets.Profiles)+1 {
			return customProfile, pickSets(w, r, idx), nil
		}
		p := assets.Profiles[n-1]
		return p.Name, p.Targets(idx), nil
	}
}

// checklist is the custom selection: every published item of every set,
// ticked or not. Base sets cannot be emptied and required items stay
// ticked while anything in their set is.
type checklist struct {
	sets   []string
	items  map[string][]string // set -> item names, sorted
	ticked map[string]bool     // item id
}

func newChecklist(idx assets.Index) *checklist {
	c := &checklist{sets: assets.Registered(), items: map[string][]string{}, ticked: map[string]bool{}}
	for id := range idx.Items {
		if s, n, err := assets.SplitID(id); err == nil && slices.Contains(c.sets, s) {
			c.items[s] = append(c.items[s], n)
		}
	}
	for _, s := range c.sets {
		slices.Sort(c.items[s])
	}
	for _, s := range assets.BaseSets {
		c.setAll(s, true)
	}
	return c
}

func (c *checklist) count(set string) int {
	n := 0
	for _, name := range c.items[set] {
		if c.ticked[assets.ID(set, name)] {
			n++
		}
	}
	return n
}

// setAll ticks or unticks every item of set. Unticking a base set keeps
// its required items, or its first item when none is required.
func (c *checklist) setAll(set string, on bool) {
	base := slices.Contains(assets.BaseSets, set)
	for _, name := range c.items[set] {
		c.ticked[assets.ID(set, name)] = on || (base && assets.Required(set, name))
	}
	if base && c.count(set) == 0 && len(c.items[set]) > 0 {
		c.ticked[assets.ID(set, c.items[set][0])] = true
	}
}

// toggle flips one item and keeps the set's required items ticked while
// any item is (and a base set never empty).
func (c *checklist) toggle(set, name string) {
	id := assets.ID(set, name)
	if assets.Required(set, name) && (c.count(set) > 1 || slices.Contains(assets.BaseSets, set)) {
		return
	}
	c.ticked[id] = !c.ticked[id]
	if c.count(set) == 0 && slices.Contains(assets.BaseSets, set) {
		c.ticked[id] = true
		return
	}
	if c.count(set) > 0 {
		for _, n := range c.items[set] {
			if assets.Required(set, n) {
				c.ticked[assets.ID(set, n)] = true
			}
		}
	}
}

// targets is the selection as kdeps registry update arguments: a fully ticked set by
// name (so items published later are included), else its ticked items.
func (c *checklist) targets() []string {
	var out []string
	for _, s := range c.sets {
		n := c.count(s)
		switch {
		case n == 0:
		case n == len(c.items[s]):
			out = append(out, s)
		default:
			for _, name := range c.items[s] {
				if c.ticked[assets.ID(s, name)] {
					out = append(out, assets.ID(s, name))
				}
			}
		}
	}
	return out
}

// pickSets is the custom checklist: sets are toggled by number, and oN
// opens set N to tick single items, until an empty answer.
func pickSets(w io.Writer, r *bufio.Reader, idx assets.Index) []string {
	c := newChecklist(idx)
	for {
		c.printSets(w)
		answer := readLine(w, r, "Toggle sets (e.g. 5 7), o<N> to pick items in set N (e.g. o6), Enter to install: ")
		if answer == "" {
			return c.targets()
		}
		for _, f := range strings.Fields(strings.ReplaceAll(answer, ",", " ")) {
			if n, ok := strings.CutPrefix(strings.ToLower(f), "o"); ok {
				if i, err := strconv.Atoi(n); err == nil && i >= 1 && i <= len(c.sets) {
					c.pickItems(w, r, c.sets[i-1])
				}
				continue
			}
			for _, i := range parseNumbers(f, len(c.sets)) {
				s := c.sets[i-1]
				c.setAll(s, c.count(s) < len(c.items[s]))
			}
		}
	}
}

// pickItems is the item view of one set: numbers and ranges toggle items,
// a ticks all, n unticks all, Enter goes back.
func (c *checklist) pickItems(w io.Writer, r *bufio.Reader, set string) {
	names := c.items[set]
	for {
		fmt.Fprintf(w, "\n%s (%d/%d ticked; * required):\n", set, c.count(set), len(names))
		for i, name := range names {
			mark, note := "[ ]", ""
			if c.ticked[assets.ID(set, name)] {
				mark = "[x]"
			}
			if assets.Required(set, name) {
				note = " *"
			}
			fmt.Fprintf(w, "  %3d) %s %s%s\n", i+1, mark, name, note)
		}
		answer := readLine(w, r, "Toggle items (e.g. 3 5-9), a = all, n = none, Enter to go back: ")
		switch answer {
		case "":
			return
		case "a", "n":
			c.setAll(set, answer == "a")
			continue
		}
		for _, f := range strings.Fields(strings.ReplaceAll(answer, ",", " ")) {
			for _, i := range parseNumbers(f, len(names)) {
				c.toggle(set, names[i-1])
			}
		}
	}
}

func (c *checklist) printSets(w io.Writer) {
	fmt.Fprintln(w, "\nAsset sets ([x] all, [-] some; the base sets are always installed):")
	for i, s := range c.sets {
		n, total := c.count(s), len(c.items[s])
		mark, note := "[ ]", ""
		switch {
		case n > 0 && n == total:
			mark = "[x]"
		case n > 0:
			mark = "[-]"
		}
		if slices.Contains(assets.BaseSets, s) {
			note = " (base)"
		}
		fmt.Fprintf(w, "  %2d) %s %-11s %3d/%-3d items%s\n", i+1, mark, s, n, total, note)
	}
}

// parseNumbers reads "7" or "3-9" as list positions within 1..limit.
func parseNumbers(f string, limit int) []int {
	lo, hi, isRange := strings.Cut(f, "-")
	a, err := strconv.Atoi(lo)
	if err != nil {
		return nil
	}
	b := a
	if isRange {
		if b, err = strconv.Atoi(hi); err != nil {
			return nil
		}
	}
	var out []int
	for i := max(a, 1); i <= min(b, limit); i++ {
		out = append(out, i)
	}
	return out
}

func readLine(w io.Writer, r *bufio.Reader, prompt string) string {
	fmt.Fprint(w, prompt)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

// SelectProfile installs a profile (or a custom selection when name is
// "custom" or empty and in is a terminal) and records it: kdeps registry update
// profile [name].
func SelectProfile(ctx context.Context, w io.Writer, in io.Reader, name string) error {
	root, c := assets.Root(), assets.NewClient()
	idx, err := c.Index(ctx)
	if err != nil {
		return err
	}
	var profile string
	var targets []string
	switch name {
	case "", customProfile:
		if in == nil {
			return errors.New("choosing a profile needs a terminal; name one: kdeps registry profile <name>")
		}
		if name == customProfile {
			profile, targets = customProfile, pickSets(w, bufio.NewReader(in), idx)
		} else if profile, targets, err = pickProfile(w, bufio.NewReader(in), idx); err != nil {
			return err
		}
	default:
		if profile, targets, err = resolveProfileChoice(name, idx); err != nil {
			return err
		}
	}
	changes, err := assets.Select(ctx, root, c, profile, targets)
	if err != nil {
		return err
	}
	printChanges(w, changes, false)
	assets.ClearCheck(root)
	ReloadAssets()
	return nil
}

// InstallComponentOnDemand returns the on-demand installer kdeps sets for
// the run: a missing item (a component a workflow calls) is installed from
// kdeps.io the first time it is needed, unless downloads are off.
func InstallComponentOnDemand(ctx context.Context, w io.Writer) func(id string) bool {
	return func(id string) bool {
		root, c := assets.Root(), assets.NewClient()
		changes, err := assets.Update(ctx, root, c, []string{id}, false)
		if err != nil {
			return false
		}
		for _, ch := range changes {
			if ch.ID == id && ch.Action == assets.ActionInstall {
				fmt.Fprintf(w, "Installed %s %s (first use).\n", id, ch.To)
				return true
			}
		}
		return false
	}
}
