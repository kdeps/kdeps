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
	"context"
	"os/exec"
	"strings"
)

// GitPoint is where a git repository is: its branch and HEAD commit.
type GitPoint struct {
	Branch string // "" when HEAD is detached
	Commit string // short commit id
}

// GitRange is the work a session note covers: from the commit HEAD was at
// when its thread began to the one it is at now, on Branch. The vault
// records only these pointers; git keeps the history (git log From..To).
type GitRange struct {
	Branch string
	From   string
	To     string
}

// gitShortLen is how many hex digits a recorded commit id has.
const gitShortLen = "12"

// ReadGitHead is dir's branch and HEAD commit, or false when dir is not in
// a git repository (or git is missing). It only reads, with two
// git rev-parse calls.
func ReadGitHead(ctx context.Context, dir string) (GitPoint, bool) {
	commit, err := gitOut(ctx, dir, "rev-parse", "--short="+gitShortLen, "HEAD")
	if err != nil || commit == "" {
		return GitPoint{}, false
	}
	branch, _ := gitOut(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
	if branch == "HEAD" {
		branch = ""
	}
	return GitPoint{Branch: branch, Commit: commit}, true
}

func gitOut(ctx context.Context, dir string, args ...string) (string, error) {
	argv := append([]string{"-C", dir}, args...)
	// The arguments are fixed rev-parse flags and the working directory.
	out, err := exec.CommandContext(ctx, "git", argv...).Output()
	return strings.TrimSpace(string(out)), err
}

// SourceBranch is the card row naming the branch the project is on.
const SourceBranch = "branch"

// SetBranch records the branch the project is on as the card's one branch
// row, replacing an older one; "" removes it.
func (c *Card) SetBranch(branch string) error {
	for _, l := range c.Links {
		if strings.EqualFold(l.Source, SourceBranch) {
			if l.Path == branch {
				return nil
			}
			if err := c.RemoveLink(l.Source, l.Path); err != nil {
				return err
			}
		}
	}
	if branch == "" {
		return nil
	}
	return c.AddLink(SourceBranch, branch)
}

// Branch is the branch the card's branch row names, or "".
func (c *Card) Branch() string {
	for _, l := range c.Links {
		if strings.EqualFold(l.Source, SourceBranch) {
			return l.Path
		}
	}
	return ""
}

// gitRangeMeta is the frontmatter value of a range: "from..to", or "to"
// when no commit was made.
func gitRangeMeta(r GitRange) string {
	if r.From == "" || r.From == r.To {
		return r.To
	}
	return r.From + ".." + r.To
}

// gitSection is the session note's Git section, or "" without a range.
func gitSection(r GitRange) string {
	if r.To == "" {
		return ""
	}
	where := "detached HEAD"
	if r.Branch != "" {
		where = "branch `" + r.Branch + "`"
	}
	if r.From == "" || r.From == r.To {
		return "On " + where + " at `" + r.To + "`; no commits in this session yet."
	}
	return "On " + where + ", commits `" + r.From + ".." + r.To + "`: `git log " + r.From + ".." + r.To +
		"` shows them (`git diff " + r.From + ".." + r.To + "` the change)."
}
