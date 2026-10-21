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

package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	toolGitOps = "git_ops"

	argPaths      = "paths"
	argWorkingDir = "workingDir"
	argMessage    = "message"
	argBranch     = "branch"
	argRemote     = "remote"
	argArgs       = "args"
	argMaxCount   = "maxCount"

	gitDefaultRemote = "origin"
	gitDefaultLogs   = 10
	msgGitMessage    = "git: commit message is required"
)

// GitOps runs one git operation and returns its JSON result.
// Operations: status, diff, log, show, branch, remote, add, commit, checkout,
// init, clone, push, pull.
func GitOps(ctx context.Context, args map[string]any) (string, error) {
	if err := ctxReady(ctx, toolGitOps); err != nil {
		return "", err
	}
	result, err := runGitOp(ctx, args)
	if err != nil {
		return "", err
	}
	return marshalIndent(result)
}

func runGitOp(ctx context.Context, args map[string]any) (map[string]any, error) {
	paths, err := gitSplit(fileArg(args, argPaths))
	if err != nil {
		return nil, fmt.Errorf("git: paths: %w", err)
	}
	extra, err := gitSplit(fileArg(args, argArgs))
	if err != nil {
		return nil, fmt.Errorf("git: args: %w", err)
	}
	remote := fileArg(args, argRemote)
	if remote == "" {
		remote = gitDefaultRemote
	}
	op := fileArg(args, argOperation)
	switch op {
	case "status":
		return gitStatus(ctx, args)
	case "diff":
		return gitDiff(ctx, args, paths)
	case "log":
		return gitLog(ctx, args, paths)
	case "show":
		return gitShow(ctx, args, extra)
	case "branch":
		return gitBranch(ctx, args)
	case "remote":
		return gitRemote(ctx, args)
	case "add":
		return gitAdd(ctx, args, paths)
	case "commit":
		return gitCommit(ctx, args)
	case "checkout":
		return gitCheckout(ctx, args, paths)
	case "init":
		return gitInit(ctx, args)
	case "clone":
		return gitClone(ctx, args)
	case "push", "pull":
		return gitSync(ctx, args, op, remote)
	default:
		return nil, fmt.Errorf("git: unsupported operation '%s'", op)
	}
}

// gitRunner runs git in dir. An empty dir uses the process working directory.
type gitRunner struct {
	ctx context.Context
	dir string
}

func (g gitRunner) run(args ...string) (string, error) {
	cmd := exec.CommandContext(g.ctx, "git", args...)
	if g.dir != "" {
		cmd.Dir = g.dir
	}
	cmd.Env = append(os.Environ(), "GIT_PAGER=cat", "GIT_TERMINAL_PROMPT=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		code := -1
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		}
		msg := fmt.Sprintf("git %s failed: exit status %d", args[0], code)
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			msg += ": " + detail
		}
		return "", errors.New(msg)
	}
	return stdout.String(), nil
}

func gitRunnerFor(ctx context.Context, args map[string]any) gitRunner {
	return gitRunner{ctx: ctx, dir: fileArg(args, argWorkingDir)}
}

func gitStatus(ctx context.Context, args map[string]any) (map[string]any, error) {
	out, err := gitRunnerFor(ctx, args).run("status", "--porcelain", "-b")
	if err != nil {
		return nil, err
	}
	branch := ""
	staged, unstaged, untracked, conflicts := []string{}, []string{}, []string{}, []string{}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "##") {
			// "## branch...upstream": keep the branch, drop the tracking info.
			parts := strings.SplitN(strings.TrimPrefix(line, "##"), "...", 2) //nolint:mnd // branch, upstream
			branch = strings.TrimSpace(parts[0])
			continue
		}
		if len(line) < 4 { //nolint:mnd // "XY path": two status chars, a space, a path
			continue
		}
		xy := line[:2]
		f := strings.TrimSpace(line[3:])
		switch {
		case xy == "??":
			untracked = append(untracked, f)
		case strings.Contains(xy, "U"):
			conflicts = append(conflicts, f)
		case !strings.ContainsRune(" ?", rune(xy[0])):
			staged = append(staged, f)
		case xy[1] != ' ':
			unstaged = append(unstaged, f)
		}
	}
	return fileSuccess(map[string]any{
		"branch": branch, "staged": staged, "unstaged": unstaged,
		"untracked": untracked, "conflicts": conflicts,
	}), nil
}

func gitDiff(ctx context.Context, args map[string]any, paths []string) (map[string]any, error) {
	cmdArgs := []string{"diff", "--no-color"}
	if len(paths) > 0 {
		cmdArgs = append(append(cmdArgs, "--"), paths...)
	}
	out, err := gitRunnerFor(ctx, args).run(cmdArgs...)
	if err != nil {
		return nil, err
	}
	additions := strings.Count(out, "\n+") - strings.Count(out, "\n+++")
	deletions := strings.Count(out, "\n-") - strings.Count(out, "\n---")
	return fileSuccess(map[string]any{"diff": out, "additions": additions, "deletions": deletions}), nil
}

func gitLog(ctx context.Context, args map[string]any, paths []string) (map[string]any, error) {
	n := gitLogCount(args[argMaxCount])
	cmdArgs := []string{"log", fmt.Sprintf("--max-count=%d", n), "--format=%H|%an|%ae|%ai|%s"}
	if len(paths) > 0 {
		cmdArgs = append(append(cmdArgs, "--"), paths...)
	}
	out, err := gitRunnerFor(ctx, args).run(cmdArgs...)
	if err != nil {
		return nil, err
	}
	commits := []map[string]any{}
	for _, line := range strings.Split(out, "\n") {
		if strings.Count(line, "|") < 4 { //nolint:mnd // hash|author|email|date|message
			continue
		}
		parts := strings.SplitN(line, "|", 5) //nolint:mnd // hash, author, email, date, message
		commits = append(commits, map[string]any{
			"hash": parts[0], "author": parts[1], "email": parts[2],
			"date": parts[3], "message": parts[4],
		})
	}
	return fileSuccess(map[string]any{"commits": commits, "count": len(commits)}), nil
}

func gitLogCount(v any) int {
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return int(n)
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(n)); err == nil && parsed > 0 {
			return parsed
		}
	}
	return gitDefaultLogs
}

func gitShow(ctx context.Context, args map[string]any, extra []string) (map[string]any, error) {
	out, err := gitRunnerFor(ctx, args).run(append([]string{"show", "--no-color", "--format=fuller"}, extra...)...)
	if err != nil {
		return nil, err
	}
	return fileSuccess(map[string]any{"output": out}), nil
}

func gitBranch(ctx context.Context, args map[string]any) (map[string]any, error) {
	out, err := gitRunnerFor(ctx, args).run("branch", "-a")
	if err != nil {
		return nil, err
	}
	branches := []map[string]any{}
	current := ""
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 2 { //nolint:mnd // "* name" for the current branch
			continue
		}
		isCurrent := line[0] == '*'
		name := strings.TrimSpace(line[1:])
		isRemote := strings.HasPrefix(name, "remotes/")
		name = strings.TrimPrefix(name, "remotes/")
		if isCurrent {
			current = name
		}
		branches = append(branches, map[string]any{
			"name": name, "current": isCurrent, "remote": isRemote,
		})
	}
	return fileSuccess(map[string]any{"branches": branches, "current": current, "count": len(branches)}), nil
}

func gitRemote(ctx context.Context, args map[string]any) (map[string]any, error) {
	out, err := gitRunnerFor(ctx, args).run("remote", "-v")
	if err != nil {
		return nil, err
	}
	remotes := []map[string]any{}
	seen := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 { //nolint:mnd // name, url, "(fetch)"/"(push)"
			continue
		}
		key := fields[0] + "\x00" + fields[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		remotes = append(remotes, map[string]any{
			"name": fields[0], "url": fields[1], "type": fields[2],
		})
	}
	return fileSuccess(map[string]any{"remotes": remotes, "count": len(remotes)}), nil
}

func gitAdd(ctx context.Context, args map[string]any, paths []string) (map[string]any, error) {
	if fileBool(args, argDryRun) {
		return fileSuccess(map[string]any{"dryRun": true, "staged": false}), nil
	}
	if len(paths) == 0 {
		paths = []string{"."}
	}
	if _, err := gitRunnerFor(ctx, args).run(append([]string{"add"}, paths...)...); err != nil {
		return nil, err
	}
	return fileSuccess(map[string]any{"staged": true}), nil
}

func gitCommit(ctx context.Context, args map[string]any) (map[string]any, error) {
	message := fileArg(args, argMessage)
	if message == "" {
		return nil, errors.New(msgGitMessage)
	}
	if fileBool(args, argDryRun) {
		return fileSuccess(map[string]any{"dryRun": true, "message": message, "committed": false}), nil
	}
	if _, err := gitRunnerFor(ctx, args).run("commit", "-m", message); err != nil {
		return nil, err
	}
	return fileSuccess(map[string]any{"message": message, "committed": true}), nil
}

func gitCheckout(ctx context.Context, args map[string]any, paths []string) (map[string]any, error) {
	branch := fileArg(args, argBranch)
	if fileBool(args, argDryRun) {
		return fileSuccess(map[string]any{
			"dryRun": true, "branch": branch, "paths": paths, "switched": false,
		}), nil
	}
	cmdArgs := []string{"checkout"}
	if branch != "" {
		cmdArgs = append(cmdArgs, branch)
	}
	if len(paths) > 0 {
		cmdArgs = append(append(cmdArgs, "--"), paths...)
	}
	if _, err := gitRunnerFor(ctx, args).run(cmdArgs...); err != nil {
		return nil, err
	}
	return fileSuccess(map[string]any{"branch": branch, "paths": paths, "switched": true}), nil
}

func gitInit(ctx context.Context, args map[string]any) (map[string]any, error) {
	if fileBool(args, argDryRun) {
		return fileSuccess(map[string]any{"dryRun": true, "initialized": false}), nil
	}
	if _, err := gitRunnerFor(ctx, args).run("init"); err != nil {
		return nil, err
	}
	dir := fileArg(args, argWorkingDir)
	if dir == "" {
		dir, _ = os.Getwd()
	}
	return fileSuccess(map[string]any{"path": dir, "initialized": true}), nil
}

func gitClone(ctx context.Context, args map[string]any) (map[string]any, error) {
	url := fileArg(args, argURL)
	if fileBool(args, argDryRun) {
		return fileSuccess(map[string]any{"dryRun": true, "url": url, "cloned": false}), nil
	}
	cmdArgs := []string{"clone", url}
	if dir := fileArg(args, argWorkingDir); dir != "" {
		cmdArgs = append(cmdArgs, dir)
	}
	// Clone runs from the process working directory; workingDir is the target.
	if _, err := (gitRunner{ctx: ctx}).run(cmdArgs...); err != nil {
		return nil, err
	}
	return fileSuccess(map[string]any{"url": url, "cloned": true}), nil
}

func gitSync(ctx context.Context, args map[string]any, op, remote string) (map[string]any, error) {
	key := op + "ed"
	branch := fileArg(args, argBranch)
	if fileBool(args, argDryRun) {
		return fileSuccess(map[string]any{
			"dryRun": true, "remote": remote, "branch": branch, key: false,
		}), nil
	}
	cmdArgs := []string{op, remote}
	if branch != "" {
		cmdArgs = append(cmdArgs, branch)
	}
	if _, err := gitRunnerFor(ctx, args).run(cmdArgs...); err != nil {
		return nil, err
	}
	return fileSuccess(map[string]any{"remote": remote, "branch": branch, key: true}), nil
}

// gitSplit splits args on spaces the way shlex does: single quotes preserve
// everything, double quotes allow \" and \\, and a backslash escapes the next
// character outside quotes.
func gitSplit(s string) ([]string, error) {
	var (
		out     = []string{}
		current strings.Builder
		started bool
	)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'':
			started = true
			end := strings.IndexByte(s[i+1:], '\'')
			if end < 0 {
				return nil, errors.New("no closing quotation")
			}
			current.WriteString(s[i+1 : i+1+end])
			i += end + 1
		case c == '"':
			started = true
			value, next, err := gitDoubleQuoted(s[i+1:])
			if err != nil {
				return nil, err
			}
			current.WriteString(value)
			i += next
		case c == '\\' && i+1 < len(s):
			started = true
			i++
			current.WriteByte(s[i])
		case c == ' ' || c == '\t' || c == '\n':
			if started {
				out = append(out, current.String())
				current.Reset()
				started = false
			}
		default:
			started = true
			current.WriteByte(c)
		}
	}
	if started {
		out = append(out, current.String())
	}
	return out, nil
}

func gitDoubleQuoted(s string) (string, int, error) {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			return b.String(), i + 1, nil
		case '\\':
			if i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
				i++
				b.WriteByte(s[i])
				continue
			}
			b.WriteByte(s[i])
		default:
			b.WriteByte(s[i])
		}
	}
	return "", 0, errors.New("no closing quotation")
}
