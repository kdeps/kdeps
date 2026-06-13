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
