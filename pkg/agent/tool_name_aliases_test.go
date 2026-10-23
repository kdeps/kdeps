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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

func TestAliasesOf(t *testing.T) {
	defs := map[string]toolDefinition{
		"bash_exec": {Aliases: []string{"bash", "Shell", "read_file"}},
		"read_file": {Aliases: []string{"cat", "shared"}},
		"other":     {Aliases: []string{"shared"}},
		"notepad":   {Name: "edit_file", Aliases: []string{"done"}},
	}
	got := aliasesOf(defs)
	assert.Equal(t, "bash_exec", got["bash"])
	assert.Equal(t, "bash_exec", got["shell"], "aliases are case-insensitive")
	assert.NotContains(t, got, "read_file", "a real tool's name is never an alias")
	assert.Equal(t, "read_file", got["cat"])
	assert.NotContains(t, got, "shared", "an alias two tools claim goes to neither")
	assert.Equal(t, "edit_file", got["done"], "an alias calls the name the tool registers under")
}

func TestResolveToolName(t *testing.T) {
	orig := toolAliases
	toolAliases = map[string]string{"bash": "bash_exec", "gone": "missing_tool"}
	t.Cleanup(func() { toolAliases = orig })
	has := func(n string) bool { return n == "bash_exec" || n == "bash"+"_real" }
	assert.Equal(t, "bash_exec", resolveToolName("bash_exec", has))
	assert.Equal(t, "bash_exec", resolveToolName("BASH", has))
	assert.Equal(t, "gone", resolveToolName("gone", has), "an alias of a tool that is not registered stays unknown")
	assert.Equal(t, "nope", resolveToolName("nope", has))
}

func TestShippedToolAliases_AreValidAndUnclaimedTwice(t *testing.T) {
	defs := loadToolDefinitionsFrom(os.DirFS(filepath.Join("..", "..", "assets")).(themeFS))
	got := aliasesOf(defs)
	for id, d := range defs {
		for _, a := range d.Aliases {
			assert.Regexp(t, toolNameRe, a, "%s alias %q", id, a)
			assert.Equal(t, toolNameOf(id, d), got[strings.ToLower(a)], "%s alias %q resolves to it", id, a)
		}
	}
	assert.Equal(t, "bash_exec", got["bash"])
	assert.Equal(t, "search_local", got["grep"])
	assert.Equal(t, "read_file", got["cat"])
}

func TestValidateToolDefinition_RejectsABadAlias(t *testing.T) {
	err := validateToolDefinition([]byte("version: 1.0.0\ndescription: x\naliases: [\"two words\"]\n"), "x")
	assert.ErrorContains(t, err, "is not a tool name")
	assert.NoError(t, validateToolDefinition([]byte("version: 1.0.0\ndescription: x\naliases: [bash]\n"), "x"))
}

func TestRunStreaming_AnAliasRunsItsTool(t *testing.T) {
	isolateEventsAndPresetsHome(t)
	t.Chdir(t.TempDir())
	reg := tools.NewRegistry()
	RegisterBuiltinTools(t.Context(), reg)
	ms := &cfgRecordingStreamer{inner: mockStreamer{responses: []mockStreamResponse{
		{
			toolCalls: []domain.StreamedToolCall{
				{ID: "c1", Name: "bash", Arguments: `{"cmd":"echo alias-ran-bash_exec"}`},
			},
		},
		{content: "done"},
	}}}
	loop := New(
		executor.NewEngine(nil),
		newTestWorkflowForSession(),
		reg,
		Config{Model: "test", Streamer: ms, MaxToolRounds: 5},
	)
	var buf bytes.Buffer
	_, err := loop.RunStreaming(context.Background(), "run it", &buf)
	require.NoError(t, err)
	require.Len(t, ms.cfgs, 2)
	sent := ms.cfgs[1].Messages + ms.cfgs[1].Prompt
	assert.Contains(t, sent, "alias-ran-bash_exec", "the bash call ran bash_exec, its cmd argument mapped to command")
	assert.NotContains(t, sent, "not found")
}
