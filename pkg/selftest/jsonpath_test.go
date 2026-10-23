// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package selftest_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/selftest"
)

func decodeDoc(t *testing.T, raw string) interface{} {
	t.Helper()
	var doc interface{}
	require.NoError(t, json.Unmarshal([]byte(raw), &doc))
	return doc
}

func TestValidJSONPath(t *testing.T) {
	valid := []string{"$", "$.ok", "$.data.name", "$.items[0]", "$.items[0].name", "$[1]"}
	for _, path := range valid {
		assert.True(t, selftest.ValidJSONPath(path), "expected %q to be valid", path)
	}
	invalid := []string{"", "ok", ".ok", "$.", "$..name", "$.items[]", "$.items[x]", "$.items[-1]", "$['a']"}
	for _, path := range invalid {
		assert.False(t, selftest.ValidJSONPath(path), "expected %q to be invalid", path)
	}
}

func TestResolveJSONPath(t *testing.T) {
	doc := decodeDoc(t, `{"ok":true,"data":{"name":"n"},"items":[{"id":1},{"id":2}]}`)

	cases := []struct {
		path   string
		want   interface{}
		exists bool
	}{
		{path: "$", want: doc, exists: true},
		{path: "$.ok", want: true, exists: true},
		{path: "$.data.name", want: "n", exists: true},
		{path: "$.items[1].id", want: float64(2), exists: true},
		{path: "$.data", want: map[string]interface{}{"name": "n"}, exists: true},
		{path: "$.items[9]", exists: false},
		{path: "$.data.missing", exists: false},
		{path: "$.ok.deeper", exists: false},
		{path: "invalid", exists: false},
	}
	for _, tc := range cases {
		got, exists := selftest.ResolveJSONPath(doc, tc.path)
		assert.Equal(t, tc.exists, exists, "path %s", tc.path)
		if tc.exists {
			assert.Equal(t, tc.want, got, "path %s", tc.path)
		}
	}
}
