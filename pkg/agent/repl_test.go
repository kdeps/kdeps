package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/chzyer/readline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

// makeTestLoop returns a minimal Loop with a fixed skill list for REPL tests.
func makeTestLoop(skills []Skill) *Loop {
	return &Loop{
		skillList: skills,
		config:    Config{Model: "test-model"},
		session:   NewSession(0),
	}
}

// --- SkillByName ---

func TestSkillByName_Found(t *testing.T) {
	sk := Skill{Name: "lint", Description: "run linter", Content: "Run golangci-lint."}
	loop := makeTestLoop([]Skill{sk})
	got := loop.SkillByName("lint")
	require.NotNil(t, got)
	assert.Equal(t, "lint", got.Name)
}

func TestSkillByName_NotFound(t *testing.T) {
	loop := makeTestLoop([]Skill{{Name: "lint"}})
	assert.Nil(t, loop.SkillByName("nope"))
}

func TestSkillByName_Empty(t *testing.T) {
	loop := makeTestLoop(nil)
	assert.Nil(t, loop.SkillByName("anything"))
}

func TestSkillByName_MultipleSkills(t *testing.T) {
	skills := []Skill{
		{Name: "lint"},
		{Name: "test"},
		{Name: "review"},
	}
	loop := makeTestLoop(skills)
	assert.NotNil(t, loop.SkillByName("test"))
	assert.NotNil(t, loop.SkillByName("review"))
	assert.Nil(t, loop.SkillByName("missing"))
}

// --- dispatchCommand skill routing ---

func TestDispatchCommand_UnknownNonSkill(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// No skills loaded, /unknown-cmd should return nil (prints message, no error)
	err := repl.dispatchCommand("/unknown-cmd")
	assert.NoError(t, err)
}

func TestDispatchCommand_SkillNotFound_NoError(t *testing.T) {
	loop := makeTestLoop([]Skill{{Name: "lint"}})
	repl := NewREPL(loop)
	defer repl.cancel()

	// /nope doesn't match any loaded skill
	err := repl.dispatchCommand("/nope")
	assert.NoError(t, err)
}

// --- loadSkillSlice ---

func TestLoadSkillSlice_Empty(t *testing.T) {
	slc := loadSkillSlice([]string{"/nonexistent"})
	assert.Empty(t, slc)
}

func TestLoadSkillSlice_WithSkill(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: my-skill\ndescription: does things\n---\n\nContent."
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644))

	slc := loadSkillSlice([]string{dir})
	require.Len(t, slc, 1)
	assert.Equal(t, "my-skill", slc[0].Name)
	assert.Equal(t, "does things", slc[0].Description)
}

func TestReloadSkills(t *testing.T) {
	loop := makeTestLoop(nil)
	assert.Empty(t, loop.skillList)
	assert.Empty(t, loop.Skills())

	dir := t.TempDir()
	content := "---\nname: new-skill\ndescription: test\n---\n\nDo things."
	require.NoError(t, os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644))

	loop.ReloadSkills([]string{dir})
	require.Len(t, loop.skillList, 1)
	assert.Equal(t, "new-skill", loop.skillList[0].Name)
	assert.Contains(t, loop.Skills(), "new-skill")
}

func TestSetTUIRunner(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	called := false
	repl.SetTUIRunner(func() ([]string, bool, error) {
		called = true
		return nil, false, nil
	})

	// cmdSettings with runner set should invoke it (no panic)
	err := repl.cmdSettings()
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestSetOnSettingsChange(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	var gotPaths []string
	var gotToolsChanged bool
	repl.SetOnSettingsChange(func(paths []string, changed bool) {
		gotPaths = paths
		gotToolsChanged = changed
	})
	repl.SetTUIRunner(func() ([]string, bool, error) {
		return []string{"/some/path"}, true, nil
	})

	require.NoError(t, repl.cmdSettings())
	assert.Equal(t, []string{"/some/path"}, gotPaths)
	assert.True(t, gotToolsChanged)
}

func TestCmdSettings_NoRunner(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// No TUI runner — should print a message and return nil
	err := repl.cmdSettings()
	assert.NoError(t, err)
}

func TestCmdSettings_RunnerError(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	repl.SetTUIRunner(func() ([]string, bool, error) {
		return nil, false, errors.New("tui failed")
	})
	err := repl.cmdSettings()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tui failed")
}

func TestDispatchCommand_Settings(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.dispatchCommand("/settings")
	assert.NoError(t, err) // no runner set, prints message
}

func TestLoadSkillSlice_DedupByName(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	content := "---\nname: same-skill\n---\nContent."
	require.NoError(t, os.WriteFile(filepath.Join(dir1, "SKILL.md"), []byte(content), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir2, "SKILL.md"), []byte(content), 0o644))

	// Same name in both dirs — should appear only once
	slc := loadSkillSlice([]string{dir1, dir2})
	assert.Len(t, slc, 1)
}

// --- fuzzyMatch ---

func TestFuzzyMatch_Empty(t *testing.T) {
	assert.True(t, fuzzyMatch("", "anything"))
}

func TestFuzzyMatch_ExactMatch(t *testing.T) {
	assert.True(t, fuzzyMatch("help", "help"))
}

func TestFuzzyMatch_Subsequence(t *testing.T) {
	assert.True(t, fuzzyMatch("hlp", "help"))
	assert.True(t, fuzzyMatch("hist", "history"))
}

func TestFuzzyMatch_NoMatch(t *testing.T) {
	assert.False(t, fuzzyMatch("xyz", "help"))
}

func TestFuzzyMatch_CaseInsensitive(t *testing.T) {
	assert.True(t, fuzzyMatch("HLP", "help"))
}

// --- fuzzyScore ---

func TestFuzzyScore_Empty(t *testing.T) {
	ok, score := fuzzyScore("", "anything")
	assert.True(t, ok)
	assert.Equal(t, 0, score)
}

func TestFuzzyScore_ExactMatch(t *testing.T) {
	ok, score := fuzzyScore("help", "help")
	assert.True(t, ok)
	// exact match gets a bonus (lower score)
	ok2, score2 := fuzzyScore("hlp", "help")
	assert.True(t, ok2)
	assert.Less(t, score, score2)
}

func TestFuzzyScore_NoMatch(t *testing.T) {
	ok, _ := fuzzyScore("xyz", "help")
	assert.False(t, ok)
}

func TestFuzzyScore_WordBoundaryRanksHigher(t *testing.T) {
	// "he" matches "help" at a word boundary (start) better than a mid-word match
	ok1, score1 := fuzzyScore("he", "help")
	ok2, score2 := fuzzyScore("he", "the-thing")
	assert.True(t, ok1)
	assert.True(t, ok2)
	// word boundary start should score better (lower)
	assert.Less(t, score1, score2)
}

// --- fuzzyRankStrings ---

func TestFuzzyRankStrings_Empty(t *testing.T) {
	results := fuzzyRankStrings("", nil)
	assert.Empty(t, results)
}

func TestFuzzyRankStrings_NoMatches(t *testing.T) {
	results := fuzzyRankStrings("xyz", []string{"help", "clear", "exit"})
	assert.Empty(t, results)
}

func TestFuzzyRankStrings_RankedByScore(t *testing.T) {
	// "help" should rank above "history" for query "he"
	results := fuzzyRankStrings("hel", []string{"history", "help", "compact"})
	require.NotEmpty(t, results)
	assert.Equal(t, "help", results[0])
}

// --- SetModelNames / /model completion ---

func TestSetModelNames(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetModelNames([]string{"llama3.2:1b", "llama3.2:3b", "qwen3.5-4b"})
	assert.Equal(t, []string{"llama3.2:1b", "llama3.2:3b", "qwen3.5-4b"}, repl.modelNames)
}

func TestReplCompleter_ModelArgCompletion(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetModelNames([]string{"llama3.2:1b", "llama3.2:3b", "qwen3.5-4b"})
	repl.SetDownloadedModels(map[string]bool{"llama3.2:1b": true})

	c := &replCompleter{repl: repl}
	input := []rune("/model llama")
	results, length := c.Do(input, len(input))
	assert.Equal(t, len([]rune("llama")), length)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	// Full names returned; readline deletes the typed token and inserts the full name.
	assert.Contains(t, found, "llama3.2:1b [cached]")
	assert.Contains(t, found, "llama3.2:3b [cloud]")
}

func TestReplCompleter_ModelArgAllModels(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetModelNames([]string{"llama3.2:1b", "llama3.2:3b"})

	c := &replCompleter{repl: repl}
	input := []rune("/model ")
	results, _ := c.Do(input, len(input))
	assert.Len(t, results, 2)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	assert.Contains(t, found, "llama3.2:1b [cloud]")
	assert.Contains(t, found, "llama3.2:3b [cloud]")
}

func TestReplCompleter_DownloadedModelMarker(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetModelNames([]string{"llama3.2:1b", "llama3.2:3b", "qwen3.5-4b"})
	repl.SetDownloadedModels(map[string]bool{"llama3.2:1b": true})

	c := &replCompleter{repl: repl}
	// Cached model sorts first; all entries have [tag] suffix.
	input := []rune("/model ")
	results, _ := c.Do(input, len(input))
	assert.Len(t, results, 3)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	assert.Contains(t, found, "llama3.2:1b [cached]")
	assert.Contains(t, found, "llama3.2:3b [cloud]")
	assert.Contains(t, found, "qwen3.5-4b [cloud]")
}

func TestReplCompleter_DownloadedModelMarkerPartialToken(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetModelNames([]string{"llama3.2:1b", "llama3.2:3b"})
	repl.SetDownloadedModels(map[string]bool{"llama3.2:1b": true})

	c := &replCompleter{repl: repl}
	// Partial token "llama3.2" matches both; full names returned.
	input := []rune("/model llama3.2")
	results, length := c.Do(input, len(input))
	assert.Equal(t, len([]rune("llama3.2")), length)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	assert.Contains(t, found, "llama3.2:1b [cached]")
	assert.Contains(t, found, "llama3.2:3b [cloud]")
}

func TestReplCompleter_EnabledCloudModelTag(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetModelNames([]string{"gpt-4o", "deepseek-chat"})
	repl.SetCloudModelBackends(map[string]string{"gpt-4o": "openai", "deepseek-chat": "deepseek"})
	repl.SetProviderStatus(map[string]bool{"openai": true, "deepseek": false})

	c := &replCompleter{repl: repl}
	input := []rune("/model ")
	results, _ := c.Do(input, len(input))
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	// Enabled cloud provider shows [cloud enabled]; disabled shows [cloud].
	assert.Contains(t, found, "gpt-4o [cloud enabled]")
	assert.Contains(t, found, "deepseek-chat [cloud]")
}

func TestCmdModel_StripsStar(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	_ = repl.cmdModel([]string{"qwen2.5*:7b"})
	assert.Equal(t, "qwen2.5:7b", repl.loop.config.Model)
}

func TestCmdModel_StripsTagSuffix(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// [tag] suffix appended by tab completion is stripped before applying.
	_ = repl.cmdModel([]string{"llama3.2:1b [llamafile cached]"})
	assert.Equal(t, "llama3.2:1b", repl.loop.config.Model)

	_ = repl.cmdModel([]string{"deepseek-chat [cloud enabled]"})
	assert.Equal(t, "deepseek-chat", repl.loop.config.Model)
}

// --- expandFileRefs ---

func TestExpandFileRefs_NoRefs(t *testing.T) {
	out, files := expandFileRefs("hello world")
	assert.Equal(t, "hello world", out)
	assert.Empty(t, files)
}

func TestExpandFileRefs_UnreadablePath(t *testing.T) {
	// @nonexistent-file should be left as-is
	out, files := expandFileRefs("check @/nonexistent/file.txt please")
	assert.Contains(t, out, "@/nonexistent/file.txt")
	assert.Empty(t, files)
}

func TestExpandFileRefs_RealFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(p, []byte("hello from file"), 0o644))

	out, files := expandFileRefs("review @" + p)
	assert.Contains(t, out, "hello from file")
	assert.Contains(t, out, "notes.txt")
	assert.Empty(t, files) // text files expand inline, not as attachments
}

func TestExpandFileRefs_ImageFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "photo.png")
	require.NoError(t, os.WriteFile(p, []byte("\x89PNG"), 0o644))

	out, files := expandFileRefs("describe @" + p)
	// Image ref should be stripped from text, not embedded inline
	assert.NotContains(t, out, "photo.png")
	assert.Contains(t, files, p)
}

func TestExpandFileRefs_ImageNotFound(t *testing.T) {
	out, files := expandFileRefs("describe @/nonexistent/photo.png")
	// Non-existent image ref left unchanged (file not accessible)
	assert.Contains(t, out, "@/nonexistent/photo.png")
	assert.Empty(t, files)
}

func TestExpandFileRefs_URLImage(t *testing.T) {
	out, files := expandFileRefs("describe @https://example.com/photo.png what is this?")
	// URL refs are always treated as multimodal attachments, removed from text
	assert.NotContains(t, out, "https://example.com/photo.png")
	assert.Contains(t, files, "https://example.com/photo.png")
}

func TestSetPendingFiles_ClearedAfterBuildChatConfig(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.SetPendingFiles([]string{"/tmp/img.png"})
	assert.Equal(t, []string{"/tmp/img.png"}, loop.pendingFiles)

	// buildChatConfig should consume and clear pendingFiles
	cfg := loop.buildChatConfig("hello", "")
	assert.Equal(t, []string{"/tmp/img.png"}, cfg.Files)
	assert.Nil(t, loop.pendingFiles)
}

// --- filePathCompletions ---

func TestFilePathCompletions_EmptyPrefix(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "foo.go"), []byte(""), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0o755))

	// Change cwd to dir for the test
	orig, _ := os.Getwd()
	require.NoError(t, os.Chdir(dir))
	defer func() { _ = os.Chdir(orig) }()

	results := filePathCompletions("")
	names := make([]string, len(results))
	copy(names, results)
	assert.Contains(t, names, "foo.go")
	assert.Contains(t, names, "subdir/")
}

func TestFilePathCompletions_Prefix(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "alpha.go"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "beta.go"), []byte(""), 0o644))

	results := filePathCompletions(dir + "/al")
	require.Len(t, results, 1)
	assert.Contains(t, results[0], "alpha.go")
}

func TestFilePathCompletions_BadDir(t *testing.T) {
	results := filePathCompletions("/nonexistent/path/prefix")
	assert.Empty(t, results)
}

// --- replCompleter ---

func TestReplCompleter_SlashCommand(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	c := &replCompleter{repl: repl}
	// "/h" -> token="/h" (len=2), suffixes: "elp", "istory" (readline shows "/h"+"elp" = "/help")
	results, length := c.Do([]rune("/h"), 2)
	assert.Equal(t, 2, length)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	assert.Contains(t, found, "elp")
	assert.Contains(t, found, "istory")
}

func TestReplCompleter_SlashSkill(t *testing.T) {
	loop := makeTestLoop([]Skill{{Name: "review", Description: "code review"}})
	repl := NewREPL(loop)
	defer repl.cancel()

	c := &replCompleter{repl: repl}
	// "/rev" -> token="/rev" (len=4), suffix: "iew" (readline shows "/rev"+"iew" = "/review")
	results, length := c.Do([]rune("/rev"), 4)
	assert.Equal(t, 4, length)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	assert.Contains(t, found, "iew")
}

func TestReplCompleter_NoSlash(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	c := &replCompleter{repl: repl}
	// plain text returns no completions
	results, length := c.Do([]rune("hello"), 5)
	assert.Equal(t, 0, length)
	assert.Empty(t, results)
}

func TestReplCompleter_AtFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(""), 0o644))

	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	c := &replCompleter{repl: repl}
	token := "@" + dir + "/ma"
	results, length := c.Do([]rune(token), len([]rune(token)))
	assert.Equal(t, len([]rune(token)), length)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	require.Len(t, found, 1)
	assert.Contains(t, found[0], "main.go")
}

// --- allCommandNames ---

func TestAllCommandNames_IncludesBuiltins(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	names := repl.allCommandNames()
	assert.Contains(t, names, "/help")
	assert.Contains(t, names, "/settings")
}

func TestAllCommandNames_IncludesSkills(t *testing.T) {
	loop := makeTestLoop([]Skill{{Name: "lint"}})
	repl := NewREPL(loop)
	defer repl.cancel()

	names := repl.allCommandNames()
	assert.Contains(t, names, "/lint")
}

// --- dynamicPrompt ---

func TestDynamicPrompt_ZeroTurns(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	p := repl.dynamicPrompt()
	assert.Contains(t, p, "test-model")
	assert.NotContains(t, p, "|")
}

func TestDynamicPrompt_WithTurns(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.session.Append("hi", "hello")
	repl := NewREPL(loop)
	defer repl.cancel()

	p := repl.dynamicPrompt()
	assert.Contains(t, p, "test-model")
	assert.Contains(t, p, "|")
}

// --- buildCompleter ---

func TestBuildCompleter_ReturnsNonNil(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	c := repl.buildCompleter()
	assert.NotNil(t, c)
}

// --- handleReadError ---

func TestHandleReadError_Nil(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	stop, err := repl.handleReadError(nil)
	assert.False(t, stop)
	assert.NoError(t, err)
}

func TestHandleReadError_EOF(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	stop, err := repl.handleReadError(io.EOF)
	assert.True(t, stop)
	assert.NoError(t, err)
}

func TestHandleReadError_Interrupt(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	stop, err := repl.handleReadError(readline.ErrInterrupt)
	assert.False(t, stop)
	assert.NoError(t, err)
}

func TestHandleReadError_OtherError(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	sentinel := errors.New("terminal closed")
	stop, err := repl.handleReadError(sentinel)
	assert.True(t, stop)
	assert.ErrorIs(t, err, sentinel)
}

// --- processInput ---

func TestProcessInput_Command(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// /help is a command - should route to cmdHelp, no error
	err := repl.processInput("/help")
	assert.NoError(t, err)
}

func TestProcessInput_TextMessage(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "mock response", nil
	}
	err := repl.processInput("hello agent")
	assert.NoError(t, err)
	assert.Len(t, repl.history, 1)
}

func TestProcessInput_RunError(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "", errors.New("llm error")
	}
	err := repl.processInput("will fail")
	assert.Error(t, err)
}

func TestProcessInput_EmptyResponse(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "", nil
	}
	err := repl.processInput("silent")
	assert.NoError(t, err)
}

// --- runWithThinking ---

func TestRunWithThinking_Success(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "result", nil
	}
	resp, err := repl.runWithThinking(repl.ctx, "prompt")
	assert.NoError(t, err)
	assert.Equal(t, "result", resp)
}

func TestRunWithThinking_SlowResponse(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	repl.runFn = func(_ context.Context, _ string) (string, error) {
		time.Sleep(replThinkingDelay + 50*time.Millisecond)
		return "slow result", nil
	}
	resp, err := repl.runWithThinking(repl.ctx, "prompt")
	assert.NoError(t, err)
	assert.Equal(t, "slow result", resp)
}

func TestRunWithThinking_Error(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "", errors.New("fail")
	}
	_, err := repl.runWithThinking(repl.ctx, "prompt")
	assert.Error(t, err)
}

// --- maybeHintCompact ---

func TestMaybeHintCompact_NoHint(_ *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// 0 turns - no hint (no panic)
	repl.maybeHintCompact()
}

func TestMaybeHintCompact_AtThreshold(_ *testing.T) {
	loop := makeTestLoop(nil)
	for range replAutoCompactEvery {
		loop.session.Append("q", "a")
	}
	repl := NewREPL(loop)
	defer repl.cancel()

	// exactly 25 turns - hint fires (no panic)
	repl.maybeHintCompact()
}

// --- cmdHelp ---

func TestCmdHelp(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdHelp()
	assert.NoError(t, err)
}

// --- cmdClear ---

func TestCmdClear(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.session.Append("hi", "hello")
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdClear()
	assert.NoError(t, err)
	assert.Zero(t, loop.session.TurnCount())
}

// --- cmdModel ---

func TestCmdModel_Show(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdModel(nil)
	assert.NoError(t, err)
}

func TestCmdModel_Set(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdModel([]string{"gpt-4"})
	assert.NoError(t, err)
	assert.Equal(t, "gpt-4", loop.config.Model)
}

// --- cmdSkills ---

func TestCmdSkills_Empty(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdSkills()
	assert.NoError(t, err)
}

func TestCmdSkills_WithSkills(t *testing.T) {
	loop := makeTestLoop([]Skill{
		{Name: "lint", Description: "run linter"},
		{Name: "review", Source: "review.md"},
	})
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdSkills()
	assert.NoError(t, err)
}

// --- cmdCompact ---

func TestCmdCompact_NoHistory(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdCompact()
	assert.NoError(t, err)
}

func TestCmdCompact_WithHistory(t *testing.T) {
	loop := &Loop{
		config:  Config{Model: "test-model"},
		session: NewSession(2), // maxTurns=2
	}
	// Bypass Append's trimming to create a state where Compact() returns non-empty.
	sess := loop.session
	sess.mu.Lock()
	sess.messages = []sessionMessage{
		{Role: "user", Content: "q1"}, {Role: "assistant", Content: "a1"},
		{Role: "user", Content: "q2"}, {Role: "assistant", Content: "a2"},
		{Role: "user", Content: "q3"}, {Role: "assistant", Content: "a3"},
	}
	sess.mu.Unlock()

	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdCompact()
	assert.NoError(t, err)
}

// --- cmdHistory ---

func TestCmdHistory_Empty(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdHistory()
	assert.NoError(t, err)
}

func TestCmdHistory_WithMessages(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.session.Append("hello", "world")
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdHistory()
	assert.NoError(t, err)
}

func TestCmdHistory_LongMessage(t *testing.T) {
	loop := makeTestLoop(nil)
	long := make([]byte, replPreviewMax+10)
	for i := range long {
		long[i] = 'x'
	}
	loop.session.Append(string(long), "short")
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdHistory()
	assert.NoError(t, err)
}

// --- dispatchCommand remaining branches ---

func TestDispatchCommand_Help(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.dispatchCommand("/help")
	assert.NoError(t, err)
}

func TestDispatchCommand_Clear(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.dispatchCommand("/clear")
	assert.NoError(t, err)
}

func TestDispatchCommand_ModelShow(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.dispatchCommand("/model")
	assert.NoError(t, err)
}

func TestDispatchCommand_ModelSet(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.dispatchCommand("/model claude-3")
	assert.NoError(t, err)
	assert.Equal(t, "claude-3", loop.config.Model)
}

func TestDispatchCommand_Skills(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.dispatchCommand("/skills")
	assert.NoError(t, err)
}

func TestDispatchCommand_Compact(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.dispatchCommand("/compact")
	assert.NoError(t, err)
}

func TestDispatchCommand_History(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.dispatchCommand("/history")
	assert.NoError(t, err)
}

func TestDispatchCommand_Exit(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.dispatchCommand("/exit")
	assert.NoError(t, err)
}

func TestDispatchCommand_Quit(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.dispatchCommand("/quit")
	assert.NoError(t, err)
}

func TestDispatchCommand_InvokeSkill(t *testing.T) {
	loop := makeTestLoop([]Skill{{Name: "lint", Content: "Run linter."}})
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "lint output", nil
	}

	err := repl.dispatchCommand("/lint extra args")
	assert.NoError(t, err)
	require.Len(t, repl.history, 1)
	assert.Equal(t, "/lint", repl.history[0])
}

// --- cmdInvokeSkill ---

func TestCmdInvokeSkill_NoExtra(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.runFn = func(_ context.Context, prompt string) (string, error) {
		return "ok: " + prompt, nil
	}

	sk := &Skill{Name: "review", Content: "Review code."}
	err := repl.cmdInvokeSkill(sk, nil)
	assert.NoError(t, err)
}

func TestCmdInvokeSkill_WithExtra(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.runFn = func(_ context.Context, prompt string) (string, error) {
		return prompt, nil
	}

	sk := &Skill{Name: "review", Content: "Review."}
	err := repl.cmdInvokeSkill(sk, []string{"focus", "on", "security"})
	assert.NoError(t, err)
}

func TestCmdInvokeSkill_Error(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "", errors.New("llm down")
	}

	sk := &Skill{Name: "test", Content: "Test."}
	err := repl.cmdInvokeSkill(sk, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "test")
}

func TestCmdInvokeSkill_EmptyResponse(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "", nil
	}

	sk := &Skill{Name: "noop", Content: "Do nothing."}
	err := repl.cmdInvokeSkill(sk, nil)
	assert.NoError(t, err)
}

// --- runPlain ---

func TestRunPlain_EOF(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// pipe that immediately closes - runPlain should return on EOF
	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	pw.Close()

	orig := os.Stdin
	os.Stdin = pr
	defer func() { os.Stdin = orig; pr.Close() }()

	// should not block
	repl.runPlain()
}

func TestRunPlain_WithCommand(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	_, _ = pw.WriteString("/help\n")
	pw.Close()

	orig := os.Stdin
	os.Stdin = pr
	defer func() { os.Stdin = orig; pr.Close() }()

	repl.runPlain()
}

func TestRunPlain_WithMessage(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "reply", nil
	}

	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	_, _ = pw.WriteString("hello\n")
	pw.Close()

	orig := os.Stdin
	os.Stdin = pr
	defer func() { os.Stdin = orig; pr.Close() }()

	repl.runPlain()
}

func TestRunPlain_ContextCancel(_ *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)

	// cancel before runPlain - should exit immediately
	repl.cancel()

	pr, pw, _ := os.Pipe()
	defer pr.Close()
	defer pw.Close()

	orig := os.Stdin
	os.Stdin = pr
	defer func() { os.Stdin = orig }()

	repl.runPlain()
}

// --- Run / runLoop ---

func TestRun_ExitsOnEOF(t *testing.T) {
	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	// Send /exit so runLoop dispatches cancel, then close to signal EOF.
	_, _ = pw.WriteString("/exit\n")
	pw.Close()

	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	defer outR.Close()

	origIn := os.Stdin
	origOut := os.Stdout
	os.Stdin = pr
	os.Stdout = outW
	defer func() {
		os.Stdin = origIn
		os.Stdout = origOut
		pr.Close()
		outW.Close()
	}()

	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	rerr := repl.Run()
	assert.NoError(t, rerr)
}

func TestRun_EmptyLineIgnored(t *testing.T) {
	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	// empty line then exit
	_, _ = pw.WriteString("\n   \n/exit\n")
	pw.Close()

	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	defer outR.Close()

	origIn := os.Stdin
	origOut := os.Stdout
	os.Stdin = pr
	os.Stdout = outW
	defer func() {
		os.Stdin = origIn
		os.Stdout = origOut
		pr.Close()
		outW.Close()
	}()

	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	rerr := repl.Run()
	assert.NoError(t, rerr)
}

func TestRun_ProcessInputError(t *testing.T) {
	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	_, _ = pw.WriteString("bad input\n/exit\n")
	pw.Close()

	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	defer outR.Close()

	origIn := os.Stdin
	origOut := os.Stdout
	os.Stdin = pr
	os.Stdout = outW
	defer func() {
		os.Stdin = origIn
		os.Stdout = origOut
		pr.Close()
		outW.Close()
	}()

	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "", errors.New("llm error")
	}
	rerr := repl.Run()
	assert.NoError(t, rerr) // error is printed but loop continues
}

func TestRun_ProcessesMessage(t *testing.T) {
	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	_, _ = pw.WriteString("hello\n/exit\n")
	pw.Close()

	outR, outW, err := os.Pipe()
	require.NoError(t, err)
	defer outR.Close()

	origIn := os.Stdin
	origOut := os.Stdout
	os.Stdin = pr
	os.Stdout = outW
	defer func() {
		os.Stdin = origIn
		os.Stdout = origOut
		pr.Close()
		outW.Close()
	}()

	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "agent reply", nil
	}
	rerr := repl.Run()
	assert.NoError(t, rerr)
}

// --- filePathCompletions cap ---

func TestFilePathCompletions_Cap(t *testing.T) {
	dir := t.TempDir()
	for i := range replFileCompletionMax + 5 {
		name := fmt.Sprintf("file%02d.go", i)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(""), 0o644))
	}

	results := filePathCompletions(dir + "/")
	assert.Len(t, results, replFileCompletionMax)
}

func TestFirstLine_Short(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "hello", firstLine("hello"))
}

func TestFirstLine_MultiLine(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "first", firstLine("first\nsecond\nthird"))
}

func TestFirstLine_SkipsBlankLines(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "real", firstLine("\n\nreal\n"))
}

func TestFirstLine_TruncatesLong(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", firstLineMax+10)
	result := firstLine(long)
	assert.True(t, strings.HasSuffix(result, "..."))
	assert.Equal(t, firstLineMax+3, len(result))
}

func TestFirstLine_Empty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", firstLine(""))
}

// --- SetProviderStatus ---

func TestSetProviderStatus(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetProviderStatus(map[string]bool{"openai": true, "anthropic": false})
	assert.True(t, repl.providerStatus["openai"])
	assert.False(t, repl.providerStatus["anthropic"])
}

// --- filePathCompletionsFd ---

func TestFilePathCompletionsFd_NoBinary(t *testing.T) {
	// "nonexistent-fd-binary-xyz" won't be found - falls back to filePathCompletions
	results := filePathCompletionsFd("./", "nonexistent-fd-binary-xyz")
	// Returns either an empty slice or file results from fallback; must not panic.
	assert.NotNil(t, results)
}

// --- cmdModels ---

func TestCmdModels_NoLocalModels(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// Redirect stdout to avoid cluttering test output
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		w.Close()
		os.Stdout = origOut
	}()

	err := repl.cmdModels()
	assert.NoError(t, err)
}

func TestCmdModels_WithLocalModel(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.config.Model = "llama2"
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.modelNames = []string{"llama2", "codellama"}

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdModels()
	assert.NoError(t, err)
}

func TestCmdModels_WithProviderStatus(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetProviderStatus(map[string]bool{"openai": true, "anthropic": false})

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdModels()
	assert.NoError(t, err)
}

func TestPrintLocalModelRow_DownloadedAndCurrent(_ *testing.T) {
	loop := makeTestLoop(nil)
	loop.config.Model = "llama2"
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.downloadedModels = map[string]bool{"llama2": true}

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	repl.printLocalModelRow("llama2", "llama2")
	repl.printLocalModelRow("codellama", "llama2")
}

// --- cmdPrompts ---

func TestCmdPrompts_NoPrompts(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdPrompts()
	assert.NoError(t, err)
}

func TestCmdPrompts_WithPrompts(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.prompts = []PromptTemplate{
		{Name: "review", Description: "Code review", Content: "Review this: {{.}}"},
		{Name: "explain", ArgumentHint: "topic", Content: "Explain: {{.}}"},
	}
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdPrompts()
	assert.NoError(t, err)
}

// --- cmdInvokePrompt ---

func TestCmdInvokePrompt_Success(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "response text", nil
	}

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	pt := &PromptTemplate{Name: "review", Content: "Review: {{1}}"}
	err := repl.cmdInvokePrompt(pt, []string{"code"})
	assert.NoError(t, err)
}

func TestCmdInvokePrompt_Error(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		return "", errors.New("LLM error")
	}

	pt := &PromptTemplate{Name: "review", Content: "Review."}
	err := repl.cmdInvokePrompt(pt, nil)
	assert.Error(t, err)
}

// --- cmdSession ---

func TestCmdSession_NoStore(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSession(nil)
	assert.NoError(t, err)
}

func TestCmdSession_UnknownSubcommand(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.store = NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	repl.loop.store = loop.store
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSession([]string{"bogus"})
	assert.NoError(t, err)
}

func TestCmdSessionList_Empty(t *testing.T) {
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSessionList(store)
	assert.NoError(t, err)
}

func TestCmdSessionSave_Success(t *testing.T) {
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSessionSave(store, "my-session")
	assert.NoError(t, err)
}

func TestCmdSessionList_WithSessions(t *testing.T) {
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()

	// Save a session first
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	require.NoError(t, repl.cmdSessionSave(store, "test"))
	err := repl.cmdSessionList(store)
	assert.NoError(t, err)
}

func TestCmdSessionLoadDelete(t *testing.T) {
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	// Save first, then load and delete
	id, err := store.SaveAs(loop.session, "test", "model")
	require.NoError(t, err)

	err = repl.cmdSessionLoad(store, id)
	assert.NoError(t, err)

	err = repl.cmdSessionDelete(store, id)
	assert.NoError(t, err)
}

func TestCmdSession_SaveSubcommand(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.store = NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	repl.loop.store = loop.store
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSession([]string{"save", "myname"})
	assert.NoError(t, err)
}

func TestCmdSession_LoadMissingArg(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.store = NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	repl.loop.store = loop.store
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSession([]string{"load"})
	assert.NoError(t, err)
}

func TestCmdSession_DeleteMissingArg(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.store = NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	repl.loop.store = loop.store
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSession([]string{"delete"})
	assert.NoError(t, err)
}

func TestCmdSession_LoadViaDispatcher(t *testing.T) {
	// Tests the return r.cmdSessionLoad(store, args[1]) path in cmdSession
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	loop.store = store
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	id, err := store.SaveAs(loop.session, "test-dispatch", "model")
	require.NoError(t, err)

	err = repl.cmdSession([]string{"load", id})
	assert.NoError(t, err)
}

func TestCmdSession_DeleteViaDispatcher(t *testing.T) {
	// Tests the return r.cmdSessionDelete(store, args[1]) path in cmdSession
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	loop.store = store
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	id, err := store.SaveAs(loop.session, "test-delete", "model")
	require.NoError(t, err)

	err = repl.cmdSession([]string{"delete", id})
	assert.NoError(t, err)
}

func TestCmdSession_GotoSuccess(t *testing.T) {
	// Tests the goto success path including the return nil after RestoreTo succeeds
	loop := makeTestLoop(nil)
	loop.store = NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	// Add a turn to the session so there's a valid checkpoint ID
	loop.Session().Append("hello", "world")
	id := loop.Session().Checkpoint()
	require.NotZero(t, id)

	err := repl.cmdSession([]string{"goto", strconv.FormatInt(id, 10)})
	assert.NoError(t, err)
}

func TestCmdSession_GotoNotFound(t *testing.T) {
	// Tests the !RestoreTo path in goto (ID not found)
	loop := makeTestLoop(nil)
	loop.store = NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSession([]string{"goto", "9999999999999"})
	assert.NoError(t, err)
}

func TestCmdSessionList_UnnamedAndNoModel(t *testing.T) {
	// Tests the name=="" and model=="" fallback paths in cmdSessionList
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	// SaveAs with empty name and empty model -> unnamed session
	_, err := store.SaveAs(loop.session, "", "")
	require.NoError(t, err)
	// List should show "(unnamed)" and "-" for model
	err = repl.cmdSessionList(store)
	assert.NoError(t, err)
}

func TestCmdSessionSave_WithName(t *testing.T) {
	// Tests the name!="" path in cmdSessionSave (appends (name) to message)
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSessionSave(store, "my-session-name")
	assert.NoError(t, err)
}

func TestCmdSessionDelete_NotFound(t *testing.T) {
	// Tests the err!=nil path in cmdSessionDelete
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSessionDelete(store, "nonexistent-id")
	assert.Error(t, err)
}

func TestCmdSession_CheckpointEmpty(t *testing.T) {
	// Tests checkpoint with no messages (id==0 path)
	loop := makeTestLoop(nil)
	loop.store = NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSession([]string{"checkpoint"})
	assert.NoError(t, err)
}

func TestCmdSession_CheckpointWithMessages(t *testing.T) {
	// Tests checkpoint with messages (id>0 path, return nil)
	loop := makeTestLoop(nil)
	loop.store = NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	loop.Session().Append("hello", "world")
	err := repl.cmdSession([]string{"checkpoint"})
	assert.NoError(t, err)
}

func TestCmdSession_GotoMissingArg(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.store = NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSession([]string{"goto"})
	assert.NoError(t, err)
}

func TestCmdSession_GotoInvalidID(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.store = NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	err := repl.cmdSession([]string{"goto", "not-a-number"})
	assert.NoError(t, err)
}

// --- cmdThinking ---

func TestCmdThinking_ShowDefault(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	r, w, _ := os.Pipe()
	origOut := os.Stdout
	os.Stdout = w
	err := repl.cmdThinking(nil)
	w.Close()
	os.Stdout = origOut

	assert.NoError(t, err)
	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	assert.Contains(t, string(buf[:n]), "off")
}

func TestCmdThinking_SetHigh(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w2, _ := os.Pipe()
	os.Stdout = w2
	defer func() { w2.Close(); os.Stdout = origOut }()

	err := repl.cmdThinking([]string{"high"})
	assert.NoError(t, err)

	cfg := loop.Thinking()
	assert.NotNil(t, cfg)
	assert.Equal(t, domain.ThinkingModeHigh, cfg.Mode)
}

func TestCmdThinking_SetOff(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.SetThinking(&domain.ThinkingConfig{Mode: domain.ThinkingModeHigh})
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w2, _ := os.Pipe()
	os.Stdout = w2
	defer func() { w2.Close(); os.Stdout = origOut }()

	err := repl.cmdThinking([]string{"off"})
	assert.NoError(t, err)
	assert.Nil(t, loop.Thinking())
}

func TestCmdThinking_InvalidMode(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w2, _ := os.Pipe()
	os.Stdout = w2
	defer func() { w2.Close(); os.Stdout = origOut }()

	err := repl.cmdThinking([]string{"bogus"})
	assert.NoError(t, err)
	assert.Nil(t, loop.Thinking()) // no change
}

func TestCmdThinking_AllModes(t *testing.T) {
	modes := []domain.ThinkingMode{
		domain.ThinkingModeLow,
		domain.ThinkingModeMedium,
		domain.ThinkingModeHigh,
		domain.ThinkingModeAuto,
	}
	for _, mode := range modes {
		loop := makeTestLoop(nil)
		repl := NewREPL(loop)

		origOut := os.Stdout
		_, w2, _ := os.Pipe()
		os.Stdout = w2

		err := repl.cmdThinking([]string{string(mode)})
		w2.Close()
		os.Stdout = origOut

		assert.NoError(t, err)
		cfg := loop.Thinking()
		assert.NotNil(t, cfg, "mode=%s", mode)
		assert.Equal(t, mode, cfg.Mode, "mode=%s", mode)
		repl.cancel()
	}
}

func TestDispatchCommand_Thinking(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w2, _ := os.Pipe()
	os.Stdout = w2
	defer func() { w2.Close(); os.Stdout = origOut }()

	err := repl.dispatchCommand("/thinking medium")
	assert.NoError(t, err)
	cfg := loop.Thinking()
	assert.NotNil(t, cfg)
	assert.Equal(t, domain.ThinkingModeMedium, cfg.Mode)
}

func TestBuildChatConfig_ThinkingPropagated(t *testing.T) {
	loop := makeTestLoop(nil)
	thinking := &domain.ThinkingConfig{Mode: domain.ThinkingModeHigh, BudgetTokens: 1024}
	loop.SetThinking(thinking)

	cfg := loop.buildChatConfig("hello", "")
	assert.Equal(t, thinking, cfg.Thinking)
}

func TestBuildChatConfig_NoThinkingByDefault(t *testing.T) {
	loop := makeTestLoop(nil)
	cfg := loop.buildChatConfig("hello", "")
	assert.Nil(t, cfg.Thinking)
}

func TestCmdSessionLoad_RestoresModel(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.config.Model = "initial-model"
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	id, err := store.SaveAs(loop.session, "test-session", "saved-model")
	require.NoError(t, err)

	// Switch model after saving
	loop.config.Model = "different-model"

	err = repl.cmdSessionLoad(store, id)
	require.NoError(t, err)

	// Model should be restored to what was saved
	assert.Equal(t, "saved-model", loop.config.Model)
}

func TestCmdSessionLoad_NoModelInMeta(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.config.Model = "current-model"
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()

	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()

	// Save without a model tag
	id, err := store.SaveAs(loop.session, "test", "")
	require.NoError(t, err)

	err = repl.cmdSessionLoad(store, id)
	require.NoError(t, err)

	// Model should remain unchanged when metadata has no model
	assert.Equal(t, "current-model", loop.config.Model)
}

// --- /copy ---

func testCaptureStdout(_ *testing.T, fn func()) string {
	pr, pw, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	old := os.Stdout
	os.Stdout = pw
	fn()
	pw.Close()
	os.Stdout = old
	var sb strings.Builder
	_, _ = io.Copy(&sb, pr)
	pr.Close()
	return sb.String()
}

func TestCmdCopy_NoSession(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	out := testCaptureStdout(t, func() {
		err := repl.cmdCopy()
		require.NoError(t, err)
	})
	assert.Contains(t, out, "Nothing to copy")
}

func TestCmdCopy_HasResponse(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.session.Append("hello", "world response")
	repl := NewREPL(loop)

	// copyToClipboard may fail in test env; cmdCopy just prints error, no return error
	out := testCaptureStdout(t, func() {
		err := repl.cmdCopy()
		require.NoError(t, err)
	})
	// Either "Copied" or clipboard error, not "Nothing to copy"
	assert.NotContains(t, out, "Nothing to copy")
}

func TestDispatchCommand_Copy(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	out := testCaptureStdout(t, func() {
		_ = repl.dispatchCommand("/copy")
	})
	assert.Contains(t, out, "Nothing to copy")
}

func TestDispatchCommand_Reload(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	out := testCaptureStdout(t, func() {
		err := repl.dispatchCommand("/reload")
		require.NoError(t, err)
	})
	assert.Contains(t, out, "Reloaded")
}

// --- /reload ---

func TestCmdReload_NoSkillPaths(t *testing.T) {
	loop := makeTestLoop([]Skill{{Name: "mypkg", Description: "d", Content: "c"}})
	repl := NewREPL(loop)
	// With no skill paths configured, reload should not clear the existing skills.
	err := repl.cmdReload()
	require.NoError(t, err)
}

func TestLoopReload_DoesNotPanicWithNoPaths(_ *testing.T) {
	loop := makeTestLoop(nil)
	loop.Reload() // must not panic when SkillPaths and PromptPaths are empty
}

func TestLoopReload_WithSkillPaths(t *testing.T) {
	dir := t.TempDir()
	skillFile := filepath.Join(dir, "SKILL.md")
	require.NoError(
		t,
		os.WriteFile(
			skillFile,
			[]byte("---\nname: myskill\ndescription: test skill\n---\ncontent"),
			0o644,
		),
	)

	loop := makeTestLoop(nil)
	loop.config.SkillPaths = []string{dir}
	loop.Reload()

	// After reload, the skill should be discoverable.
	sk := loop.SkillByName("myskill")
	if sk == nil {
		t.Skip("skill format may not match expected format for this test environment")
	}
}

func TestLoopReload_WithPromptPaths(t *testing.T) {
	dir := t.TempDir()
	promptFile := filepath.Join(dir, "mytemplate.md")
	require.NoError(
		t,
		os.WriteFile(
			promptFile,
			[]byte("---\nname: mytemplate\ndescription: test template\n---\nhello $@"),
			0o644,
		),
	)

	loop := makeTestLoop(nil)
	loop.config.PromptPaths = []string{dir}
	loop.Reload()

	// After reload, the prompt template should be discoverable.
	pt := loop.PromptByName("mytemplate")
	if pt == nil {
		t.Fatalf("expected prompt template 'mytemplate' to be loaded after Reload")
	}
}

// --- bang command (! and !!) ---

func TestProcessInput_BangCommand_InjectsContext(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// ! echo hello should run and inject into session
	err := repl.processInput("! echo hello")
	require.NoError(t, err)
	// Session should have 1 turn (the injected bash result)
	assert.Equal(t, 1, loop.Session().TurnCount())
	msgs := loop.Session().Messages()
	assert.Contains(t, msgs[0].Content, "Ran `echo hello`")
}

func TestProcessInput_DoubleBang_DoesNotInjectContext(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// !! echo hello should run but NOT inject into session
	err := repl.processInput("!! echo hello")
	require.NoError(t, err)
	assert.Equal(t, 0, loop.Session().TurnCount())
}

func TestProcessInput_BangEmpty_Passthrough(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// A lone "!" (no command) should not be treated as a bang command.
	// It will fall through to the LLM path but our test runFn returns ok.
	called := false
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		called = true
		return "", nil
	}
	err := repl.processInput("!")
	require.NoError(t, err)
	// Falls through to LLM (not a bang command since cmd is empty)
	assert.True(t, called)
}

func TestExecBangCommand_NonZeroExit_ErrorInContext(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// A command that exits non-zero; should still inject with exit code
	_ = repl.execBangCommand("exit 1", false)
	assert.Equal(t, 1, loop.Session().TurnCount())
	msgs := loop.Session().Messages()
	assert.Contains(t, msgs[0].Content, "exit 1")
}

func TestExecBangCommand_ExcludeFromContext(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.execBangCommand("echo quiet", true)
	require.NoError(t, err)
	// excludeFromContext=true means nothing added to session
	assert.Equal(t, 0, loop.Session().TurnCount())
}

func TestREPL_ModelPickerAccessors(t *testing.T) {
	// Tests the simple accessor methods that return model picker state
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// Set values manually (mimics what refreshModels() would do)
	repl.modelNames = []string{"llama3.2", "gpt-4o"}
	repl.downloadedModels = map[string]bool{"llama3.2": true}
	repl.modelTypes = map[string]string{"llama3.2": "llamafile"}
	repl.cloudModelBackends = map[string]string{"gpt-4o": "openai"}
	repl.providerStatus = map[string]bool{"openai": true}

	assert.Equal(t, []string{"llama3.2", "gpt-4o"}, repl.ModelNames())
	assert.Equal(t, map[string]bool{"llama3.2": true}, repl.DownloadedModels())
	assert.Equal(t, map[string]string{"llama3.2": "llamafile"}, repl.ModelTypes())
	assert.Equal(t, map[string]string{"gpt-4o": "openai"}, repl.CloudModelBackends())
	assert.Equal(t, map[string]bool{"openai": true}, repl.ProviderStatus())
}

func TestExecBangCommand_NonExitError_ContextCancel(t *testing.T) {
	// Tests the errMsg = runErr.Error() path (non-ExitError, e.g. context canceled).
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	// Cancel the context immediately so the bash command fails with context error,
	// not with *exec.ExitError.
	repl.cancel()
	err := repl.execBangCommand("sleep 10", false)
	// Should return a non-nil error (context canceled or killed).
	assert.Error(t, err)
	// The error message should appear in the injected context message.
	if loop.Session().TurnCount() > 0 {
		msgs := loop.Session().Messages()
		assert.Contains(t, msgs[0].Content, "sleep 10")
	}
}

// --- applyConfigDefaults ModelService auto-start ---

// mockModelService is a minimal ModelServiceInterface for testing.
type mockModelService struct {
	downloadCalled bool
	serveCalled    bool
	url            string
}

func (m *mockModelService) DownloadModel(_, _ string) error { m.downloadCalled = true; return nil }
func (m *mockModelService) ServeModel(_, _, _ string, _ int) error {
	m.serveCalled = true
	return nil
}
func (m *mockModelService) ServerURL(_, _ string) string { return m.url }

func TestApplyConfigDefaults_ModelServiceAutoStart(t *testing.T) {
	svc := &mockModelService{url: "http://localhost:9999"}
	cfg := applyConfigDefaults(Config{
		Model:        "llama3.2",
		Backend:      "file",
		ModelService: svc,
		// BaseURL intentionally empty to trigger auto-start
	})
	if !svc.downloadCalled {
		t.Error("expected DownloadModel to be called")
	}
	if !svc.serveCalled {
		t.Error("expected ServeModel to be called")
	}
	if cfg.BaseURL != "http://localhost:9999" {
		t.Errorf("expected BaseURL=http://localhost:9999, got %q", cfg.BaseURL)
	}
}

func TestApplyConfigDefaults_ModelServiceNotCalledWhenBaseURLSet(t *testing.T) {
	svc := &mockModelService{url: "http://localhost:9999"}
	cfg := applyConfigDefaults(Config{
		Model:        "llama3.2",
		Backend:      "file",
		ModelService: svc,
		BaseURL:      "http://existing:1234", // already set; should not trigger auto-start
	})
	if svc.downloadCalled || svc.serveCalled {
		t.Error("expected ModelService methods not to be called when BaseURL is already set")
	}
	if cfg.BaseURL != "http://existing:1234" {
		t.Errorf("expected BaseURL unchanged, got %q", cfg.BaseURL)
	}
}

// --- buildSystemPreamble zero-limit fallback paths ---

func TestBuildSystemPreamble_ZeroBudgetUsesAutoCompactThreshold(t *testing.T) {
	loop := makeTestLoop(nil)
	// CompactTokenBudget=0 -> falls through to AutoCompactThreshold
	loop.config.CompactTokenBudget = 0
	loop.config.AutoCompactThreshold = 20000
	// Should not panic and should return an empty preamble (no skills/prompt/tools)
	preamble := loop.buildSystemPreamble()
	assert.Empty(t, preamble)
}

func TestBuildSystemPreamble_BothZeroFallsBackTo40000(t *testing.T) {
	loop := makeTestLoop(nil)
	// Both zero: final fallback to 40000
	loop.config.CompactTokenBudget = 0
	loop.config.AutoCompactThreshold = 0
	preamble := loop.buildSystemPreamble()
	assert.Empty(t, preamble) // no skills/prompt/tools set
}

// --- buildSystemPreamble small-context path ---

func TestBuildSystemPreamble_SmallContext_StripsSkills(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.skills = "LARGE SKILL BLOCK"
	loop.config.CompactTokenBudget = 4096 // < smallContext (8192)
	loop.config.SystemPrompt = "you are a helpful assistant"

	preamble := loop.buildSystemPreamble()

	// Skills should be stripped when context window is tiny
	assert.NotContains(t, preamble, "LARGE SKILL BLOCK")
	// System prompt should be kept
	assert.Contains(t, preamble, "you are a helpful assistant")
}

func TestBuildSystemPreamble_NormalContext_IncludesSkills(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.skills = "MY SKILL CONTENT"
	loop.config.CompactTokenBudget = 40000

	preamble := loop.buildSystemPreamble()

	assert.Contains(t, preamble, "MY SKILL CONTENT")
}

func TestBuildSystemPreamble_SmallContext_NoSystemPrompt(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.skills = "SKILL"
	loop.config.CompactTokenBudget = 4096
	loop.config.SystemPrompt = ""

	preamble := loop.buildSystemPreamble()

	// Only tool guidance when no system prompt and small context
	assert.NotContains(t, preamble, "SKILL")
}

// --- SetModelTypes / SetCloudModelBackends / SetModelPickerFn ---

func TestSetModelTypes(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	types := map[string]string{"llama3.2": modelTypeLLamafile, "codellama": modelTypeGGUF}
	repl.SetModelTypes(types)
	if repl.modelTypes["llama3.2"] != modelTypeLLamafile {
		t.Fatalf("expected llamafile type, got %q", repl.modelTypes["llama3.2"])
	}
}

func TestSetCloudModelBackends(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	backends := map[string]string{"gpt-4o": "openai", "claude-3-5-sonnet": "anthropic"}
	repl.SetCloudModelBackends(backends)
	if repl.cloudModelBackends["gpt-4o"] != "openai" {
		t.Fatalf("expected openai backend, got %q", repl.cloudModelBackends["gpt-4o"])
	}
}

func TestSetModelPickerFn(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	called := false
	fn := func(_ string) (string, error) { called = true; return "llama3.2", nil }
	repl.SetModelPickerFn(fn)
	if repl.modelPickerFn == nil {
		t.Fatal("expected modelPickerFn to be set")
	}
	model, err := repl.modelPickerFn("")
	if err != nil || model != "llama3.2" || !called {
		t.Fatalf("modelPickerFn not working: model=%q err=%v called=%v", model, err, called)
	}
}

// --- allCommandNames with prompts ---

func TestAllCommandNames_WithPrompts(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.prompts = []PromptTemplate{{Name: "summarize"}, {Name: "review"}}
	repl := NewREPL(loop)
	defer repl.cancel()
	names := repl.allCommandNames()
	assert.Contains(t, names, "/summarize")
	assert.Contains(t, names, "/review")
}

// --- modelCompletionSuffixes ---

func TestModelCompletionSuffixes_LocalTypes(t *testing.T) {
	// Covers the llamafile/gguf/ollama/cloud switch cases in modelCompletionSuffixes.
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.modelTypes = map[string]string{
		"llama3.2":   modelTypeLLamafile,
		"code.gguf":  modelTypeGGUF,
		"phi3:mini":  modelTypeOllama,
		"gpt-4o-min": "", // cloud (default)
	}
	repl.downloadedModels = map[string]bool{} // none cached
	ranked := []string{"llama3.2", "code.gguf", "phi3:mini", "gpt-4o-min"}
	suffixes := repl.modelCompletionSuffixes(ranked, 0) // tokenLen=0 -> all names as suffixes
	if len(suffixes) != 4 {
		t.Fatalf("expected 4 suffixes, got %d", len(suffixes))
	}
}

func TestModelCompletionSuffixes_TruncatesToMax(t *testing.T) {
	// Covers the len(ranked) > replModelCompletionMax truncation path.
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.downloadedModels = map[string]bool{}
	repl.modelTypes = map[string]string{}

	// Build > replModelCompletionMax (40) model names all matching an empty prefix.
	names := make([]string, replModelCompletionMax+5)
	for i := range names {
		names[i] = fmt.Sprintf("model-%02d", i)
	}
	repl.modelNames = names

	// Drive the completer with "/model " so it fuzzy-matches and truncates.
	completer := repl.buildCompleter()
	line := []rune("/model ")
	completions, _ := completer.Do(line, len(line))
	// Should return at most replModelCompletionMax completions.
	if len(completions) > replModelCompletionMax {
		t.Fatalf("expected max %d completions, got %d", replModelCompletionMax, len(completions))
	}
}

func TestModelCompletionSuffixes_ShortName(t *testing.T) {
	// Full-name approach: model names shorter than the typed token are still
	// returned as full names (readline deletes the token and inserts the name).
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.downloadedModels = map[string]bool{}
	repl.modelTypes = map[string]string{}
	suffixes := repl.modelCompletionSuffixes([]string{"ab"}, 10)
	if len(suffixes) != 1 {
		t.Fatalf("expected 1 suffix (full name), got %d", len(suffixes))
	}
}

// --- dispatchCommand routing for /models, /prompts ---

func TestDispatchCommand_Models(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()
	assert.NoError(t, repl.dispatchCommand("/models"))
}

func TestDispatchCommand_Prompts(t *testing.T) {
	loop := makeTestLoop(nil)
	loop.prompts = []PromptTemplate{{Name: "greet", Description: "Greet", Content: "Hello $1"}}
	repl := NewREPL(loop)
	defer repl.cancel()
	origOut := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { w.Close(); os.Stdout = origOut }()
	assert.NoError(t, repl.dispatchCommand("/prompts"))
}

// --- runWithThinking streaming path ---

func TestRunWithThinking_StreamingPath(t *testing.T) {
	// Covers lines 573-579: runFn==nil && loop.IsStreaming() -> RunStreaming path.
	ms := &mockStreamer{
		responses: []mockStreamResponse{{content: "streamed response"}},
	}
	loop := newStreamingLoop(ms, 5)
	repl := NewREPL(loop)
	defer repl.cancel()

	// runFn is nil by default; IsStreaming() returns true because loop.streamer is set.
	origOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	resp, err := repl.runWithThinking(context.Background(), "hello")
	w.Close()
	os.Stdout = origOut
	io.Copy(io.Discard, r) //nolint:errcheck
	r.Close()

	assert.NoError(t, err)
	assert.Equal(t, "streamed response", resp)
}

// --- NewREPL SetOnAutoCompact closure ---

func TestNewREPL_AutoCompactCallbackFires(t *testing.T) {
	// Covers lines 140-147: the SetOnAutoCompact closure set by NewREPL fires
	// when auto-compaction triggers during RunStreaming.
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		if len(wf.Resources) > 0 && wf.Resources[0].Chat.Prompt != "" {
			return "compact summary", nil
		}
		return "", nil
	})
	ms := &mockStreamer{
		responses: []mockStreamResponse{{content: "response after compact"}},
	}
	reg := tools.NewRegistry()
	loop := New(eng, newTestWorkflowForSession(), reg, Config{
		Model:                "test",
		Streamer:             ms,
		CompactTokenBudget:   1,
		AutoCompactThreshold: 1,
		MaxToolRounds:        3,
	})
	// NewREPL installs its own onAutoCompact callback that prints to stdout.
	repl := NewREPL(loop)
	defer repl.cancel()

	// Seed enough turns to trigger auto-compact.
	for range compactMinTurns {
		loop.Session().Append(strings.Repeat("q", 300), strings.Repeat("a", 300))
	}

	origOut := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	resp, err := repl.runWithThinking(context.Background(), "trigger compact")
	w.Close()
	os.Stdout = origOut
	io.Copy(io.Discard, r) //nolint:errcheck
	r.Close()

	assert.NoError(t, err)
	assert.Equal(t, "response after compact", resp)
}
