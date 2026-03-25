// Copyright 2025 Kdeps, KvK 94834768
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

package federation

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanonicalize(t *testing.T) {
	c := &Canonicalizer{}

	tests := []struct {
		name     string
		yaml     string
		mustContain []string
		mustNotContain []string
	}{
		{
			name: "simple map with ordered keys",
			yaml: `
a: 1
b: 2
c: 3
`,
			mustContain: []string{`"a":1`, `"b":2`, `"c":3`},
			mustNotContain: []string{"a: 1", "comments"},
		},
		{
			name: "nested objects",
			yaml: `
outer:
  inner1: value1
  inner2: value2
`,
			mustContain: []string{`"outer":{"inner1":"value1","inner2":"value2"}`},
			mustNotContain: []string{`inner1: value1`}, // YAML format should be gone
		},
		{
			name: "trim whitespace in strings",
			yaml: `key: "  spaced value  "
another: "	\tvalue	"
`,
			mustContain: []string{`"key":"spaced value"`, `"another":"value"`},
			mustNotContain: []string{"  ", "\t"},
		},
		{
			name: "arrays preserve order",
			yaml: `
items:
  - third
  - first
  - second
`,
			mustContain: []string{`["third","first","second"]`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Canonicalize([]byte(tt.yaml))
			assert.NoError(t, err, "canonicalization failed")
			jsonStr := string(got)

			for _, substr := range tt.mustContain {
				assert.Contains(t, jsonStr, substr, "missing required content")
			}
			for _, substr := range tt.mustNotContain {
				assert.NotContains(t, jsonStr, substr, "contains forbidden content")
			}
		})
	}
}

func TestComputeHash(t *testing.T) {
	c := &Canonicalizer{}

	yaml := `
name: test-agent
version: v1.0.0
`
	sha256Hash, err := c.SHA256([]byte(yaml))
	assert.NoError(t, err)
	assert.Len(t, sha256Hash, 32)

	// Same YAML should produce same hash
	sha256Hash2, _ := c.SHA256([]byte(yaml))
	assert.Equal(t, sha256Hash, sha256Hash2)

	// Different YAML produces different hash
	yaml2 := `
name: test-agent
version: v1.0.1
`
	sha256Hash3, _ := c.SHA256([]byte(yaml2))
	assert.NotEqual(t, sha256Hash, sha256Hash3)
}

func TestHashFormatting(t *testing.T) {
	c := &Canonicalizer{}

	yaml := `key: value`
	hash, _ := c.SHA256([]byte(yaml))
	hexStr := c.HashHex(hash)

	assert.Len(t, hexStr, 64)
	// Verify it's valid hex by decoding
	_, err := hex.DecodeString(hexStr)
	assert.NoError(t, err, "hex string should be valid")
}
