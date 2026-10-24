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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/executor/llm"
	"github.com/kdeps/kdeps/v2/pkg/knowledge"
)

// contextOps are the session_ops operations behind /prompt and /search; the
// context window is the context setting.
//
//nolint:gochecknoglobals // fixed operation table
var contextOps = map[string]func(r *REPL, args map[string]any) (map[string]any, error){
	"last_request":   (*REPL).lastRequestOp,
	"search_index":   (*REPL).searchIndexOp,
	"search_folders": (*REPL).searchFoldersOp,
	"search_folder":  (*REPL).searchFolderOp,
}

// lastRequestOp is the last request sent to the LLM: each message and tool
// with its local token estimate, or (args.raw) the request as JSON.
func (r *REPL) lastRequestOp(args map[string]any) (map[string]any, error) {
	messages, tools := r.loop.LastSentMessages(), r.loop.LastSentTools()
	if len(messages) == 0 {
		return map[string]any{"messages": []any{}}, nil
	}
	if stringArgValue(args, "raw") == "true" {
		raw, err := json.MarshalIndent(map[string]any{"messages": messages, "tools": tools}, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal last sent request: %w", err)
		}
		return map[string]any{"raw": string(raw), "messages": messages}, nil
	}
	model := r.loop.config.Model
	total := 0
	msgs := make([]map[string]any, 0, len(messages))
	for i, m := range messages {
		content := promptMessageContentPreview(m)
		n := llm.CountTokens(model, content)
		total += n
		role, _ := m["role"].(string)
		if role == "" {
			role = "?"
		}
		msgs = append(msgs, map[string]any{"index": i, "role": strings.ToUpper(role), "tokens": n, "content": content})
	}
	toolTotal := 0
	list := make([]map[string]any, 0, len(tools))
	for _, tl := range tools {
		n := promptToolTokenCount(model, tl)
		toolTotal += n
		list = append(list, map[string]any{"name": tl.Name, "tokens": n, "description": tl.Description})
	}
	return map[string]any{
		"messages": msgs, "tools": list, "tool_tokens": toolTotal, "tokens": total + toolTotal,
	}, nil
}

// searchIndexOp brings the working project's folder indexes up to date.
func (r *REPL) searchIndexOp(_ map[string]any) (map[string]any, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("search: getwd failed: %w", err)
	}
	var out strings.Builder
	indexProjectFolders(&out, r.loop.knowledge, wd)
	return map[string]any{"output": strings.TrimSpace(out.String())}, nil
}

// searchFoldersOp lists the folders the working project searches: its own
// places and its search folders, each with whether it is indexed yet.
func (r *REPL) searchFoldersOp(_ map[string]any) (map[string]any, error) {
	k := r.loop.knowledge
	if k == nil {
		return nil, errNoVault
	}
	card, err := k.project()
	if err != nil {
		return nil, err
	}
	row := func(p string) map[string]any {
		return map[string]any{"path": p, "indexed": indexNote(knowledge.ExpandPath(p, k.vault.Root)) == ""}
	}
	places, folders := []map[string]any{}, []map[string]any{}
	for _, p := range card.Places() {
		places = append(places, row(p))
	}
	for _, p := range card.SearchFolders() {
		folders = append(folders, row(p))
	}
	return map[string]any{"project": card.Name, "places": places, "folders": folders}, nil
}

// searchFolderOp adds a folder to the working project's search folders, or
// removes one (args.action add or remove).
func (r *REPL) searchFolderOp(args map[string]any) (map[string]any, error) {
	action, folder := stringArgValue(args, "action"), strings.TrimSpace(stringArgValue(args, "folder"))
	if folder == "" {
		return nil, fmt.Errorf("give a folder: /search %s <folder>", action)
	}
	if action != "add" && action != "remove" {
		return nil, errors.New("action is add or remove")
	}
	var out strings.Builder
	if err := changeProjectFolders(&out, r.loop.knowledge, action, folder); err != nil {
		return nil, err
	}
	return map[string]any{"output": strings.TrimSpace(out.String())}, nil
}
