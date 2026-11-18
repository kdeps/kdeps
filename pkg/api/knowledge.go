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

package api

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/kdeps/kdeps/v3/pkg/knowledge"
)

const toolKnowledgeRecall = "knowledge_recall"

// KnowledgeRecall is the knowledge vault's notes related to query, walked
// from the configured vault's link graph (no model involved), as agent
// mode's knowledge_recall gives them. A vault the config turns off refuses.
func KnowledgeRecall(ctx context.Context, args map[string]any) (string, error) {
	q, _ := args[argQuery].(string)
	if strings.TrimSpace(q) == "" {
		return "", errors.New("knowledge_recall: query is required")
	}
	vault, kc := knowledge.Configured()
	if kc.Enabled != nil && !*kc.Enabled {
		return "", errors.New("knowledge_recall: the knowledge vault is off")
	}
	none := knowledge.Prompt(knowledge.PromptRecallNone, nil)
	if _, err := os.Stat(vault); errors.Is(err, fs.ErrNotExist) {
		return none, nil // no vault yet: nothing to recall
	}
	if err := knowledge.Ensure(ctx, vault); err != nil {
		return "", err
	}
	if ix := knowledge.SharedIndex(knowledge.GraphDir(vault)); ix != nil {
		if text := knowledge.FormatRecall(ix.Recall(q, kc.Recall.Hops, kc.Recall.BudgetTokens), vault); text != "" {
			return text, nil
		}
	}
	return none, nil
}
