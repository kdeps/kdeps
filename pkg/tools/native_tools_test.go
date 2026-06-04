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

package tools

import "testing"

func TestNativeToolDefs_TotalCount(t *testing.T) {
	tools := NativeToolDefs()
	if len(tools) != 9 {
		t.Fatalf("expected 9 native tools, got %d", len(tools))
	}
}

func TestNativeToolDefs_Names(t *testing.T) {
	tools := NativeToolDefs()
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.Name] = true
	}
	expected := []string{
		"kdeps_python", "kdeps_exec", "kdeps_sql",
		"kdeps_embedding", "kdeps_search_local",
		"kdeps_http", "kdeps_scraper", "kdeps_search_web", "kdeps_browser",
	}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing tool: %s", name)
		}
	}
}

func TestNativeToolDefs_DescriptionsAndParams(t *testing.T) {
	tools := NativeToolDefs()
	for _, tool := range tools {
		if tool.Name == "" {
			t.Error("every native tool must have a non-empty Name")
		}
		if tool.Description == "" {
			t.Errorf("tool %q must have a non-empty Description", tool.Name)
		}
		if tool.Parameters == nil {
			t.Errorf("tool %q must have Parameters", tool.Name)
		}
	}
}

func TestNativeToolDefs_RequiredParams(t *testing.T) {
	python := getNativeTool(t, "kdeps_python")
	if !python.Parameters["script"].Required {
		t.Error("kdeps_python 'script' must be required")
	}

	execTool := getNativeTool(t, "kdeps_exec")
	if !execTool.Parameters["command"].Required {
		t.Error("kdeps_exec 'command' must be required")
	}

	sql := getNativeTool(t, "kdeps_sql")
	if !sql.Parameters["query"].Required {
		t.Error("kdeps_sql 'query' must be required")
	}

	http := getNativeTool(t, "kdeps_http")
	if !http.Parameters["url"].Required {
		t.Error("kdeps_http 'url' must be required")
	}

	embed := getNativeTool(t, "kdeps_embedding")
	if !embed.Parameters["operation"].Required {
		t.Error("kdeps_embedding 'operation' must be required")
	}

	search := getNativeTool(t, "kdeps_search_local")
	if !search.Parameters["path"].Required {
		t.Error("kdeps_search_local 'path' must be required")
	}

	webSearch := getNativeTool(t, "kdeps_search_web")
	if !webSearch.Parameters["query"].Required {
		t.Error("kdeps_search_web 'query' must be required")
	}

	browser := getNativeTool(t, "kdeps_browser")
	if !browser.Parameters["url"].Required {
		t.Error("kdeps_browser 'url' must be required")
	}

	scraper := getNativeTool(t, "kdeps_scraper")
	if !scraper.Parameters["url"].Required {
		t.Error("kdeps_scraper 'url' must be required")
	}
}

func getNativeTool(t *testing.T, name string) *Tool {
	t.Helper()
	tools := NativeToolDefs()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %q not found", name)
	return nil
}

func TestNativeToolDefs_KdepsPythonOptionalTimeout(t *testing.T) {
	python := getNativeTool(t, "kdeps_python")
	if p, ok := python.Parameters["timeout"]; ok {
		if p.Required {
			t.Error("kdeps_python 'timeout' should be optional")
		}
	} else {
		t.Error("kdeps_python missing optional 'timeout' param")
	}
}

func TestNativeToolDefs_EnumValues(t *testing.T) {
	embed := getNativeTool(t, "kdeps_embedding")
	if ops := embed.Parameters["operation"].Enum; len(ops) != 4 {
		t.Errorf("expected 4 enum values for embedding operation, got %d: %v", len(ops), ops)
	}

	webSearch := getNativeTool(t, "kdeps_search_web")
	if providers := webSearch.Parameters["provider"].Enum; len(providers) != 4 {
		t.Errorf("expected 4 enum values for search provider, got %d: %v", len(providers), providers)
	}

	browser := getNativeTool(t, "kdeps_browser")
	if actions := browser.Parameters["action"].Enum; len(actions) != 4 {
		t.Errorf("expected 4 enum values for browser action, got %d: %v", len(actions), actions)
	}
}
