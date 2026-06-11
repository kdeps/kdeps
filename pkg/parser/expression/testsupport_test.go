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

package expression_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/namespace"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

func makeNamespaceAPI() *domain.UnifiedAPI {
	configMap := map[string]any{
		"llm": map[string]any{
			"openai_api_key": "sk-test",
			"model":          "llama3.2",
		},
		"defaults": map[string]any{
			"timezone": "UTC",
		},
	}
	workflowMap := map[string]any{
		"metadata": map[string]any{
			"name":    "my-wf",
			"version": "1.0",
		},
	}

	return &domain.UnifiedAPI{
		Get: func(_ string, _ ...string) (interface{}, error) {
			return nil, errors.New("not found")
		},
		Set: func(_ string, _ interface{}, _ ...string) error { return nil },
		GetConfigField: func(fullPath string) (any, error) {
			switch fullPath {
			case "config.llm.openai_api_key":
				return "sk-test", nil
			case "workflow.metadata.name":
				return "my-wf", nil
			default:
				return nil, errors.New("not found: " + fullPath)
			}
		},
		SetConfigField: func(fullPath string, _ any) error {
			if fullPath == "config.llm.openai_api_key" || fullPath == "workflow.metadata.name" {
				return nil
			}
			return errors.New("not found: " + fullPath)
		},
		ConfigNamespace: func(ns string) map[string]any {
			switch ns {
			case namespace.Config:
				return configMap
			case namespace.Workflow:
				return workflowMap
			default:
				return nil
			}
		},
	}
}

// storageBackend simulates the full set of get() storage type backends.
type storageBackend struct {
	memory    map[string]interface{}
	session   map[string]interface{}
	outputs   map[string]interface{}
	params    map[string]interface{} // query / param
	headers   map[string]interface{}
	files     map[string]interface{} // file content
	filepaths map[string]interface{} // filepath
	filetypes map[string]interface{} // filetype
	info      map[string]interface{}
	body      map[string]interface{}
	items     map[string]interface{}
	loops     map[string]interface{}
}

// buildGetFunc returns a Get function that routes by type hint.
func (b *storageBackend) buildGetFunc() func(string, ...string) (interface{}, error) {
	auto := func(name string) (interface{}, error) {
		// Mirrors getWithAutoDetection priority order:
		// Items → Memory → Session → Output → Param → Header → Info
		for _, store := range []map[string]interface{}{
			b.items, b.memory, b.session, b.outputs, b.params, b.headers, b.info,
		} {
			if store != nil {
				if v, ok := store[name]; ok {
					return v, nil
				}
			}
		}
		return nil, fmt.Errorf("key '%s' not found", name)
	}

	// Map each type hint to its backing store for O(1) dispatch.
	storeByHint := map[string]map[string]interface{}{
		"memory":   b.memory,
		"session":  b.session,
		"output":   b.outputs,
		"param":    b.params,
		"query":    b.params,
		"header":   b.headers,
		"file":     b.files,
		"filepath": b.filepaths,
		"filetype": b.filetypes,
		"info":     b.info,
		"data":     b.body,
		"body":     b.body,
		"item":     b.items,
		"loop":     b.loops,
	}

	return func(name string, hints ...string) (interface{}, error) {
		if len(hints) == 0 {
			return auto(name)
		}
		store, found := storeByHint[hints[0]]
		if !found {
			return nil, fmt.Errorf("unknown storage type: %s", hints[0])
		}
		if v, ok := store[name]; ok {
			return v, nil
		}
		return nil, fmt.Errorf("%s key %q not found", hints[0], name)
	}
}

func newBackend() *storageBackend {
	return &storageBackend{
		memory:    make(map[string]interface{}),
		session:   make(map[string]interface{}),
		outputs:   make(map[string]interface{}),
		params:    make(map[string]interface{}),
		headers:   make(map[string]interface{}),
		files:     make(map[string]interface{}),
		filepaths: make(map[string]interface{}),
		filetypes: make(map[string]interface{}),
		info:      make(map[string]interface{}),
		body:      make(map[string]interface{}),
		items:     make(map[string]interface{}),
		loops:     make(map[string]interface{}),
	}
}

func newEvaluatorWithBackend(b *storageBackend) *expression.Evaluator {
	api := &domain.UnifiedAPI{
		Get: b.buildGetFunc(),
	}
	return expression.NewEvaluator(api)
}

func evalDirect(t *testing.T, ev *expression.Evaluator, raw string) interface{} {
	t.Helper()
	expr := &domain.Expression{Raw: raw, Type: domain.ExprTypeDirect}
	result, err := ev.Evaluate(expr, map[string]interface{}{})
	require.NoError(t, err)
	return result
}

func evalInterpolated(t *testing.T, ev *expression.Evaluator, raw string) interface{} {
	t.Helper()
	expr := &domain.Expression{Raw: raw, Type: domain.ExprTypeInterpolated}
	result, err := ev.Evaluate(expr, map[string]interface{}{})
	require.NoError(t, err)
	return result
}

func TestGet_AutoDetect_FromMemory(t *testing.T) {
	b := newBackend()
	b.memory["username"] = "alice"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "alice", evalDirect(t, ev, "get('username')"))
}

func TestGet_AutoDetect_FromSession(t *testing.T) {
	b := newBackend()
	b.session["token"] = "sess-abc"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "sess-abc", evalDirect(t, ev, "get('token')"))
}

func TestGet_AutoDetect_FromOutput(t *testing.T) {
	b := newBackend()
	b.outputs["llm-step"] = map[string]interface{}{"text": "hello"}
	ev := newEvaluatorWithBackend(b)
	result := evalDirect(t, ev, "get('llm-step')")
	assert.Equal(t, map[string]interface{}{"text": "hello"}, result)
}

func TestGet_AutoDetect_FromParam(t *testing.T) {
	b := newBackend()
	b.params["name"] = "Alice"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "Alice", evalDirect(t, ev, "get('name')"))
}

func TestGet_AutoDetect_FromHeader(t *testing.T) {
	b := newBackend()
	b.headers["x-api-key"] = "key-123"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "key-123", evalDirect(t, ev, "get('x-api-key')"))
}

func TestGet_AutoDetect_FromInfo(t *testing.T) {
	b := newBackend()
	b.info["requestID"] = "req-xyz"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "req-xyz", evalDirect(t, ev, "get('requestID')"))
}

func TestGet_AutoDetect_PriorityOrder_ItemBeforeMemory(t *testing.T) {
	b := newBackend()
	b.items["key"] = "from-item"
	b.memory["key"] = "from-memory"
	ev := newEvaluatorWithBackend(b)
	// Items take priority over memory in auto-detection.
	assert.Equal(t, "from-item", evalDirect(t, ev, "get('key')"))
}

func TestGet_AutoDetect_PriorityOrder_MemoryBeforeSession(t *testing.T) {
	b := newBackend()
	b.memory["key"] = "from-memory"
	b.session["key"] = "from-session"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "from-memory", evalDirect(t, ev, "get('key')"))
}

func TestGet_AutoDetect_PriorityOrder_SessionBeforeOutput(t *testing.T) {
	b := newBackend()
	b.session["key"] = "from-session"
	b.outputs["key"] = "from-output"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "from-session", evalDirect(t, ev, "get('key')"))
}

func TestGet_AutoDetect_PriorityOrder_OutputBeforeParam(t *testing.T) {
	b := newBackend()
	b.outputs["key"] = "from-output"
	b.params["key"] = "from-param"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "from-output", evalDirect(t, ev, "get('key')"))
}

func TestGet_AutoDetect_PriorityOrder_ParamBeforeHeader(t *testing.T) {
	b := newBackend()
	b.params["key"] = "from-param"
	b.headers["key"] = "from-header"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "from-param", evalDirect(t, ev, "get('key')"))
}

func TestGet_AutoDetect_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('nonexistent')"))
}

func TestGet_TypeHint_Memory_Found(t *testing.T) {
	b := newBackend()
	b.memory["counter"] = 42
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, 42, evalDirect(t, ev, "get('counter', 'memory')"))
}

func TestGet_TypeHint_Memory_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('missing', 'memory')"))
}

func TestGet_TypeHint_Memory_DoesNotFallThroughToParam(t *testing.T) {
	b := newBackend()
	b.params["key"] = "in-param"
	// Key is only in params, not memory — should be nil, not "in-param".
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('key', 'memory')"))
}

func TestGet_TypeHint_Session_Found(t *testing.T) {
	b := newBackend()
	b.session["cart"] = []string{"item1", "item2"}
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, []string{"item1", "item2"}, evalDirect(t, ev, "get('cart', 'session')"))
}

func TestGet_TypeHint_Session_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('cart', 'session')"))
}

func TestGet_TypeHint_Session_DoesNotFallThroughToMemory(t *testing.T) {
	b := newBackend()
	b.memory["key"] = "in-memory"
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('key', 'session')"))
}

func TestGet_TypeHint_Output_Found(t *testing.T) {
	b := newBackend()
	b.outputs["summarize"] = map[string]interface{}{"summary": "short text"}
	ev := newEvaluatorWithBackend(b)
	result := evalDirect(t, ev, "get('summarize', 'output')")
	assert.Equal(t, map[string]interface{}{"summary": "short text"}, result)
}

func TestGet_TypeHint_Output_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('missing-step', 'output')"))
}

func TestGet_TypeHint_Param_QueryParam(t *testing.T) {
	b := newBackend()
	b.params["name"] = "Alice"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "Alice", evalDirect(t, ev, "get('name', 'param')"))
}

func TestGet_TypeHint_Param_MultipleParams(t *testing.T) {
	b := newBackend()
	b.params["page"] = "2"
	b.params["limit"] = "20"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "2", evalDirect(t, ev, "get('page', 'param')"))
	assert.Equal(t, "20", evalDirect(t, ev, "get('limit', 'param')"))
}

func TestGet_TypeHint_Param_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('q', 'param')"))
}

func TestGet_TypeHint_Query_Found(t *testing.T) {
	b := newBackend()
	b.params["search"] = "golang"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "golang", evalDirect(t, ev, "get('search', 'query')"))
}

func TestGet_TypeHint_Query_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('search', 'query')"))
}

func TestGet_TypeHint_Query_SameResultAsParam(t *testing.T) {
	b := newBackend()
	b.params["q"] = "test"
	ev := newEvaluatorWithBackend(b)
	// "query" and "param" must return the same value.
	assert.Equal(t,
		evalDirect(t, ev, "get('q', 'param')"),
		evalDirect(t, ev, "get('q', 'query')"),
	)
}

func TestGet_TypeHint_Header_Found(t *testing.T) {
	b := newBackend()
	b.headers["Authorization"] = "Bearer tok"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "Bearer tok", evalDirect(t, ev, "get('Authorization', 'header')"))
}

func TestGet_TypeHint_Header_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('X-Custom', 'header')"))
}

func TestGet_TypeHint_Header_DoesNotFallThroughToParam(t *testing.T) {
	b := newBackend()
	b.params["Authorization"] = "in-param"
	ev := newEvaluatorWithBackend(b)
	// Header lookup must not fall back to params.
	assert.Nil(t, evalDirect(t, ev, "get('Authorization', 'header')"))
}

func TestGet_TypeHint_File_Found(t *testing.T) {
	b := newBackend()
	b.files["report.txt"] = "line1\nline2"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "line1\nline2", evalDirect(t, ev, "get('report.txt', 'file')"))
}

func TestGet_TypeHint_File_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('missing.txt', 'file')"))
}

func TestGet_TypeHint_Filepath_Found(t *testing.T) {
	b := newBackend()
	b.filepaths["upload"] = "/tmp/uploads/abc.pdf"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "/tmp/uploads/abc.pdf", evalDirect(t, ev, "get('upload', 'filepath')"))
}

func TestGet_TypeHint_Filepath_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('upload', 'filepath')"))
}

func TestGet_TypeHint_Filetype_Found(t *testing.T) {
	b := newBackend()
	b.filetypes["upload"] = "application/pdf"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "application/pdf", evalDirect(t, ev, "get('upload', 'filetype')"))
}

func TestGet_TypeHint_Filetype_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('upload', 'filetype')"))
}

func TestGet_TypeHint_Info_Found(t *testing.T) {
	b := newBackend()
	b.info["requestID"] = "req-42"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "req-42", evalDirect(t, ev, "get('requestID', 'info')"))
}

func TestGet_TypeHint_Info_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('missing', 'info')"))
}

func TestGet_TypeHint_Data_Found(t *testing.T) {
	b := newBackend()
	b.body["email"] = "user@example.com"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "user@example.com", evalDirect(t, ev, "get('email', 'data')"))
}

func TestGet_TypeHint_Body_Found(t *testing.T) {
	b := newBackend()
	b.body["message"] = "hello"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "hello", evalDirect(t, ev, "get('message', 'body')"))
}

func TestGet_TypeHint_DataAndBody_SameResult(t *testing.T) {
	b := newBackend()
	b.body["key"] = "value"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t,
		evalDirect(t, ev, "get('key', 'data')"),
		evalDirect(t, ev, "get('key', 'body')"),
	)
}

func TestGet_TypeHint_Data_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('field', 'data')"))
}

func TestGet_TypeHint_Item_Found(t *testing.T) {
	b := newBackend()
	b.items["current"] = map[string]interface{}{"id": 1, "name": "widget"}
	ev := newEvaluatorWithBackend(b)
	result := evalDirect(t, ev, "get('current', 'item')")
	assert.Equal(t, map[string]interface{}{"id": 1, "name": "widget"}, result)
}

func TestGet_TypeHint_Item_IndexFound(t *testing.T) {
	b := newBackend()
	b.items["index"] = 3
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, 3, evalDirect(t, ev, "get('index', 'item')"))
}

func TestGet_TypeHint_Item_CountFound(t *testing.T) {
	b := newBackend()
	b.items["count"] = 10
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, 10, evalDirect(t, ev, "get('count', 'item')"))
}

func TestGet_TypeHint_Item_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('current', 'item')"))
}

func TestGet_TypeHint_Loop_IndexFound(t *testing.T) {
	b := newBackend()
	b.loops["index"] = 0
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, 0, evalDirect(t, ev, "get('index', 'loop')"))
}

func TestGet_TypeHint_Loop_CountFound(t *testing.T) {
	b := newBackend()
	b.loops["count"] = 5
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, 5, evalDirect(t, ev, "get('count', 'loop')"))
}

func TestGet_TypeHint_Loop_ResultsFound(t *testing.T) {
	b := newBackend()
	b.loops["results"] = []interface{}{"a", "b", "c"}
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, []interface{}{"a", "b", "c"}, evalDirect(t, ev, "get('results', 'loop')"))
}

func TestGet_TypeHint_Loop_NotFound_ReturnsNil(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Nil(t, evalDirect(t, ev, "get('index', 'loop')"))
}

func TestGet_DefaultValue_KeyFound_DefaultIgnored(t *testing.T) {
	// The original bug scenario: query param exists → should return actual value, not default.
	b := newBackend()
	b.params["name"] = "Alice"
	ev := newEvaluatorWithBackend(b)
	// "World" is not a type hint — auto-detect, key exists → return "Alice".
	assert.Equal(t, "Alice", evalDirect(t, ev, "get('name', 'World')"))
}

func TestGet_DefaultValue_KeyNotFound_ReturnsDefault(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	// Key not in any backend → return the default "World".
	assert.Equal(t, "World", evalDirect(t, ev, "get('name', 'World')"))
}

func TestGet_DefaultValue_EmptyStringDefault(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	// Empty string is a valid default (not a type hint).
	assert.Equal(t, "", evalDirect(t, ev, "get('missing', '')"))
}

func TestGet_DefaultValue_NumericLookingStringDefault(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "0", evalDirect(t, ev, "get('missing', '0')"))
}

func TestGet_DefaultValue_InMemoryBeatsDefault(t *testing.T) {
	b := newBackend()
	b.memory["setting"] = "stored"
	ev := newEvaluatorWithBackend(b)
	// Found in memory → return stored value, not the default.
	assert.Equal(t, "stored", evalDirect(t, ev, "get('setting', 'fallback')"))
}

func TestGet_DefaultValue_InSessionBeatsDefault(t *testing.T) {
	b := newBackend()
	b.session["locale"] = "fr"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "fr", evalDirect(t, ev, "get('locale', 'en')"))
}

func TestGet_DefaultValue_InHeaderBeatsDefault(t *testing.T) {
	b := newBackend()
	b.headers["Accept-Language"] = "de"
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, "de", evalDirect(t, ev, "get('Accept-Language', 'en')"))
}

func TestGet_DefaultValue_NotATypeHint_UsesAutoDetect(t *testing.T) {
	// Ensure that a non-hint second arg triggers auto-detection (not a direct type lookup).
	b := newBackend()
	b.memory["greet"] = "hi"    // exists in memory
	b.params["greet"] = "hello" // also in params (lower priority)
	ev := newEvaluatorWithBackend(b)
	// Auto-detect should find memory first.
	assert.Equal(t, "hi", evalDirect(t, ev, "get('greet', 'DEFAULT')"))
}

func TestGet_TypeHint_Recognized_NeverTreatedAsDefault(t *testing.T) {
	recognizedHints := []string{
		"memory", "session", "output", "param", "query",
		"header", "file", "filepath", "filetype", "info",
		"data", "body", "item", "loop",
	}
	b := newBackend()
	// Put a value in params for "key"
	b.params["key"] = "param-value"
	ev := newEvaluatorWithBackend(b)

	for _, hint := range recognizedHints {
		t.Run(hint, func(t *testing.T) {
			// For "param" and "query" the value exists → not nil.
			// For all others the value is absent → nil (not the hint string itself).
			result := evalDirect(t, ev, fmt.Sprintf("get('key', '%s')", hint))
			if hint == "param" || hint == "query" {
				assert.Equal(t, "param-value", result, "hint=%s", hint)
			} else {
				assert.Nil(t, result, "hint=%s should return nil when key absent, not the hint string", hint)
			}
		})
	}
}

func TestGet_Interpolated_KeyFound(t *testing.T) {
	b := newBackend()
	b.params["name"] = "Alice"
	ev := newEvaluatorWithBackend(b)
	result := evalInterpolated(t, ev, "Hello, {{ get('name') }}!")
	assert.Equal(t, "Hello, Alice!", result)
}

func TestGet_Interpolated_DefaultValue_KeyMissing(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	// The original bug: get('name', 'World') inside a template should use default.
	result := evalInterpolated(t, ev, "Hello, {{ get('name', 'World') }}!")
	assert.Equal(t, "Hello, World!", result)
}

func TestGet_Interpolated_DefaultValue_KeyPresent(t *testing.T) {
	b := newBackend()
	b.params["name"] = "Bob"
	ev := newEvaluatorWithBackend(b)
	result := evalInterpolated(t, ev, "Hello, {{ get('name', 'World') }}!")
	assert.Equal(t, "Hello, Bob!", result)
}

func TestGet_Interpolated_TypeHint_Param(t *testing.T) {
	b := newBackend()
	b.params["q"] = "golang"
	ev := newEvaluatorWithBackend(b)
	result := evalInterpolated(t, ev, "Search: {{ get('q', 'param') }}")
	assert.Equal(t, "Search: golang", result)
}

func TestGet_Interpolated_MultipleGetCalls(t *testing.T) {
	b := newBackend()
	b.params["first"] = "John"
	b.params["last"] = "Doe"
	ev := newEvaluatorWithBackend(b)
	// Leading text forces the multi-interpolation path (not the single-block fast path).
	result := evalInterpolated(t, ev, "Name: {{ get('first') }} {{ get('last') }}")
	assert.Equal(t, "Name: John Doe", result)
}

func TestGet_Interpolated_MixGetAndLiteral(t *testing.T) {
	b := newBackend()
	b.memory["count"] = "3"
	ev := newEvaluatorWithBackend(b)
	result := evalInterpolated(t, ev, "You have {{ get('count') }} messages")
	assert.Equal(t, "You have 3 messages", result)
}

func TestGet_Interpolated_DefaultInAllMissing(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	// Leading text forces the multi-interpolation path.
	result := evalInterpolated(t, ev,
		"Hello: {{ get('first', 'Guest') }} {{ get('last', 'User') }}")
	assert.Equal(t, "Hello: Guest User", result)
}

func TestGet_StructuredValue_Memory(t *testing.T) {
	b := newBackend()
	b.memory["user"] = map[string]interface{}{"name": "Alice", "age": 30}
	ev := newEvaluatorWithBackend(b)
	result := evalDirect(t, ev, "get('user', 'memory')")
	assert.Equal(t, map[string]interface{}{"name": "Alice", "age": 30}, result)
}

func TestGet_StructuredValue_DefaultUnused(t *testing.T) {
	b := newBackend()
	b.memory["config"] = map[string]interface{}{"debug": true}
	ev := newEvaluatorWithBackend(b)
	// Structured value found → default is ignored.
	result := evalDirect(t, ev, "get('config', 'none')")
	assert.Equal(t, map[string]interface{}{"debug": true}, result)
}

func TestGet_BooleanValue_Memory(t *testing.T) {
	b := newBackend()
	b.memory["flag"] = true
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, true, evalDirect(t, ev, "get('flag', 'memory')"))
}

func TestGet_IntValue_Param(t *testing.T) {
	b := newBackend()
	b.params["page"] = 2
	ev := newEvaluatorWithBackend(b)
	assert.Equal(t, 2, evalDirect(t, ev, "get('page', 'param')"))
}

func TestGet_UsedInCondition_KeyFound(t *testing.T) {
	b := newBackend()
	b.params["mode"] = "admin"
	ev := newEvaluatorWithBackend(b)
	result := evalDirect(t, ev, "get('mode', 'param') == 'admin'")
	assert.Equal(t, true, result)
}

func TestGet_UsedInCondition_DefaultFallback(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	// Default kicks in → 'guest' != 'admin'
	result := evalDirect(t, ev, "get('mode', 'guest') == 'admin'")
	assert.Equal(t, false, result)
}

func TestGet_DefaultValueEqualsDefaultCondition(t *testing.T) {
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	result := evalDirect(t, ev, "get('role', 'guest') == 'guest'")
	assert.Equal(t, true, result)
}

func TestGet_TypeHint_InvalidHint_IsNotTreatedAsDefault(t *testing.T) {
	// A recognized type hint that produces no result must return nil, not the hint string.
	b := newBackend()
	ev := newEvaluatorWithBackend(b)
	// "memory" is valid hint, key absent → nil (not "memory").
	assert.Nil(t, evalDirect(t, ev, "get('key', 'memory')"))
}

func TestInterpolatedExpressions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		env      map[string]interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "simple variable",
			template: "{{q}}",
			env:      map[string]interface{}{"q": "hello world"},
			expected: "hello world",
		},
		{
			name:     "simple variable with spaces",
			template: "{{ q }}",
			env:      map[string]interface{}{"q": "hello world"},
			expected: "hello world",
		},
		{
			name:     "simple variable with text",
			template: "Query: {{q}}",
			env:      map[string]interface{}{"q": "test query"},
			expected: "Query: test query",
		},
		{
			name:     "multiple variables",
			template: "Hello {{name}}, you scored {{score}}",
			env:      map[string]interface{}{"name": "Alice", "score": 95},
			expected: "Hello Alice, you scored 95",
		},
		{
			name:     "nested object",
			template: "{{user.name}}",
			env: map[string]interface{}{
				"user": map[string]interface{}{"name": "Bob", "age": 30},
			},
			expected: "Bob",
		},
		{
			name:     "missing variable returns empty",
			template: "{{missing}}",
			env:      map[string]interface{}{},
			expected: "",
		},
		{
			name:     "integer value",
			template: "{{count}}",
			env:      map[string]interface{}{"count": 42},
			expected: 42,
		},
		{
			name:     "boolean value",
			template: "{{active}}",
			env:      map[string]interface{}{"active": true},
			expected: true,
		},
		{
			name:     "mixed interpolation and expr-lang",
			template: "Hello {{name}}, time is {{ info('current_time') }}",
			env:      map[string]interface{}{"name": "Alice"},
			expected: "", // Special case - checked separately
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateInterpolatedTemplate(t, tt.template, tt.env)
			validateInterpolatedResult(t, tt.name, result, tt.expected, tt.wantErr)
		})
	}
}

// evaluateInterpolatedTemplate parses and evaluates an interpolated template.
func evaluateInterpolatedTemplate(
	t *testing.T,
	template string,
	env map[string]interface{},
) interface{} {
	t.Helper()
	parser := expression.NewParser()
	expr, err := parser.Parse(template)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// All interpolated expressions are ExprTypeInterpolated
	if expr.Type != domain.ExprTypeInterpolated {
		t.Errorf("Expected ExprTypeInterpolated, got %v", expr.Type)
	}

	evaluator := expression.NewEvaluator(createMockAPI())
	result, err := evaluator.Evaluate(expr, env)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	return result
}

// validateInterpolatedResult validates the result of an interpolated expression.
func validateInterpolatedResult(
	t *testing.T,
	testName string,
	result, expected interface{},
	_ bool,
) {
	t.Helper()

	// Special handling for mixed expression test
	if testName == "mixed interpolation and expr-lang" {
		resultStr, ok := result.(string)
		if !ok {
			t.Errorf("Expected string result, got %T", result)
			return
		}
		if !strings.Contains(resultStr, "Alice") {
			t.Errorf("Result should contain 'Alice', got: %v", resultStr)
		}
		return
	}

	if result != expected {
		t.Errorf("Evaluate() = %v, want %v", result, expected)
	}
}
