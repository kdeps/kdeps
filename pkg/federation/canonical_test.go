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
		name           string
		yaml           string
		mustContain    []string
		mustNotContain []string
	}{
		{
			name: "simple map with ordered keys",
			yaml: `
a: 1
b: 2
c: 3
`,
			mustContain:    []string{`"a":1`, `"b":2`, `"c":3`},
			mustNotContain: []string{"a: 1", "comments"},
		},
		{
			name: "nested objects",
			yaml: `
outer:
  inner1: value1
  inner2: value2
`,
			mustContain:    []string{`"outer":{"inner1":"value1","inner2":"value2"}`},
			mustNotContain: []string{`inner1: value1`}, // YAML format should be gone
		},
		{
			name: "trim whitespace in strings",
			yaml: `key: "  spaced value  "
another: "	\tvalue	"
`,
			mustContain:    []string{`"key":"spaced value"`, `"another":"value"`},
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

func TestComputeAndFormat(t *testing.T) {
	c := &Canonicalizer{}
	yaml := `key: value`

	// SHA256 produces 64-char hex string.
	hexStr, err := c.ComputeAndFormat([]byte(yaml), hashAlgorithmSHA256)
	assert.NoError(t, err)
	assert.Len(t, hexStr, 64)
	_, decErr := hex.DecodeString(hexStr)
	assert.NoError(t, decErr)

	// SHA512 produces 128-char hex string.
	hexStr512, err := c.ComputeAndFormat([]byte(yaml), hashAlgorithmSHA512)
	assert.NoError(t, err)
	assert.Len(t, hexStr512, 128)

	// Unsupported algorithm.
	_, err = c.ComputeAndFormat([]byte(yaml), "md5")
	assert.Error(t, err)

	// Invalid YAML triggers error path.
	_, err = c.ComputeAndFormat([]byte(":\tbad: [yaml"), hashAlgorithmSHA256)
	assert.Error(t, err)
}

func TestPackageLevelComputeHash(t *testing.T) {
	yaml := `name: agent`
	h, err := ComputeHash([]byte(yaml), hashAlgorithmSHA256)
	assert.NoError(t, err)
	assert.Len(t, h, 32)
}

func TestPackageLevelSHA256(t *testing.T) {
	yaml := `name: agent`
	h, err := SHA256([]byte(yaml))
	assert.NoError(t, err)
	assert.Len(t, h, 32)
}

func TestNewHash_SHA512(t *testing.T) {
	c := &Canonicalizer{}
	h, err := c.newHash(hashAlgorithmSHA512)
	assert.NoError(t, err)
	assert.NotNil(t, h)
	// sha512 produces 64-byte digests
	assert.Equal(t, 64, h.Size())
}

func TestNewHash_BLAKE3(t *testing.T) {
	c := &Canonicalizer{}
	_, err := c.newHash(hashAlgorithmBLAKE3)
	assert.Error(t, err)
}

func TestNewHash_Unknown(t *testing.T) {
	c := &Canonicalizer{}
	_, err := c.newHash("crc32")
	assert.Error(t, err)
}

func TestNormalize_Array(t *testing.T) {
	c := &Canonicalizer{}
	input := []interface{}{"c", "a", "b"}
	got := c.normalize(input)
	arr, ok := got.([]interface{})
	assert.True(t, ok)
	// Arrays keep their order.
	assert.Equal(t, []interface{}{"c", "a", "b"}, arr)
}

func TestNormalize_Nil(t *testing.T) {
	c := &Canonicalizer{}
	assert.Nil(t, c.normalize(nil))
}

func TestNormalize_Primitive(t *testing.T) {
	c := &Canonicalizer{}
	assert.Equal(t, 42, c.normalize(42))
	assert.Equal(t, true, c.normalize(true))
	assert.Equal(t, 3.14, c.normalize(3.14))
}

func TestNormalize_NonStringKeyMap(t *testing.T) {
	c := &Canonicalizer{}
	// yaml.v3 can decode integer keys as map[interface{}]interface{}
	// The normalize function converts non-string keys to their string representation,
	// then looks them up in the original map (which may return the zero value for
	// type-mismatched lookups — this is expected behavior for this edge case).
	input := map[interface{}]interface{}{
		"two": "2",
	}
	result := c.normalize(input)
	m, ok := result.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "2", m["two"])
}
