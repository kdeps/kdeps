//go:build !js

package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

const (
	defaultMaxCount = 10
	defaultRemote   = "origin"
)

// CommandRunner runs git commands (overridable for testing).
type CommandRunner interface {
	Run(cmd *exec.Cmd) (stdout, stderr string, exitCode int, err error)
}

// DefaultCommandRunner implements CommandRunner using os/exec.
type DefaultCommandRunner struct{}

func (r *DefaultCommandRunner) Run(cmd *exec.Cmd) (string, string, int, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			return "", "", -1, err
		}
	}
	return stdout.String(), stderr.String(), exitCode, nil
}

// Executor executes git operations.
type Executor struct {
	runner CommandRunner
}

// NewExecutor creates a new git executor.
func NewExecutor() *Executor {
	return &Executor{runner: &DefaultCommandRunner{}}
}

// NewExecutorWithRunner creates a new git executor with a custom runner (for testing).
func NewExecutorWithRunner(runner CommandRunner) *Executor {
	return &Executor{runner: runner}
}

// Execute dispatches to the appropriate git operation.
func (e *Executor) Execute(
	_ *executor.ExecutionContext,
	config *domain.GitResourceConfig,
) (interface{}, error) {
	if config.Operation == "" {
		return nil, errors.New("git: operation is required")
	}

	switch config.Operation {
	case domain.GitOpStatus:
		return e.status(config)
	case domain.GitOpDiff:
		return e.diff(config)
	case domain.GitOpLog:
		return e.log(config)
	case domain.GitOpShow:
		return e.show(config)
	case domain.GitOpBranch:
		return e.branch(config)
	case domain.GitOpRemote:
		return e.remote(config)
	case domain.GitOpAdd:
		return e.add(config)
	case domain.GitOpCommit:
		return e.commit(config)
	case domain.GitOpCheckout:
		return e.checkout(config)
	case domain.GitOpInit:
		return e.init(config)
	case domain.GitOpClone:
		return e.cloneOp(config)
	case domain.GitOpPush:
		return e.push(config)
	case domain.GitOpPull:
		return e.pull(config)
	default:
		return nil, fmt.Errorf("git: unsupported operation %q", config.Operation)
	}
}

// --- helpers ---

func (e *Executor) buildGitCmd(config *domain.GitResourceConfig, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(),
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
	)
	if config.WorkingDir != "" {
		cmd.Dir = config.WorkingDir
	}
	return cmd
}

func result(success bool, data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["success"] = success
	return data
}

// --- Read operations ---

func (e *Executor) status(config *domain.GitResourceConfig) (interface{}, error) {
	cmd := e.buildGitCmd(config, "status", "--porcelain", "-b")
	stdout, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git status failed: %w", err)
	}

	branch := ""
	var staged, unstaged, untracked, conflicts []string

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "##") {
			parts := strings.SplitN(line[2:], "...", 2)
			branch = strings.TrimSpace(parts[0])
			continue
		}
		if len(line) < 4 {
			continue
		}
		xy := line[:2]
		file := strings.TrimSpace(line[3:])
		switch {
		case xy == "??":
			untracked = append(untracked, file)
		case xy[0] == 'U' || xy[1] == 'U':
			conflicts = append(conflicts, file)
		case xy[0] != ' ' && xy[0] != '?':
			staged = append(staged, file)
		case xy[1] != ' ':
			unstaged = append(unstaged, file)
		}
	}

	return result(true, map[string]interface{}{
		"branch":    branch,
		"staged":    staged,
		"unstaged":  unstaged,
		"untracked": untracked,
		"conflicts": conflicts,
	}), nil
}

func (e *Executor) diff(config *domain.GitResourceConfig) (interface{}, error) {
	args := []string{"diff", "--no-color"}
	if config.Paths != nil {
		args = append(args, "--")
		args = append(args, config.Paths...)
	}
	cmd := e.buildGitCmd(config, args...)
	stdout, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git diff failed: %w", err)
	}

	additions := strings.Count(stdout, "\n+") - strings.Count(stdout, "\n+++")
	deletions := strings.Count(stdout, "\n-") - strings.Count(stdout, "\n---")

	return result(true, map[string]interface{}{
		"diff":      stdout,
		"additions": additions,
		"deletions": deletions,
	}), nil
}

func (e *Executor) log(config *domain.GitResourceConfig) (interface{}, error) {
	maxCount := config.MaxCount
	if maxCount <= 0 {
		maxCount = defaultMaxCount
	}

	args := []string{"log", "--oneline",
		fmt.Sprintf("--max-count=%d", maxCount),
		"--format=%H|%an|%ae|%ai|%s"}
	if config.Paths != nil {
		args = append(args, "--")
		args = append(args, config.Paths...)
	}
	cmd := e.buildGitCmd(config, args...)
	stdout, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git log failed: %w", err)
	}

	var commits []map[string]interface{}
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 5)
		if len(parts) >= 5 {
			commits = append(commits, map[string]interface{}{
				"hash":    parts[0],
				"author":  parts[1],
				"email":   parts[2],
				"date":    parts[3],
				"message": parts[4],
			})
		}
	}

	return result(true, map[string]interface{}{
		"commits": commits,
		"count":   len(commits),
	}), nil
}

func (e *Executor) show(config *domain.GitResourceConfig) (interface{}, error) {
	args := []string{"show", "--no-color", "--format=fuller"}
	if len(config.Args) > 0 {
		args = append(args, config.Args...)
	}
	cmd := e.buildGitCmd(config, args...)
	stdout, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git show failed: %w", err)
	}
	return result(true, map[string]interface{}{"output": stdout}), nil
}

func (e *Executor) branch(config *domain.GitResourceConfig) (interface{}, error) {
	cmd := e.buildGitCmd(config, "branch", "-a")
	stdout, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git branch failed: %w", err)
	}

	var branches []map[string]interface{}
	current := ""
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}
		isCurrent := line[0] == '*'
		name := strings.TrimSpace(line[1:])
		isRemote := strings.HasPrefix(name, "remotes/")
		if isRemote {
			name = strings.TrimPrefix(name, "remotes/")
		}
		if isCurrent {
			current = name
		}
		branches = append(branches, map[string]interface{}{
			"name":    name,
			"current": isCurrent,
			"remote":  isRemote,
		})
	}

	return result(true, map[string]interface{}{
		"branches": branches,
		"current":  current,
		"count":    len(branches),
	}), nil
}

func (e *Executor) remote(config *domain.GitResourceConfig) (interface{}, error) {
	cmd := e.buildGitCmd(config, "remote", "-v")
	stdout, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git remote failed: %w", err)
	}

	var remotes []map[string]interface{}
	seen := map[string]bool{}
	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			key := parts[0] + "|" + parts[1]
			if !seen[key] {
				seen[key] = true
				remotes = append(remotes, map[string]interface{}{
					"name": parts[0],
					"url":  parts[1],
					"type": parts[2],
				})
			}
		}
	}

	return result(true, map[string]interface{}{
		"remotes": remotes,
		"count":   len(remotes),
	}), nil
}

// --- Write operations ---

func (e *Executor) add(config *domain.GitResourceConfig) (interface{}, error) {
	if config.DryRun {
		return result(true, map[string]interface{}{
			"dryRun": true,
			"staged": false,
		}), nil
	}

	args := []string{"add"}
	if config.Paths != nil {
		args = append(args, config.Paths...)
	} else {
		args = append(args, ".")
	}
	cmd := e.buildGitCmd(config, args...)
	_, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git add failed: %w", err)
	}

	return result(true, map[string]interface{}{"staged": true}), nil
}

func (e *Executor) commit(config *domain.GitResourceConfig) (interface{}, error) {
	if config.Message == "" {
		return nil, errors.New("git: commit message is required")
	}
	if config.DryRun {
		return result(true, map[string]interface{}{
			"dryRun":    true,
			"message":   config.Message,
			"committed": false,
		}), nil
	}

	cmd := e.buildGitCmd(config, "commit", "-m", config.Message)
	_, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{
			"error":   stderr,
			"message": config.Message,
		}), fmt.Errorf("git commit failed: %w", err)
	}

	return result(true, map[string]interface{}{
		"message":   config.Message,
		"committed": true,
	}), nil
}

func (e *Executor) checkout(config *domain.GitResourceConfig) (interface{}, error) {
	if config.DryRun {
		return result(true, map[string]interface{}{
			"dryRun":   true,
			"branch":   config.Branch,
			"paths":    config.Paths,
			"switched": false,
		}), nil
	}

	args := []string{"checkout"}
	if config.Branch != "" {
		args = append(args, config.Branch)
	}
	if config.Paths != nil {
		args = append(args, "--")
		args = append(args, config.Paths...)
	}
	cmd := e.buildGitCmd(config, args...)
	_, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git checkout failed: %w", err)
	}

	return result(true, map[string]interface{}{
		"branch":   config.Branch,
		"paths":    config.Paths,
		"switched": true,
	}), nil
}

func (e *Executor) init(config *domain.GitResourceConfig) (interface{}, error) {
	if config.DryRun {
		return result(true, map[string]interface{}{
			"dryRun":      true,
			"initialized": false,
		}), nil
	}

	cmd := e.buildGitCmd(config, "init")
	_, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git init failed: %w", err)
	}

	path := config.WorkingDir
	if path == "" {
		path, _ = os.Getwd()
	}

	return result(true, map[string]interface{}{
		"path":        path,
		"initialized": true,
	}), nil
}

func (e *Executor) cloneOp(config *domain.GitResourceConfig) (interface{}, error) {
	if config.URL == "" {
		return nil, errors.New("git: URL is required for clone operation")
	}
	if config.DryRun {
		return result(true, map[string]interface{}{
			"dryRun": true,
			"url":    config.URL,
			"cloned": false,
		}), nil
	}

	args := []string{"clone", config.URL}
	if config.WorkingDir != "" {
		args = append(args, config.WorkingDir)
	}
	cmd := e.buildGitCmd(config, args...)
	_, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git clone failed: %w", err)
	}

	return result(true, map[string]interface{}{
		"url":    config.URL,
		"cloned": true,
	}), nil
}

func (e *Executor) push(config *domain.GitResourceConfig) (interface{}, error) {
	if config.DryRun {
		return result(true, map[string]interface{}{
			"dryRun": true,
			"remote": remoteOrDefault(config.Remote),
			"branch": config.Branch,
			"pushed": false,
		}), nil
	}

	args := []string{"push", remoteOrDefault(config.Remote)}
	if config.Branch != "" {
		args = append(args, config.Branch)
	}
	cmd := e.buildGitCmd(config, args...)
	_, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git push failed: %w", err)
	}

	return result(true, map[string]interface{}{
		"remote": remoteOrDefault(config.Remote),
		"branch": config.Branch,
		"pushed": true,
	}), nil
}

func (e *Executor) pull(config *domain.GitResourceConfig) (interface{}, error) {
	if config.DryRun {
		return result(true, map[string]interface{}{
			"dryRun": true,
			"remote": remoteOrDefault(config.Remote),
			"branch": config.Branch,
			"pulled": false,
		}), nil
	}

	args := []string{"pull", remoteOrDefault(config.Remote)}
	if config.Branch != "" {
		args = append(args, config.Branch)
	}
	cmd := e.buildGitCmd(config, args...)
	_, stderr, exitCode, err := e.runner.Run(cmd)
	if err != nil || exitCode != 0 {
		return result(false, map[string]interface{}{"error": stderr}), fmt.Errorf("git pull failed: %w", err)
	}

	return result(true, map[string]interface{}{
		"remote": remoteOrDefault(config.Remote),
		"branch": config.Branch,
		"pulled": true,
	}), nil
}

func remoteOrDefault(r string) string {
	if r == "" {
		return defaultRemote
	}
	return r
}
