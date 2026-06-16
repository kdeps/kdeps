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

package jsonschema

import (
	"reflect"
	"testing"
)

type testInput struct {
	Query   string  `json:"query"   desc:"search query"    required:"true"`
	MaxRows int     `json:"maxRows" desc:"max result rows"`
	Score   float64 `json:"score"   desc:"min relevance"`
	Exact   bool    `json:"exact"   desc:"exact match"`
	Format  string  `json:"format"  desc:"output format"                   enum:"json,csv,table"`
	ignored string  //nolint:unused
}

func TestFromStruct_BasicFields(t *testing.T) {
	t.Parallel()
	params := FromStruct(testInput{})

	if len(params) != 5 {
		t.Fatalf("expected 5 params, got %d: %v", len(params), params)
	}

	q := params["query"]
	if q.Type != "string" {
		t.Errorf("query type = %q, want 'string'", q.Type)
	}
	if q.Description != "search query" {
		t.Errorf("query desc = %q, want 'search query'", q.Description)
	}
	if !q.Required {
		t.Error("query should be required")
	}

	m := params["maxRows"]
	if m.Type != "integer" {
		t.Errorf("maxRows type = %q, want 'integer'", m.Type)
	}

	s := params["score"]
	if s.Type != "number" {
		t.Errorf("score type = %q, want 'number'", s.Type)
	}

	e := params["exact"]
	if e.Type != "boolean" {
		t.Errorf("exact type = %q, want 'boolean'", e.Type)
	}

	f := params["format"]
	if len(f.Enum) != 3 {
		t.Errorf("format enum len = %d, want 3", len(f.Enum))
	}
}

func TestFromStruct_Pointer(t *testing.T) {
	t.Parallel()
	params := FromStruct(&testInput{})
	if len(params) != 5 {
		t.Fatalf("expected 5 params from pointer, got %d", len(params))
	}
}

func TestFromStruct_Nil(t *testing.T) {
	t.Parallel()
	params := FromStruct(nil)
	if params != nil {
		t.Fatalf("expected nil from nil input, got %v", params)
	}
}

func TestFromStruct_NonStruct(t *testing.T) {
	t.Parallel()
	params := FromStruct("not a struct")
	if params != nil {
		t.Fatalf("expected nil from non-struct, got %v", params)
	}
}

type dashField struct {
	Hidden string `json:"-"     desc:"should be excluded"`
	Shown  string `json:"shown" desc:"included"`
}

func TestFromStruct_DashJSON(t *testing.T) {
	t.Parallel()
	params := FromStruct(dashField{})
	if _, ok := params["Hidden"]; ok {
		t.Error("Hidden field with json:\"-\" should be excluded")
	}
	if _, ok := params["-"]; ok {
		t.Error("field with json:\"-\" should not appear as '-'")
	}
	if _, ok := params["shown"]; !ok {
		t.Error("shown field should be included")
	}
}

func TestFromStruct_FallbackFieldName(t *testing.T) {
	t.Parallel()
	type noTag struct {
		SearchQuery string `desc:"the query"`
	}
	params := FromStruct(noTag{})
	if _, ok := params["searchQuery"]; !ok {
		t.Errorf("expected 'searchQuery' (camelCase fallback), got: %v", params)
	}
}

func TestGoTypeToJSONType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		typ  reflect.Type
		want string
	}{
		{reflect.TypeOf(""), "string"},
		{reflect.TypeOf(true), "boolean"},
		{reflect.TypeOf(0), "integer"},
		{reflect.TypeOf(int64(0)), "integer"},
		{reflect.TypeOf(float32(0)), "number"},
		{reflect.TypeOf(float64(0)), "number"},
		{reflect.TypeOf([]byte{}), "string"},
	}
	for _, c := range cases {
		got := goTypeToJSONType(c.typ)
		if got != c.want {
			t.Errorf("goTypeToJSONType(%s) = %q, want %q", c.typ, got, c.want)
		}
	}
}
