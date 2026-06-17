//go:build !js

package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// mockRunner captures git commands and returns canned output.
type mockRunner struct {
	entries map[string]mockEntry
}

type mockEntry struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
}

func (r *mockRunner) Run(cmd *exec.Cmd) (string, string, int, error) {
	key := strings.Join(cmd.Args[1:], " ")
	if e, ok := r.entries[key]; ok {
		return e.stdout, e.stderr, e.exitCode, e.err
	}
	return "", "", 0, fmt.Errorf("unexpected command: %s", key)
}

func newTestExecutor(runner CommandRunner) *Executor {
	return NewExecutorWithRunner(runner)
}

// --- Helper to create a real temp git repo ---

func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	}
	for _, args := range cmds {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	return dir
}

func addFileToRepo(t *testing.T, repoDir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repoDir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

// --- Tests ---

func TestExecute_RequiresOperation(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{})
	if err == nil || !strings.Contains(err.Error(), "operation is required") {
		t.Fatalf("expected operation required error, got: %v", err)
	}
}

func TestExecute_UnsupportedOperation(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{Operation: "invalid"})
	if err == nil || !strings.Contains(err.Error(), `unsupported operation "invalid"`) {
		t.Fatalf("expected unsupported error, got: %v", err)
	}
}

func TestStatus_Success(t *testing.T) {
	repo := initTestRepo(t)
	addFileToRepo(t, repo, "new.txt", "hello")

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpStatus,
		WorkingDir: repo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m["branch"] != "main" && m["branch"] != "master" {
		t.Fatalf("expected branch 'main' or 'master', got %q", m["branch"])
	}
	untracked := m["untracked"].([]string)
	if len(untracked) != 1 {
		t.Fatalf("expected 1 untracked file, got %d: %v", len(untracked), untracked)
	}
}

func TestStatus_Clean(t *testing.T) {
	repo := initTestRepo(t)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpStatus,
		WorkingDir: repo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if len(m["untracked"].([]string)) != 0 {
		t.Fatalf("expected 0 untracked files on clean repo")
	}
}

func TestStatus_WithRunner(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"status --porcelain -b": {
				stdout: "## main...origin/main\n M modified.txt\n?? new.txt\n",
			},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpStatus,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m["branch"] != "main" {
		t.Fatalf("expected branch 'main', got %q", m["branch"])
	}
}

func TestDiff_WithChanges(t *testing.T) {
	repo := initTestRepo(t)
	addFileToRepo(t, repo, "README.md", "# modified\n")

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpDiff,
		WorkingDir: repo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m["diff"].(string) == "" {
		t.Fatal("expected non-empty diff")
	}
}

func TestDiff_Clean(t *testing.T) {
	repo := initTestRepo(t)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpDiff,
		WorkingDir: repo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m["diff"].(string) != "" {
		t.Fatal("expected empty diff on clean repo")
	}
}

func TestLog_WithCommits(t *testing.T) {
	repo := initTestRepo(t)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpLog,
		WorkingDir: repo,
		MaxCount:   5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	commits := m["commits"].([]map[string]interface{})
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0]["message"] != "initial" {
		t.Fatalf("expected commit message 'initial', got %q", commits[0]["message"])
	}
}

func TestLog_DefaultMaxCount(t *testing.T) {
	e := newTestExecutor(&mockRunner{
		entries: map[string]mockEntry{
			"log --oneline --max-count=10 --format=%H|%an|%ae|%ai|%s": {
				stdout: "abc|Alice|a@b.com|2026-01-01|feat\n",
			},
		},
	})
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpLog,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestBranch_WithRunner(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"branch -a": {stdout: "* main\n  feature\n  remotes/origin/main\n"},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpBranch,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	branches := m["branches"].([]map[string]interface{})
	if len(branches) != 3 {
		t.Fatalf("expected 3 branches, got %d", len(branches))
	}
	if m["current"] != "main" {
		t.Fatalf("expected current 'main', got %q", m["current"])
	}
}

func TestBranch_Real(t *testing.T) {
	repo := initTestRepo(t)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpBranch,
		WorkingDir: repo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	branches := m["branches"].([]map[string]interface{})
	if len(branches) < 1 {
		t.Fatal("expected at least 1 branch")
	}
}

func TestRemote_WithRunner(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"remote -v": {
				stdout: "origin\thttps://github.com/user/repo.git (fetch)\norigin\thttps://github.com/user/repo.git (push)\n",
			},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpRemote,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m["count"].(int) != 1 {
		t.Fatalf("expected 1 remote, got %d", m["count"])
	}
}

func TestRemote_Empty(t *testing.T) {
	repo := initTestRepo(t)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpRemote,
		WorkingDir: repo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m["count"].(int) != 0 {
		t.Fatalf("expected 0 remotes for fresh repo, got %d", m["count"])
	}
}

func TestAdd_StagesFiles(t *testing.T) {
	repo := initTestRepo(t)
	addFileToRepo(t, repo, "added.txt", "content")

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpAdd,
		WorkingDir: repo,
		Paths:      []string{"added.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}

	// Verify file is now staged
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repo
	out, _ := cmd.Output()
	if !strings.Contains(string(out), "A  added.txt") {
		t.Fatalf("expected staged file in status, got: %s", out)
	}
}

func TestAdd_DryRun(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpAdd,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["dryRun"] != true {
		t.Fatal("expected dryRun true")
	}
}

func TestCommit_CreatesCommit(t *testing.T) {
	repo := initTestRepo(t)
	addFileToRepo(t, repo, "commit.txt", "new file")

	// Stage first
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpCommit,
		WorkingDir: repo,
		Message:    "test commit",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m["committed"] != true {
		t.Fatal("expected committed true")
	}

	// Verify commit exists
	cmd = exec.Command("git", "log", "--oneline", "--max-count=1", "--format=%s")
	cmd.Dir = repo
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) != "test commit" {
		t.Fatalf("expected commit message 'test commit', got %q", strings.TrimSpace(string(out)))
	}
}

func TestCommit_RequiresMessage(t *testing.T) {
	e := NewExecutor()
	_, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpCommit,
	})
	if err == nil || !strings.Contains(err.Error(), "commit message is required") {
		t.Fatalf("expected message required error, got: %v", err)
	}
}

func TestCommit_DryRun(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpCommit,
		Message:   "test",
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["dryRun"] != true {
		t.Fatal("expected dryRun true")
	}
}

func TestCheckout_SwitchesBranch(t *testing.T) {
	repo := initTestRepo(t)

	// Create a branch first
	cmd := exec.Command("git", "branch", "feature")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch: %v\n%s", err, out)
	}

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpCheckout,
		WorkingDir: repo,
		Branch:     "feature",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}

	// Verify branch switch
	cmd = exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repo
	out, _ := cmd.Output()
	if strings.TrimSpace(string(out)) != "feature" {
		t.Fatalf("expected branch 'feature', got %q", strings.TrimSpace(string(out)))
	}
}

func TestCheckout_DryRun(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpCheckout,
		Branch:    "other",
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["dryRun"] != true {
		t.Fatal("expected dryRun true")
	}
}

func TestInit_CreatesRepo(t *testing.T) {
	dir := t.TempDir()

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpInit,
		WorkingDir: dir,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if m["initialized"] != true {
		t.Fatal("expected initialized true")
	}

	if _, statErr := os.Stat(filepath.Join(dir, ".git")); os.IsNotExist(statErr) {
		t.Fatal("expected .git directory to exist")
	}
}

func TestInit_DryRun(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpInit,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["dryRun"] != true {
		t.Fatal("expected dryRun true")
	}
}

func TestAdd_StagesAllWhenNoPaths(t *testing.T) {
	repo := initTestRepo(t)
	addFileToRepo(t, repo, "all.txt", "all")

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpAdd,
		WorkingDir: repo,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestShow_WithRunner(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"show --no-color --format=fuller": {stdout: "commit abc123\nAuthor: Test\n"},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpShow,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	if !strings.Contains(m["output"].(string), "commit") {
		t.Fatal("expected output to contain 'commit'")
	}
}

func TestShow_Real(t *testing.T) {
	repo := initTestRepo(t)

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpShow,
		WorkingDir: repo,
		Args:       []string{"HEAD"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestPull_DryRun(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpPull,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["dryRun"] != true {
		t.Fatal("expected dryRun true")
	}
}

func TestPush_DryRun(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpPush,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["dryRun"] != true {
		t.Fatal("expected dryRun true")
	}
}

func TestClone_RequiresURL(t *testing.T) {
	e := NewExecutor()
	_, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpClone,
	})
	if err == nil || !strings.Contains(err.Error(), "URL is required") {
		t.Fatalf("expected URL required error, got: %v", err)
	}
}

func TestClone_DryRun(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpClone,
		URL:       "https://example.com/repo.git",
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["dryRun"] != true {
		t.Fatal("expected dryRun true")
	}
}

func TestDefaultRunner_Run(t *testing.T) {
	r := &DefaultCommandRunner{}
	stdout, stderr, _, err := r.Run(exec.Command("echo", "hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = stdout
	_ = stderr
}

func TestDefaultRunner_Run_ExitCode(t *testing.T) {
	r := &DefaultCommandRunner{}
	_, _, exitCode, err := r.Run(exec.Command("sh", "-c", "exit 2"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 2 {
		t.Fatalf("expected exitCode 2, got %d", exitCode)
	}
}

func TestDefaultRunner_Run_BadBinary(t *testing.T) {
	r := &DefaultCommandRunner{}
	_, _, exitCode, err := r.Run(exec.Command("/nonexistent_kdeps_binary_xyz"))
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}
	if exitCode != -1 {
		t.Fatalf("expected exitCode -1, got %d", exitCode)
	}
}

func TestRemoteOrDefault(t *testing.T) {
	if remoteOrDefault("") != defaultRemote {
		t.Fatalf("expected default remote 'origin', got %q", remoteOrDefault(""))
	}
	if remoteOrDefault("upstream") != "upstream" {
		t.Fatalf("expected 'upstream', got %q", remoteOrDefault("upstream"))
	}
}

// --- Error branch tests using empty mockRunner ---

func TestStatus_Error(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpStatus})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDiff_Error(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpDiff})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLog_Error(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpLog})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestShow_Error(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpShow})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBranch_Error(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpBranch})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRemote_Error(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpRemote})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClone_ExecutionError(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpClone,
		URL:       "https://example.com/repo.git",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClone_Success(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"clone https://example.com/repo.git": {stdout: ""},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpClone,
		URL:       "https://example.com/repo.git",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["cloned"] != true {
		t.Fatal("expected cloned true")
	}
}

func TestClone_WithWorkingDir(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"clone https://example.com/repo.git /tmp/dest": {stdout: ""},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation:  domain.GitOpClone,
		URL:        "https://example.com/repo.git",
		WorkingDir: "/tmp/dest",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["cloned"] != true {
		t.Fatal("expected cloned true")
	}
}

func TestPull_ExecutionError(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpPull})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPull_Success(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"pull origin": {stdout: "Already up to date."},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpPull})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["pulled"] != true {
		t.Fatal("expected pulled true")
	}
}

func TestPull_WithBranch(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"pull origin main": {stdout: ""},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpPull,
		Branch:    "main",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["pulled"] != true {
		t.Fatal("expected pulled true")
	}
}

func TestPush_ExecutionError(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpPush})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPush_Success(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"push origin": {stdout: ""},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpPush})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["pushed"] != true {
		t.Fatal("expected pushed true")
	}
}

func TestResult_NilData(t *testing.T) {
	m := result(false, nil)
	if m["success"] != false {
		t.Fatal("expected success false")
	}
}

func TestStatus_ConflictsAndStaged(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"status --porcelain -b": {
				stdout: "## main\nUU conflict.txt\nM  staged.txt\n   \n",
			},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpStatus})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	conflicts := m["conflicts"].([]string)
	if len(conflicts) != 1 || conflicts[0] != "conflict.txt" {
		t.Fatalf("expected 1 conflict, got %v", conflicts)
	}
	staged := m["staged"].([]string)
	if len(staged) != 1 || staged[0] != "staged.txt" {
		t.Fatalf("expected 1 staged, got %v", staged)
	}
}

func TestStatus_ShortLine(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"status --porcelain -b": {
				stdout: "## main\n   \n",
			},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpStatus})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestDiff_WithPaths(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"diff --no-color -- file.txt": {stdout: "diff --git a/file.txt b/file.txt\n+new line\n"},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpDiff,
		Paths:     []string{"file.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestLog_WithPaths(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"log --oneline --max-count=10 --format=%H|%an|%ae|%ai|%s -- main.go": {
				stdout: "abc123|Author|a@b.com|2025-01-01|fix bug\n",
			},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpLog,
		Paths:     []string{"main.go"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestBranch_ShortLine(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"branch -a": {stdout: "* main\n \n"},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{Operation: domain.GitOpBranch})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["success"] != true {
		t.Fatal("expected success true")
	}
}

func TestAdd_ExecutionError(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpAdd,
		Paths:     []string{"file.txt"},
	})
	if err == nil {
		t.Fatal("expected error for add failure")
	}
}

func TestCommit_ExecutionError(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpCommit,
		Message:   "test",
	})
	if err == nil {
		t.Fatal("expected error for commit failure")
	}
}

func TestCheckout_WithPaths(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"checkout -- file.txt": {stdout: ""},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpCheckout,
		Paths:     []string{"file.txt"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["switched"] != true {
		t.Fatal("expected switched true")
	}
}

func TestCheckout_ExecutionError(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpCheckout,
		Branch:    "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for checkout failure")
	}
}

func TestInit_ExecutionError(t *testing.T) {
	e := newTestExecutor(&mockRunner{})
	_, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpInit,
	})
	if err == nil {
		t.Fatal("expected error for init failure")
	}
}

func TestInit_UsesCwd(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]mockEntry{
			"init": {stdout: ""},
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.GitResourceConfig{
		Operation: domain.GitOpInit,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.(map[string]interface{})["initialized"] != true {
		t.Fatal("expected initialized true")
	}
}
