package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// --- expandFileRefs ---

func TestExpandFileRefs_NoRefs(t *testing.T) {
	out := expandFileRefs("hello world")
	assert.Equal(t, "hello world", out)
}

func TestExpandFileRefs_UnreadablePath(t *testing.T) {
	// @nonexistent-file should be left as-is
	out := expandFileRefs("check @/nonexistent/file.txt please")
	assert.Contains(t, out, "@/nonexistent/file.txt")
}

func TestExpandFileRefs_RealFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(p, []byte("hello from file"), 0o644))

	out := expandFileRefs("review @" + p)
	assert.Contains(t, out, "hello from file")
	assert.Contains(t, out, "notes.txt")
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
	// "/h" should fuzzy-match "/help" and "/history"
	results, length := c.Do([]rune("/h"), 2)
	assert.Equal(t, 2, length)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	assert.Contains(t, found, "/help")
	assert.Contains(t, found, "/history")
}

func TestReplCompleter_SlashSkill(t *testing.T) {
	loop := makeTestLoop([]Skill{{Name: "review", Description: "code review"}})
	repl := NewREPL(loop)
	defer repl.cancel()

	c := &replCompleter{repl: repl}
	// "/rev" should match "/review"
	results, length := c.Do([]rune("/rev"), 4)
	assert.Equal(t, 4, length)
	found := make([]string, 0, len(results))
	for _, r := range results {
		found = append(found, string(r))
	}
	assert.Contains(t, found, "/review")
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
