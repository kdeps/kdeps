package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chzyer/readline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
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

	c := &replCompleter{repl: repl}
	// "/model llama" - token="llama" (5), suffixes should be "3.2:1b" and "3.2:3b"
	input := []rune("/model llama")
	results, length := c.Do(input, len(input))
	assert.Equal(t, len([]rune("llama")), length)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	// readline displays same+suffix: "llama"+"3.2:1b" = "llama3.2:1b"
	assert.Contains(t, found, "3.2:1b")
	assert.Contains(t, found, "3.2:3b")
	assert.NotContains(t, found, "qwen3.5-4b")
}

func TestReplCompleter_ModelArgAllModels(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetModelNames([]string{"llama3.2:1b", "qwen3.5-4b"})

	c := &replCompleter{repl: repl}
	// "/model " - empty token, suffixes = full model names (tokenLen=0)
	input := []rune("/model ")
	results, _ := c.Do(input, len(input))
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	assert.Contains(t, found, "llama3.2:1b")
	assert.Contains(t, found, "qwen3.5-4b")
}

func TestReplCompleter_DownloadedModelMarker(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetModelNames([]string{"llama3.2:1b", "llama3.2:3b", "qwen3.5-4b"})
	repl.SetDownloadedModels(map[string]bool{"llama3.2:1b": true})

	c := &replCompleter{repl: repl}
	// empty token - all models, downloaded first with "*" prefix
	input := []rune("/model ")
	results, _ := c.Do(input, len(input))
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	// downloaded model gets "*" prefix
	assert.Contains(t, found, "*llama3.2:1b")
	// non-downloaded models have no marker
	assert.Contains(t, found, "llama3.2:3b")
	assert.Contains(t, found, "qwen3.5-4b")
	// downloaded model appears first
	assert.Equal(t, "*llama3.2:1b", found[0])
}

func TestReplCompleter_DownloadedModelMarkerPartialToken(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()
	repl.SetModelNames([]string{"llama3.2:1b", "llama3.2:3b"})
	repl.SetDownloadedModels(map[string]bool{"llama3.2:1b": true})

	c := &replCompleter{repl: repl}
	// token="llama3.2" - suffixes: "*:1b" (downloaded), ":3b" (not)
	input := []rune("/model llama3.2")
	results, length := c.Do(input, len(input))
	assert.Equal(t, len([]rune("llama3.2")), length)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	assert.Contains(t, found, "*:1b")
	assert.Contains(t, found, ":3b")
	assert.Equal(t, "*:1b", found[0])
}

func TestCmdModel_StripsStar(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// Simulate selection of a "*"-prefixed model name.
	_ = repl.cmdModel([]string{"qwen2.5*:7b"})
	assert.Equal(t, "qwen2.5:7b", repl.loop.config.Model)
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
