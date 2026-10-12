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

package agent

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/assets"
)

// withCommands swaps the loaded commands for files (name -> YAML) for one test.
func withCommands(t *testing.T, files map[string]string) {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, data := range files {
		fsys["commands/"+name+".yaml"] = &fstest.MapFile{Data: []byte(data)}
	}
	orig := slashCommands
	t.Cleanup(func() { slashCommands = orig })
	slashCommands = loadCommandsFrom(fsys)
}

// Every command kdeps handles in Go has a builtin asset (so /help and
// completion list it), and every shipped file parses.
func TestCommands_EveryGoCommandHasAnAsset(t *testing.T) {
	names := commandNames()
	for _, c := range goCommands {
		assert.Contains(t, names, "/"+c, "no commands/ asset for /%s", c)
	}
	entries, err := builtinCommandsFS.ReadDir("commands")
	require.NoError(t, err)
	assert.Len(t, slashCommands, len(entries), "every shipped command file must load")
}

func TestCommands_BuiltinItemsAreRequired(t *testing.T) {
	items := builtinCommandItems()
	assert.Contains(t, items, "model")
	assert.Contains(t, items, "instruct-now")
	assert.NotContains(t, items, "review", "declared commands can be removed")
	require.Error(t, assets.Remove(t.TempDir(), "commands/model"))
}

func TestParseYAMLCommand_Errors(t *testing.T) {
	cases := map[string]string{
		"unknown field":     "name: x\nusage: [{help: h}]\nrun: [{command: /help}]\nbogus: 1\n",
		"bad name":          "name: a b\nusage: [{help: h}]\nrun: [{command: /help}]\n",
		"slash in name":     "name: /x\nusage: [{help: h}]\nrun: [{command: /help}]\n",
		"no usage":          "name: x\nrun: [{command: /help}]\n",
		"empty help":        "name: x\nusage: [{args: a}]\nrun: [{command: /help}]\n",
		"unknown builtin":   "name: nope\nbuiltin: true\nusage: [{help: h}]\n",
		"builtin with run":  "name: help\nbuiltin: true\nusage: [{help: h}]\nrun: [{command: /help}]\n",
		"shadows a builtin": "name: help\nusage: [{help: h}]\nrun: [{command: /help}]\n",
		"alias shadows":     "name: x\naliases: [exit]\nusage: [{help: h}]\nrun: [{command: /help}]\n",
		"no run":            "name: x\nusage: [{help: h}]\n",
		"both in a step":    "name: x\nusage: [{help: h}]\nrun: [{command: /help, prompt: p}]\n",
		"neither in a step": "name: x\nusage: [{help: h}]\nrun: [{}]\n",
		"command needs a /": "name: x\nusage: [{help: h}]\nrun: [{command: help}]\n",
		"not yaml":          "name: [\n",
	}
	for label, data := range cases {
		_, err := parseYAMLCommand([]byte(data), label)
		assert.Error(t, err, label)
	}
	c, err := parseYAMLCommand([]byte("name: x\naliases: [y]\nusage: [{help: h}]\nrun: [{prompt: p}]\n"), "x")
	require.NoError(t, err)
	assert.Equal(t, []string{"x", "y"}, c.names())
}

func TestLoadCommandsFrom_SortsSkipsAndReplaces(t *testing.T) {
	withCommands(t, map[string]string{
		"b":      "name: b\norder: 20\nusage: [{help: hb}]\nrun: [{command: /help}]\n",
		"a":      "name: a\norder: 20\nusage: [{help: ha}]\nrun: [{command: /help}]\n",
		"first":  "name: first\norder: 10\nusage: [{help: hf}]\nrun: [{command: /help}]\n",
		"broken": "name: [\n",
	})
	names := make([]string, 0, len(slashCommands))
	for _, c := range slashCommands {
		names = append(names, c.Name)
	}
	assert.Equal(t, []string{"first", "a", "b"}, names)
	assert.Nil(t, loadCommandsFrom(fstest.MapFS{}), "no commands dir")
}

func TestCommandHelpLines(t *testing.T) {
	withCommands(t, map[string]string{
		"x": "name: x\nusage: [{help: Show x}, {args: \"set <n>\", help: Set x}]\nrun: [{command: /help}]\n",
	})
	assert.Equal(t, []string{
		"  /x                                 Show x",
		"  /x set <n>                         Set x",
	}, commandHelpLines())
}

func TestDeclaredCommand_RunsCommandAndPromptSteps(t *testing.T) {
	withCommands(t, map[string]string{
		"brief": "name: brief\naliases: [b]\nusage: [{help: h}]\nrun:\n" +
			"  - command: /instruct $1\n  - prompt: \"Summarize $ARGUMENTS\"\n",
	})
	loop := makeTestLoop(nil)
	repl := NewREPL(context.Background(), loop)
	defer repl.cancel()
	var sent []string
	repl.runFn = func(_ context.Context, msg string) (string, error) {
		sent = append(sent, msg)
		return "done", nil
	}

	before := loop.Session().TurnCount()
	out := testCaptureStdout(t, func() {
		require.NoError(t, repl.dispatchCommand("/b memory please"))
	})
	assert.Equal(t, before+1, loop.Session().TurnCount(), "the /instruct step briefed the model")
	assert.Equal(t, []string{"Summarize memory please"}, sent)
	assert.Contains(t, out, "done")
	assert.True(t, slices.Contains(commandNames(), "/b"), "aliases complete too")
	assert.True(t, slices.Contains(repl.allCommandNames(), "/brief"))
}

func TestDeclaredCommand_StopsAtTheFirstError(t *testing.T) {
	withCommands(t, map[string]string{
		"fail": "name: fail\nusage: [{help: h}]\nrun: [{prompt: one}, {prompt: two}]\n",
	})
	repl := NewREPL(context.Background(), makeTestLoop(nil))
	defer repl.cancel()
	calls := 0
	repl.runFn = func(_ context.Context, _ string) (string, error) {
		calls++
		return "", errors.New("llm down")
	}
	err := repl.dispatchCommand("/fail")
	require.ErrorContains(t, err, "/fail: llm down")
	assert.Equal(t, 1, calls)
}

func TestDeclaredCommand_CommandStepError(t *testing.T) {
	withCommands(t, map[string]string{
		"outer": "name: outer\nusage: [{help: h}]\nrun: [{command: /inner}, {prompt: never}]\n",
		"inner": "name: inner\nusage: [{help: h}]\nrun: [{prompt: boom}]\n",
	})
	repl := NewREPL(context.Background(), makeTestLoop(nil))
	defer repl.cancel()
	var sent []string
	repl.runFn = func(_ context.Context, msg string) (string, error) {
		sent = append(sent, msg)
		return "", errors.New("llm down")
	}
	err := repl.dispatchCommand("/outer")
	require.ErrorContains(t, err, "/outer: /inner: llm down")
	assert.Equal(t, []string{"boom"}, sent, "the outer command stops after the failed step")
}

func TestDeclaredCommand_RecursionIsBounded(t *testing.T) {
	withCommands(t, map[string]string{
		"loop": "name: loop\nusage: [{help: h}]\nrun: [{command: /loop}]\n",
	})
	repl := NewREPL(context.Background(), makeTestLoop(nil))
	defer repl.cancel()
	err := repl.dispatchCommand("/loop")
	require.ErrorContains(t, err, "nested more than")
	assert.Zero(t, repl.commandDepth)
}

func TestShippedDeclaredCommands(t *testing.T) {
	repl := NewREPL(context.Background(), makeTestLoop(nil))
	defer repl.cancel()
	var sent string
	repl.runFn = func(_ context.Context, msg string) (string, error) {
		sent = msg
		return "", nil
	}
	_ = testCaptureStdout(t, func() {
		require.NoError(t, repl.dispatchCommand("/explain the asset store"))
	})
	assert.Contains(t, sent, "Explain how the asset store works")

	_ = testCaptureStdout(t, func() {
		require.NoError(t, repl.dispatchCommand("/review error handling"))
	})
	assert.Contains(t, sent, "Extra focus, if given: error handling")
}
