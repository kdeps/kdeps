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

package assets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// Actions reported in a Change.
const (
	ActionInstall = "install" // new item, or newer version
	ActionPin     = "pin"     // explicit item@version
	ActionReset   = "reset"   // back to the source version
	ActionRemove  = "remove"  // hidden from loaders
	ActionCurrent = "current" // nothing newer
	ActionSkip    = "skip"    // pinned/removed/incompatible; see Note
	ActionFailed  = "failed"  // see Note
)

// Change is what update did (or, in check mode, would do) to one item.
type Change struct {
	ID     string `json:"id"`
	From   string `json:"from,omitempty"`
	To     string `json:"to,omitempty"`
	Action string `json:"action"`
	Note   string `json:"note,omitempty"`
}

// Pending reports whether the change installs or resets something.
func (c Change) Pending() bool {
	return c.Action == ActionInstall || c.Action == ActionPin || c.Action == ActionReset
}

// target is one parsed update argument.
type target struct {
	set, name, version string
}

func parseTargets(args []string) ([]target, error) {
	out := make([]target, 0, len(args))
	for _, a := range args {
		ref, version, _ := strings.Cut(a, "@")
		set, name, err := SplitID(ref)
		if err != nil {
			return nil, err
		}
		if version != "" && name == "" {
			return nil, fmt.Errorf("%s: a version needs an item (set/name@version), not a whole set", a)
		}
		out = append(out, target{set: set, name: name, version: version})
	}
	return out, nil
}

// Update brings items up to date from the index. With no args it covers
// every item (skipping pinned and removed ones); a set name covers that set
// the same way; set/name updates that item to its newest compatible version,
// unpinning and restoring it; set/name@version installs and pins that
// version. With check, nothing is changed.
func Update(ctx context.Context, root string, c *Client, args []string, check bool) ([]Change, error) {
	targets, err := parseTargets(args)
	if err != nil {
		return nil, err
	}
	idx, err := c.Index(ctx)
	if err != nil {
		return nil, err
	}
	local, err := localItems(root)
	if err != nil {
		return nil, err
	}

	var changes []Change
	if len(targets) == 0 {
		if targets, err = defaultTargets(root, local); err != nil {
			return nil, err
		}
	}
	seen := map[string]bool{}
	for _, t := range targets {
		for _, id := range scope(t, local, idx) {
			if seen[id] {
				continue
			}
			seen[id] = true
			changes = append(changes, plan(id, t, local[id], idx.Items[id]))
		}
	}
	changes = withRequires(changes, local, idx)
	if !check {
		applyAll(ctx, root, c, changes, idx)
	}
	return changes, nil
}

// defaultTargets is what a bare update covers: the recorded selection plus
// every item already installed; without a selection, the sets present.
func defaultTargets(root string, local map[string]Item) ([]target, error) {
	lock, err := ReadLock(root)
	if err != nil {
		return nil, err
	}
	if len(lock.Selection) == 0 {
		// No profile recorded: the sets already present (every set on a
		// fresh store), so an update never installs what was not chosen.
		sets := Registered()
		if missing := MissingOf(sets); len(missing) < len(sets) {
			sets = slices.DeleteFunc(slices.Clone(sets), func(s string) bool { return slices.Contains(missing, s) })
		}
		out := make([]target, 0, len(sets))
		for _, s := range sets {
			out = append(out, target{set: s})
		}
		return out, nil
	}
	args := slices.Clone(lock.Selection)
	for id, it := range local {
		if it.Downloaded && !slices.Contains(args, id) && !slices.Contains(args, it.Set) {
			args = append(args, id)
		}
	}
	sort.Strings(args)
	return parseTargets(args)
}

// requires is what the version a change installs needs (see Header.Requires).
func requires(ch Change, idx Index) map[string]string {
	if ch.Action != ActionInstall && ch.Action != ActionPin {
		return nil
	}
	v, _ := idx.Items[ch.ID].Find(ch.To)
	return v.Requires
}

// withRequires adds an install for every item a pending install requires and
// does not have, recursively, and orders each item after what it requires. An
// install whose requirement cannot be met (not published, or pinned outside
// the range) fails instead.
func withRequires(changes []Change, local map[string]Item, idx Index) []Change {
	byID := make(map[string]Change, len(changes))
	order := make([]string, 0, len(changes))
	for _, ch := range changes {
		byID[ch.ID] = ch
		order = append(order, ch.ID)
	}
	queue := slices.Clone(order)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ch := byID[id]
		for _, req := range sortedKeys(requires(ch, idx)) {
			planned, note := need(req, requires(ch, idx)[req], id, byID, local, idx)
			if note != "" {
				ch.Action, ch.Note = ActionFailed, note
				break
			}
			if planned {
				if !slices.Contains(order, req) {
					order = append(order, req)
				}
				queue = append(queue, req) // its new version may require more
			}
		}
		byID[id] = ch
	}
	return requiresFirst(order, byID, idx)
}

// requiresFirst lists the changes in order, each after the ones it requires.
func requiresFirst(order []string, byID map[string]Change, idx Index) []Change {
	out := make([]Change, 0, len(order))
	done := map[string]bool{}
	var visit func(id string)
	visit = func(id string) {
		if done[id] {
			return
		}
		done[id] = true
		for _, req := range sortedKeys(requires(byID[id], idx)) {
			if _, ok := byID[req]; ok {
				visit(req)
			}
		}
		out = append(out, byID[id])
	}
	for _, id := range order {
		visit(id)
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// need makes sure byID leaves req, which parent requires, at a version in rng,
// planning an install when it would not. It reports whether it planned one,
// or why it cannot.
func need(req, rng, parent string, byID map[string]Change, local map[string]Item, idx Index) (bool, string) {
	it, ch := local[req], byID[req]
	cur, pinned := it.Version, it.Pinned
	switch {
	case ch.Action == ActionPin:
		cur, pinned = ch.To, true
	case ch.Pending() && ch.To != "":
		cur = ch.To
	case it.Removed:
		cur = ""
	}
	if cur != "" && inRange(rng, cur) {
		return false, ""
	}
	want := strings.TrimSpace(req + " " + rng)
	if pinned && ch.Action != ActionInstall {
		return false, fmt.Sprintf("needs %s; %s is pinned at %s", want, req, cur)
	}
	best, ok := idx.Items[req].NewestIn(rng)
	if !ok {
		return false, "needs " + want + ", which is not published for this kdeps"
	}
	from := it.Version
	if it.Removed {
		from = ""
	}
	byID[req] = Change{ID: req, From: from, To: best.Version, Action: ActionInstall, Note: "required by " + parent}
	return true, ""
}

// downloadWorkers bounds concurrent downloads: a first run fetches every
// item, and one at a time that takes most of a minute.
const downloadWorkers = 8

// applyAll carries out every pending change in place: downloads run in
// parallel, installs (which rewrite the lock) one at a time.
func applyAll(ctx context.Context, root string, c *Client, changes []Change, idx Index) {
	data := make([][]byte, len(changes))
	errs := make([]error, len(changes))
	sem := make(chan struct{}, downloadWorkers)
	var wg sync.WaitGroup
	for i, ch := range changes {
		if ch.Action != ActionInstall && ch.Action != ActionPin {
			continue
		}
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			v, _ := idx.Items[ch.ID].Find(ch.To)
			data[i], errs[i] = c.File(ctx, ch.ID, v)
		})
	}
	wg.Wait()
	failed := map[string]bool{}
	for i, ch := range changes {
		if ch.Action == ActionFailed {
			failed[ch.ID] = true
		}
		if !ch.Pending() {
			continue
		}
		err := errs[i]
		for req := range requires(ch, idx) {
			if failed[req] {
				err = fmt.Errorf("required %s failed", req)
			}
		}
		switch {
		case err != nil:
		case ch.Action == ActionReset:
			err = Reset(root, ch.ID)
		default:
			err = Install(root, ch.ID, data[i], ch.Action == ActionPin)
		}
		if err != nil {
			changes[i].Action, changes[i].Note = ActionFailed, err.Error()
			failed[ch.ID] = true
		}
	}
}

// scope lists the item ids a target covers.
func scope(t target, local map[string]Item, idx Index) []string {
	if t.name != "" {
		return []string{ID(t.set, t.name)}
	}
	ids := map[string]bool{}
	for id, it := range local {
		if it.Set == t.set {
			ids[id] = true
		}
	}
	for id := range idx.Items {
		if s, n, err := SplitID(id); err == nil && s == t.set && n != "" {
			ids[id] = true
		}
	}
	out := make([]string, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// plan decides what to do with one item.
func plan(id string, t target, it Item, pub IndexItem) Change {
	ch := Change{ID: id, From: it.Version}
	if it.Removed {
		ch.From = ""
	}
	if t.name == "" { // bare update or a whole set
		switch {
		case it.Removed:
			return skip(ch, "removed; update it by name to restore it")
		case it.Pinned:
			return skip(ch, "pinned; update it by name to unpin it")
		}
	}
	if t.version != "" {
		v, ok := pub.Find(t.version)
		switch {
		case !ok:
			ch.Action, ch.Note = ActionFailed, "version "+t.version+" is not published"
		case !Compatible(v.Kdeps):
			ch.Action, ch.Note = ActionFailed, "version "+t.version+" needs kdeps "+v.Kdeps
		default:
			ch.Action, ch.To = ActionPin, v.Version
		}
		return ch
	}
	best, ok := pub.NewestCompatible()
	if t.name != "" && (!ok || !Newer(best.Version, it.BaseVersion)) && it.BaseVersion != "" {
		// Named item with nothing newer than its source version: make sure the
		// source is what is in use (drops a pin, removal or old download).
		if it.Downloaded || it.Pinned || it.Removed {
			ch.Action, ch.To = ActionReset, it.BaseVersion
			return ch
		}
		ch.Action = ActionCurrent
		return ch
	}
	switch {
	case !ok && len(pub.Versions) > 0:
		return skip(ch, "newer versions need a newer kdeps; run kdeps upgrade")
	case !ok:
		if t.name != "" && it.BaseVersion == "" && !it.Downloaded {
			ch.Action, ch.Note = ActionFailed, "no such item"
			return ch
		}
		ch.Action = ActionCurrent
	case it.Removed || it.Pinned || Newer(best.Version, it.Version) || it.Version == "":
		ch.Action, ch.To = ActionInstall, best.Version
	default:
		ch.Action = ActionCurrent
	}
	return ch
}

func skip(ch Change, note string) Change {
	ch.Action, ch.Note = ActionSkip, note
	return ch
}

// localItems maps every known item id to its local state.
func localItems(root string) (map[string]Item, error) {
	out := map[string]Item{}
	for _, s := range Registered() {
		items, err := Items(s, root)
		if err != nil {
			return nil, err
		}
		for _, it := range items {
			out[it.ID] = it
		}
	}
	return out, nil
}

// RemoveItems removes each named item (see Remove).
func RemoveItems(root string, ids []string) []Change {
	local, _ := localItems(root)
	changes := make([]Change, 0, len(ids))
	for _, id := range ids {
		ch := Change{ID: id, From: local[id].Version, Action: ActionRemove}
		if err := Remove(root, id); err != nil {
			ch.Action, ch.Note = ActionFailed, err.Error()
		}
		changes = append(changes, ch)
	}
	return changes
}

// Missing lists the registered sets that have no items, as Open sees them.
func Missing() []string { return MissingOf(Registered()) }

// Initialized reports whether the asset store has been set up (lock written).
func Initialized(root string) bool {
	_, err := os.Stat(filepath.Join(root, lockFileName))
	return err == nil
}

// MarkInitialized writes an empty lock when none exists, so the first-run
// download is not retried.
func MarkInitialized(root string) error {
	if Initialized(root) {
		return nil
	}
	return WriteLock(root, Lock{Items: map[string]LockEntry{}})
}

const checkFileName = "check.json"

// checkCache is the last update check, so the notice costs one download a day.
type checkCache struct {
	CheckedAt time.Time `json:"checkedAt"`
	Updates   []Change  `json:"updates"`
}

// AvailableUpdates returns what a bare kdeps update would install, from a
// cache under root younger than ttl, or a fresh check (cached on success).
func AvailableUpdates(ctx context.Context, root string, c *Client, ttl time.Duration) ([]Change, error) {
	path := filepath.Join(root, checkFileName)
	if data, err := os.ReadFile(path); err == nil {
		var cc checkCache
		if json.Unmarshal(data, &cc) == nil && time.Since(cc.CheckedAt) < ttl {
			return cc.Updates, nil
		}
	}
	changes, err := Update(ctx, root, c, nil, true)
	if err != nil {
		return nil, err
	}
	var pending []Change
	for _, ch := range changes {
		if ch.Pending() {
			pending = append(pending, ch)
		}
	}
	data, err := json.Marshal(checkCache{CheckedAt: time.Now().UTC(), Updates: pending})
	if err == nil {
		if mkErr := os.MkdirAll(root, dirPerm); mkErr == nil {
			err = writeAtomic(path, data)
		}
	}
	return pending, err
}

// ClearCheck drops the cached check, so the next notice reflects an update.
func ClearCheck(root string) {
	_ = os.Remove(filepath.Join(root, checkFileName))
}
