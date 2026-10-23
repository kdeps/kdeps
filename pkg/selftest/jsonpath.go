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

package selftest

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
)

// jsonPathSegment is one step of a JSONPath expression: a map key or a slice index.
type jsonPathSegment struct {
	key   string
	index int
	isIdx bool
}

// ValidJSONPath reports whether path uses the supported subset of JSONPath:
// a "$" root, ".key" map steps, and "[n]" slice steps (e.g. $.items[0].name).
func ValidJSONPath(path string) bool {
	_, ok := parseJSONPath(path)
	return ok
}

// ResolveJSONPath resolves path against a decoded JSON document. It returns
// the resolved value and whether the path exists.
func ResolveJSONPath(doc interface{}, path string) (interface{}, bool) {
	segments, ok := parseJSONPath(path)
	if !ok {
		return nil, false
	}
	current := doc
	for _, seg := range segments {
		next, found := stepJSONPath(current, seg)
		if !found {
			return nil, false
		}
		current = next
	}
	return current, true
}

func stepJSONPath(current interface{}, seg jsonPathSegment) (interface{}, bool) {
	if seg.isIdx {
		items, ok := current.([]interface{})
		if !ok || seg.index < 0 || seg.index >= len(items) {
			return nil, false
		}
		return items[seg.index], true
	}
	obj, ok := current.(map[string]interface{})
	if !ok {
		return nil, false
	}
	value, found := obj[seg.key]
	return value, found
}

func parseJSONPath(path string) ([]jsonPathSegment, bool) {
	rest, ok := strings.CutPrefix(path, "$")
	if !ok {
		return nil, false
	}
	var segments []jsonPathSegment
	for rest != "" {
		switch rest[0] {
		case '.':
			key, remaining, keyOK := readJSONPathKey(rest)
			if !keyOK {
				return nil, false
			}
			segments = append(segments, jsonPathSegment{key: key})
			rest = remaining
		case '[':
			seg, remaining, indexOK := readJSONPathIndex(rest)
			if !indexOK {
				return nil, false
			}
			segments = append(segments, seg)
			rest = remaining
		default:
			return nil, false
		}
	}
	return segments, true
}

func readJSONPathKey(rest string) (string, string, bool) {
	end := 1
	for end < len(rest) && rest[end] != '.' && rest[end] != '[' {
		end++
	}
	key := rest[1:end]
	if key == "" {
		return "", "", false
	}
	return key, rest[end:], true
}

func readJSONPathIndex(rest string) (jsonPathSegment, string, bool) {
	end := strings.IndexByte(rest, ']')
	if end < 0 {
		return jsonPathSegment{}, "", false
	}
	index, err := strconv.Atoi(rest[1:end])
	if err != nil || index < 0 {
		return jsonPathSegment{}, "", false
	}
	return jsonPathSegment{index: index, isIdx: true}, rest[end+1:], true
}

// valuesEqual compares a JSON-decoded value with the expected YAML value.
// JSON numbers decode to float64 while YAML integers decode to int, so
// numeric pairs compare by value rather than by Go type.
func valuesEqual(actual, expected interface{}) bool {
	if actualNum, ok := toFloat64(actual); ok {
		expectedNum, expectedOK := toFloat64(expected)
		return expectedOK && actualNum == expectedNum
	}
	return reflect.DeepEqual(actual, expected)
}

func toFloat64(value interface{}) (float64, bool) {
	switch num := value.(type) {
	case float64:
		return num, true
	case float32:
		return float64(num), true
	case int:
		return float64(num), true
	case int64:
		return float64(num), true
	case json.Number:
		parsed, err := num.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
