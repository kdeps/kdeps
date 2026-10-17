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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gitRepo makes a repository with one commit on branch main and returns it.
func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t",
			"GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL="+os.DevNull)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "first")
	return dir
}

func gitCommit(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "commit", "-q", "--allow-empty", "-m", msg)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t",
		"GIT_COMMITTER_EMAIL=t@t", "GIT_CONFIG_GLOBAL="+os.DevNull)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func TestReadGitHead(t *testing.T) {
	dir := gitRepo(t)
	head, ok := ReadGitHead(t.Context(), dir)
	require.True(t, ok)
	assert.Equal(t, "main", head.Branch)
	assert.Len(t, head.Commit, 12)

	_, ok = ReadGitHead(t.Context(), t.TempDir())
	assert.False(t, ok, "not a repository")

	cmd := exec.Command("git", "-C", dir, "checkout", "-q", "--detach")
	require.NoError(t, cmd.Run())
	detached, ok := ReadGitHead(t.Context(), dir)
	require.True(t, ok)
	assert.Empty(t, detached.Branch, "detached HEAD has no branch")
	assert.Equal(t, head.Commit, detached.Commit)
}

func TestSetBranch(t *testing.T) {
	v, card, _ := searchFixture(t)
	require.NoError(t, card.SetBranch("feat/a"))
	assert.Equal(t, "feat/a", card.Branch())
	require.NoError(t, card.SetBranch("feat/a"), "same branch: unchanged")
	require.NoError(t, card.SetBranch("feat/b"))
	data, _ := os.ReadFile(card.File)
	assert.Equal(t, 1, strings.Count(string(data), "| branch | "), "one branch row")
	assert.Contains(t, string(data), "| branch | feat/b |")
	assert.NotNil(t, v.CardForDir(card.Places()[0]), "a branch row is not a place")
	require.NoError(t, card.SetBranch(""))
	assert.Empty(t, card.Branch())
}

func TestWriteSession_RecordsTheGitRange(t *testing.T) {
	_, card, _ := searchFixture(t)
	now := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	summary := "## Goal\nShip git pointers\n"
	sn, err := WriteSession(
		card,
		"s1",
		summary,
		now,
		GitRange{Branch: "feat/x", From: "aaaaaaaaaaaa", To: "aaaaaaaaaaaa"},
	)
	require.NoError(t, err)
	data, _ := os.ReadFile(sn.File)
	assert.Contains(t, string(data), "branch: feat/x\ncommits: aaaaaaaaaaaa\n")
	assert.Contains(t, string(data), "no commits in this session yet")

	// Rewritten later in a fresh process: the start it recorded stays.
	sn, err = WriteSession(
		card,
		"s1",
		summary,
		now,
		GitRange{Branch: "feat/x", From: "bbbbbbbbbbbb", To: "cccccccccccc"},
	)
	require.NoError(t, err)
	data, _ = os.ReadFile(sn.File)
	assert.Contains(t, string(data), "commits: aaaaaaaaaaaa..cccccccccccc")
	assert.Contains(t, string(data), "`git log aaaaaaaaaaaa..cccccccccccc`")

	other, err := WriteSession(card, "s2", summary, now, GitRange{})
	require.NoError(t, err)
	data, _ = os.ReadFile(other.File)
	assert.NotContains(t, string(data), "commits:", "outside a repository: no git lines")
	assert.NotContains(t, string(data), "## Git")
	assert.Equal(t, "", recordedGitStart(filepath.Join(t.TempDir(), "none.md")))
}

func TestGitSection(t *testing.T) {
	assert.Empty(t, gitSection(GitRange{}))
	assert.Contains(t, gitSection(GitRange{To: "abc"}), "detached HEAD at `abc`")
	assert.Contains(t, gitSection(GitRange{Branch: "main", From: "a", To: "b"}), "branch `main`, commits `a..b`")
	assert.Equal(t, "b", gitRangeMeta(GitRange{From: "b", To: "b"}))
	assert.Equal(t, "a..b", gitRangeMeta(GitRange{From: "a", To: "b"}))
}

func TestReadGitHead_FollowsNewCommits(t *testing.T) {
	dir := gitRepo(t)
	first, _ := ReadGitHead(t.Context(), dir)
	gitCommit(t, dir, "second")
	second, _ := ReadGitHead(t.Context(), dir)
	assert.NotEqual(t, first.Commit, second.Commit)
}
