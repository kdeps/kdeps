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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestCode_NilAndCanceled(t *testing.T) {
	var noCtx context.Context
	if _, err := CodeSearch(noCtx, map[string]any{"query": "q", "path": "p"}); err == nil ||
		err.Error() != "code_search: nil context" {
		t.Fatalf("nil: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := CodeIndexFolder(ctx, nil); err == nil || !strings.Contains(err.Error(), "canceled") {
		t.Fatalf("canceled: %v", err)
	}
}

func TestCode_RootPathAndRequiredQuery(t *testing.T) {
	t.Setenv(envAllowRoot, "")
	if _, err := CodeIndexFolder(t.Context(), map[string]any{"path": "/"}); err == nil ||
		!strings.Contains(err.Error(), "code_index_folder:") || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("root: %v", err)
	}
	dir := t.TempDir()
	if _, err := CodeSearch(t.Context(), map[string]any{"path": dir}); err == nil ||
		err.Error() != "codeIntelligence: query is required for symbolSearch" {
		t.Fatalf("query: %v", err)
	}
}

func TestCode_IndexAndGraph(t *testing.T) {
	root := t.TempDir()
	for name, body := range map[string]string{
		"a.md": "---\ntopics: [go]\n---\nSee [b](b.md).",
		"b.md": "---\ntopics: [go]\n---\nNo links.",
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(root)
	db := filepath.Join(t.TempDir(), "graph.db")
	out, err := CodeIndexFolder(t.Context(), map[string]any{
		argGraphDBPath: db,
		argExtensions:  []any{".md"},
	})
	if err != nil || !strings.Contains(out, `"filesIndexed": 2`) {
		t.Fatalf("index: %v %s", err, out)
	}
	// A path argument must not redirect indexFolder off the working directory.
	other := t.TempDir()
	out, err = CodeIndexFolder(t.Context(), map[string]any{
		"path":         other,
		argGraphDBPath: db,
		argExtensions:  []any{".md"},
	})
	if err != nil || !strings.Contains(out, `"filesIndexed": 2`) {
		t.Fatalf("index ignores path: %v %s", err, out)
	}
	out, err = CodeGraphFile(t.Context(), map[string]any{
		"path":         filepath.Join(root, "a.md"),
		argGraphDBPath: db,
	})
	if err != nil || !strings.Contains(out, "relatedByTopic") {
		t.Fatalf("graph file: %v %s", err, out)
	}
	out, err = CodeGraphTopic(t.Context(), map[string]any{argTopic: "go", argGraphDBPath: db})
	if err != nil || !strings.Contains(out, `"topic": "go"`) {
		t.Fatalf("graph topic: %v %s", err, out)
	}
	out, err = CodeGraphAll(t.Context(), map[string]any{argGraphDBPath: db})
	if err != nil || !strings.Contains(out, "roots") {
		t.Fatalf("graph all: %v %s", err, out)
	}
}

func TestCode_GraphRequiresFields(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("KDEPS_DATA_DIR", t.TempDir())
	if _, err := CodeGraphFile(t.Context(), map[string]any{}); err == nil ||
		err.Error() != "codeIntelligence: path is required for graphFile" {
		t.Fatalf("graph file: %v", err)
	}
	if _, err := CodeGraphTopic(t.Context(), map[string]any{}); err == nil ||
		err.Error() != "codeIntelligence: topic is required for graphTopic" {
		t.Fatalf("graph topic: %v", err)
	}
	if _, err := CodeSymbols(t.Context(), nil); err == nil {
		t.Fatal("symbols with no path must fail")
	}
	if _, err := CodeHover(t.Context(), map[string]any{"path": t.TempDir()}); err == nil {
		t.Fatal("hover with no symbol must fail")
	}
}

func TestCodeConfig_SearchOptions(t *testing.T) {
	dir := t.TempDir()
	config, err := codeConfig(toolCodeSearch, domain.CodeIntOpSymbolSearch, map[string]any{
		"path":        dir,
		"query":       "q",
		argPattern:    "*.go",
		argLanguage:   "go",
		argLanguageID: "go",
		argContext:    float64(2),
		argLimit:      "10",
	})
	if err != nil {
		t.Fatalf("codeConfig: %v", err)
	}
	if config.Pattern != "*.go" || config.Language != "go" || config.LanguageID != "go" {
		t.Fatalf("string options lost: %+v", config)
	}
	if config.Context != 2 || config.Limit != 10 {
		t.Fatalf("numeric options lost: %+v", config)
	}
}
