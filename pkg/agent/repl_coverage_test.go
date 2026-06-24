package agent

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- dispatchCommand coverage: untested routes ---

func TestDispatchCommand_Session(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// Store must be nil so the "store == nil" branch in cmdSession is hit.
	out := testCaptureStdout(t, func() {
		err := repl.dispatchCommand("/session")
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Session store not available")
}

func TestDispatchCommand_Editor(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "/nonexistent-editor-for-test")

	err := repl.dispatchCommand("/editor")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "editor:")
}

func TestDispatchCommand_Processes(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	out := testCaptureStdout(t, func() {
		err := repl.dispatchCommand("/processes")
		assert.NoError(t, err)
	})
	// /processes with no args prints the list output.
	assert.Contains(t, out, "No local model servers")
}

func TestDispatchCommand_HFF(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	out := testCaptureStdout(t, func() {
		err := repl.dispatchCommand("/hff")
		assert.NoError(t, err)
	})
	// /hff with no args prints usage.
	assert.Contains(t, out, "Usage:")
}

func TestDispatchCommand_PromptByName(t *testing.T) {
	loop := makeTestLoop(nil)
	// Register a prompt so the dispatch can find it.
	loop.prompts = []PromptTemplate{{Name: "greet", Description: "greeting", Content: "hello"}}
	repl := NewREPL(loop)
	defer repl.cancel()

	// runFn intercepts the LLM call so it doesn't hit a real engine.
	called := false
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		called = true
		return "", nil
	}

	err := repl.dispatchCommand("/greet")
	assert.NoError(t, err)
	assert.True(t, called, "runFn should have been called for prompt dispatch")
}

// --- cmdModel coverage: default subcommand routing through cmdModel ---

func TestCmdModel_DefaultSubcommand(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	saved := ""
	repl.SetSaveDefaultFn(func(m string) error { saved = m; return nil })

	err := repl.cmdModel([]string{"default", "my-model"})
	assert.NoError(t, err)
	assert.Equal(t, "my-model", saved)
}

func TestCmdModel_PickerWithFilter(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	repl.SetModelNames([]string{"llama3.2:1b", "gpt-4o"})
	calledWith := ""
	repl.SetModelPickerFn(func(filter string) (string, error) {
		calledWith = filter
		return "", nil
	})

	// Unknown model name with pickerFn set: opens picker with the name as filter.
	err := repl.cmdModel([]string{"unknown-model"})
	assert.NoError(t, err)
	assert.Equal(t, "unknown-model", calledWith)
}

func TestCmdModel_PickerNoArgs(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	called := false
	repl.SetModelPickerFn(func(filter string) (string, error) {
		called = true
		assert.Equal(t, "", filter, "no-args picker should get empty filter")
		return "", nil
	})

	err := repl.cmdModel(nil)
	assert.NoError(t, err)
	assert.True(t, called)
}

// --- cmdModelDefault coverage: stripTagKeywordPrefix resolution ---

func TestCmdModelDefault_WithTagKeywordPrefix(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	repl.SetModelNames([]string{"my-model"})
	saved := ""
	repl.SetSaveDefaultFn(func(m string) error { saved = m; return nil })

	// "ggufmy-model" should be resolved to "my-model" by stripTagKeywordPrefix.
	err := repl.cmdModelDefault([]string{"ggufmy-model"})
	assert.NoError(t, err)
	assert.Equal(t, "my-model", saved)
}

// --- autoSaveOnExit coverage: SaveAs error path ---

func TestAutoSaveOnExit_SaveError(t *testing.T) {
	dir := t.TempDir()
	// Make the session store directory unwritable to trigger a save error.
	store := NewSessionStore(dir)
	loop := makeTestLoop(nil)
	loop.store = store
	loop.session.Append("hi", "hello")

	repl := NewREPL(loop)
	defer repl.cancel()

	// Remove write permission on the store directory so SaveAs fails.
	require.NoError(t, os.Chmod(dir, 0500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

	r, w, _ := os.Pipe()
	orig := os.Stderr
	os.Stderr = w

	repl.autoSaveOnExit()

	w.Close()
	os.Stderr = orig
	out, _ := io.ReadAll(r)
	r.Close()

	assert.Contains(t, string(out), "session auto-save failed")
}

// --- cmdSession coverage: subcommand routing through cmdSession ---

func TestCmdSession_ListSubcommand(t *testing.T) {
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	loop.store = store
	repl := NewREPL(loop)
	defer repl.cancel()

	out := testCaptureStdout(t, func() {
		err := repl.cmdSession([]string{"list"})
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "No saved sessions")
}

func TestCmdSession_ImportSubcommand_MissingArg(t *testing.T) {
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	loop.store = store
	repl := NewREPL(loop)
	defer repl.cancel()

	out := testCaptureStdout(t, func() {
		err := repl.cmdSession([]string{"import"})
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Usage: /session import")
}

func TestCmdSession_ImportSubcommand_Success(t *testing.T) {
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	loop.store = store
	repl := NewREPL(loop)
	defer repl.cancel()

	// Create a valid session file to import.
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "session.jsonl")
	content := `{"type":"session_meta","ts":1000,"sessionId":"test","turns":0}` + "\n"
	require.NoError(t, os.WriteFile(srcPath, []byte(content), 0600))

	out := testCaptureStdout(t, func() {
		err := repl.cmdSession([]string{"import", srcPath})
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "loaded")
}

func TestCmdSession_BranchesSubcommand(t *testing.T) {
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	loop.store = store
	repl := NewREPL(loop)
	defer repl.cancel()

	out := testCaptureStdout(t, func() {
		err := repl.cmdSession([]string{"branches"})
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "No stashed branches")
}

// --- cmdSessionBranches coverage: display loop with stashes ---

func TestCmdSessionBranches_WithStashesLoop(t *testing.T) {
	loop := makeTestLoop(nil)
	session := NewSession(0)
	// Append 1 turn, checkpoint, then append more so RestoreTo prunes turn 2.
	session.Append("hello", "world")
	cp := session.Checkpoint()
	session.Append("followup", "response")
	session.RestoreTo(cp) // creates 1 stash

	loop.session = session
	repl := NewREPL(loop)
	defer repl.cancel()

	out := testCaptureStdout(t, func() {
		err := repl.cmdSessionBranches()
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "stashed")
	// Should show branch point and turn IDs.
	assert.Contains(t, out, "branch")
	assert.Contains(t, out, "Entry IDs")
}

// --- cmdSessionList coverage: error path ---

func TestCmdSessionList_StoreError(t *testing.T) {
	loop := makeTestLoop(nil)
	// Create a store where the base path is a file, so ListMeta fails.
	dir := t.TempDir()
	store := NewSessionStore(dir)
	loop.store = store
	repl := NewREPL(loop)
	defer repl.cancel()

	// Remove the directory and create a regular file in its place so
	// os.ReadDir in ListMeta fails (not IsNotExist).
	basePath := store.sessionBasePath()
	require.NoError(t, os.RemoveAll(basePath))
	require.NoError(t, os.WriteFile(basePath, []byte("not-a-dir"), 0600))
	t.Cleanup(func() { _ = os.Remove(basePath) })

	err := repl.cmdSessionList(store)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session list")
}

// --- cmdSessionSave coverage: error path ---

func TestCmdSessionSave_StoreError(t *testing.T) {
	loop := makeTestLoop(nil)
	dir := t.TempDir()
	store := NewSessionStore(dir)
	repl := NewREPL(loop)
	defer repl.cancel()

	// Remove write permission so SaveAs fails.
	require.NoError(t, os.Chmod(dir, 0500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

	err := repl.cmdSessionSave(store, "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session save")
}

// --- cmdSessionLoad coverage: error path ---

func TestCmdSessionLoad_StoreError(t *testing.T) {
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()

	err := repl.cmdSessionLoad(store, "nonexistent-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session load")
}

// --- cmdSessionImport coverage: ~/ expansion and Import error ---

func TestCmdSessionImport_TildeExpansion(t *testing.T) {
	loop := makeTestLoop(nil)
	store := NewSessionStore(t.TempDir())
	repl := NewREPL(loop)
	defer repl.cancel()

	// /session import ~/nonexistent should expand to $HOME/nonexistent
	out := testCaptureStdout(t, func() {
		err := repl.cmdSessionImport(store, "~/nonexistent-import-file")
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "File not found")
}

func TestCmdSessionImport_ImportError(t *testing.T) {
	loop := makeTestLoop(nil)
	dir := t.TempDir()
	// Make the store dir read-only so Import's os.WriteFile fails.
	store := NewSessionStore(dir)
	repl := NewREPL(loop)
	defer repl.cancel()

	require.NoError(t, os.Chmod(dir, 0500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0700) })

	srcPath := filepath.Join(t.TempDir(), "valid.jsonl")
	require.NoError(t, os.WriteFile(srcPath, []byte("{\"type\":\"session_meta\"}\n"), 0600))

	err := repl.cmdSessionImport(store, srcPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "session import")
}

// --- cmdEditor coverage: fallback to vi when both VISUAL and EDITOR are empty ---

func TestCmdEditor_FallbackWhenBothEmpty(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	t.Setenv("PATH", "")

	loop := makeTestLoop(nil)
	repl := NewREPL(loop)
	defer repl.cancel()

	// Both env vars are empty, so editor should default to "vi".
	// With PATH empty, vi won't be found, so exec fails immediately.
	err := repl.cmdEditor()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "editor:")
}

// --- helper: capture stdout ---

func TestHistoryPath_ReturnsPath(t *testing.T) {
	t.Setenv("HOME", "/test/home/user")
	path := historyPath()
	assert.Contains(t, path, ".kdeps")
	assert.Contains(t, path, "history")
	assert.Contains(t, path, "/test/home/user")
}
