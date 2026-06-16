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

// Package jsonschema provides helpers to generate tool parameter schemas
// from Go structs using reflection and struct tags. This mirrors the
// langchaingo jsonschema/ package but is scoped to kdeps domain types.
package jsonschema

import (
	"reflect"
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// FromStruct generates a map[string]domain.ToolParam from the exported fields
// of a Go struct (or pointer to struct). Uses the following struct tags:
//
//   - json:"name"     — field name (falls back to lowercase Go name)
//   - desc:"..."      — parameter description
//   - required:"true" — marks the parameter as required
//   - enum:"a,b,c"   — comma-separated list of allowed values (string fields)
//
// Only exported fields with non-zero JSON names are included.
// Supported Go types → JSON Schema types:
//
//	string        → "string"
//	bool          → "boolean"
//	int/int64/... → "integer"
//	float32/64    → "number"
//	everything else → "string"
func FromStruct(v interface{}) map[string]domain.ToolParam {
	t := reflect.TypeOf(v)
	if t == nil {
		return nil
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	params := make(map[string]domain.ToolParam, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		name := fieldName(f)
		if name == "" || name == "-" {
			continue
		}

		p := domain.ToolParam{
			Type:        goTypeToJSONType(f.Type),
			Description: f.Tag.Get("desc"),
			Required:    f.Tag.Get("required") == "true",
		}

		if raw := f.Tag.Get("enum"); raw != "" {
			for _, v := range strings.Split(raw, ",") {
				v = strings.TrimSpace(v)
				if v != "" {
					p.Enum = append(p.Enum, v)
				}
			}
		}

		params[name] = p
	}
	return params
}

// fieldName returns the JSON field name from struct tags, falling back to
// the lowercase first-letter version of the Go field name.
func fieldName(f reflect.StructField) string {
	if tag := f.Tag.Get("json"); tag != "" {
		const maxParts = 2
		parts := strings.SplitN(tag, ",", maxParts)
		if parts[0] != "" {
			return parts[0]
		}
	}
	return strings.ToLower(f.Name[:1]) + f.Name[1:]
}

// goTypeToJSONType maps a reflect.Type to a JSON Schema type string.
func goTypeToJSONType(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() { //nolint:exhaustive // only handle JSON-mapped kinds; default to "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	default:
		return "string"
	}
}
