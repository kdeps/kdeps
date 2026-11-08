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

// Package subagent is the subagent tool shared by agent mode and workflow
// mode: a task handed to a fresh model run that has not seen the calling
// conversation and may call only read-only tools. Each mode supplies the run
// (agent mode: a side loop; workflow mode: a chat call).
package subagent

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// Name is the subagent tool's name.
const Name = "subagent"

// askQuestion acts on the caller's turn, so a subagent never gets it.
const askQuestion = "ask_question"

// Settings are the subagent's limits, prompt, tool policy and definition.
// Agent mode installs them at startup from the subagent-max-rounds event, the
// subagent-system harness, the permission policy and the subagent tool asset.
type Settings struct {
	// MaxRounds caps one subagent's tool rounds.
	MaxRounds func() int
	// SystemPrompt renders the subagent's instructions.
	SystemPrompt func(instructions string) string
	// ReadOnly reports whether a tool only reads.
	ReadOnly func(tool string) bool
	// Definition is the tool's model-facing description and parameters.
	Definition func() (domain.Tool, bool)
}

//nolint:gochecknoglobals // process-wide settings, installed once at startup
var (
	settingsMu sync.RWMutex
	settings   Settings
)

// Configure installs the subagent settings for both modes.
func Configure(s Settings) {
	settingsMu.Lock()
	settings = s
	settingsMu.Unlock()
}

func current() Settings {
	settingsMu.RLock()
	defer settingsMu.RUnlock()
	return settings
}

// MaxRounds is a subagent's tool-round cap; 0 when not configured.
func MaxRounds() int {
	if f := current().MaxRounds; f != nil {
		return f()
	}
	return 0
}

// SystemPrompt renders the subagent's instructions, or "" when not configured.
func SystemPrompt(instructions string) string {
	if f := current().SystemPrompt; f != nil {
		return f(strings.TrimSpace(instructions))
	}
	return ""
}

// Definition is the subagent tool's description and parameters.
func Definition() (domain.Tool, bool) {
	if f := current().Definition; f != nil {
		return f()
	}
	return domain.Tool{}, false
}

// Request is one subagent call's arguments.
type Request struct {
	Task         string
	Instructions string
	Tools        []string
}

// ParseArgs reads a subagent call's arguments; task is required.
func ParseArgs(args map[string]any) (Request, error) {
	task, _ := args["task"].(string)
	instructions, _ := args["instructions"].(string)
	r := Request{
		Task:         strings.TrimSpace(task),
		Instructions: strings.TrimSpace(instructions),
		Tools:        ToolNames(args["tools"]),
	}
	if r.Task == "" {
		return r, errors.New("subagent: task is required")
	}
	return r, nil
}

// ToolNames reads a tool list sent as "a, b", ["a","b"] or []string.
func ToolNames(v any) []string {
	var raw []string
	switch x := v.(type) {
	case string:
		raw = strings.FieldsFunc(x, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' })
	case []any:
		for _, e := range x {
			if s, ok := e.(string); ok {
				raw = append(raw, s)
			}
		}
	case []string:
		raw = x
	}
	out := make([]string, 0, len(raw))
	for _, n := range raw {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// Tools picks a subagent's tools from the caller's: every read-only one, or
// the named ones. A name that is not a read-only tool of the caller is an
// error, so a subagent never writes or runs commands.
func Tools(available, names []string) ([]string, error) {
	readOnly := current().ReadOnly
	var allowed []string
	for _, n := range available {
		if n != Name && n != askQuestion && readOnly != nil && readOnly(n) {
			allowed = append(allowed, n)
		}
	}
	if len(names) == 0 {
		return allowed, nil
	}
	var refused []string
	for _, n := range names {
		if !slices.Contains(allowed, n) {
			refused = append(refused, n)
		}
	}
	if len(refused) > 0 {
		return nil, fmt.Errorf("subagent: not a read-only tool here: %s", strings.Join(refused, ", "))
	}
	return names, nil
}
