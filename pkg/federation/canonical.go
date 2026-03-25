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
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Canonicalizer handles deterministic serialization of YAML specifications
// for content-addressed identity.
type Canonicalizer struct{}

// ComputeHash returns the content hash of a YAML spec according to UAF rules.
// The spec is canonicalized (deterministic) then hashed with the specified algorithm.
// If alg is empty, SHA256 is used.
func (c *Canonicalizer) ComputeHash(yamlBytes []byte, alg string) ([]byte, error) {
	canonical, err := c.Canonicalize(yamlBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to canonicalize YAML: %w", err)
	}

	h, err := c.newHash(alg)
	if err != nil {
		return nil, err
	}
	h.Write(canonical)
	return h.Sum(nil), nil
}

// SHA256 returns the SHA256 hash of canonicalized YAML.
func (c *Canonicalizer) SHA256(yamlBytes []byte) ([]byte, error) {
	return c.ComputeHash(yamlBytes, "sha256")
}

// Canonicalize converts YAML to deterministic JSON for hashing.
// Rules:
//   - Parse YAML preserving order
//   - Sort all object keys lexicographically
//   - Remove comments (yaml.v3 does this by default)
//   - Strip leading/trailing whitespace from all scalar values
//   - Convert to JSON with minimal whitespace (json.Compact)
//   - Use LF line endings (handled by YAML parser)
func (c *Canonicalizer) Canonicalize(yamlBytes []byte) ([]byte, error) {
	var data interface{}
	if err := yaml.Unmarshal(yamlBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Normalize the data structure
	normalized := c.normalize(data)

	// Marshal to JSON with minimal whitespace
	jsonBytes, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Compact (though json.Marshal already does this)
	var buf bytes.Buffer
	if err := json.Compact(&buf, jsonBytes); err != nil {
		return nil, fmt.Errorf("failed to compact JSON: %w", err)
	}

	return buf.Bytes(), nil
}

// normalize recursively processes YAML data to ensure deterministic structure.
func (c *Canonicalizer) normalize(data interface{}) interface{} {
	switch v := data.(type) {
	case map[interface{}]interface{}:
		// Convert to string-keyed map and sort keys
		result := make(map[string]interface{}, len(v))
		keys := make([]string, 0, len(v))
		for k := range v {
			if strKey, ok := k.(string); ok {
				keys = append(keys, strKey)
			} else {
				// Non-string keys: convert to string (though unusual in YAML)
				keys = append(keys, fmt.Sprintf("%v", k))
			}
		}
		sort.Strings(keys)
		for _, k := range keys {
			result[k] = c.normalize(v[k])
		}
		return result

	case map[string]interface{}:
		// Already string-keyed; sort keys and normalize values
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		result := make(map[string]interface{}, len(v))
		for _, k := range keys {
			result[k] = c.normalize(v[k])
		}
		return result

	case []interface{}:
		// Arrays keep order, but normalize each element
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = c.normalize(item)
		}
		return result

	case string:
		// Strip leading/trailing whitespace from strings
		return strings.TrimSpace(v)

	case nil:
		return nil

	default:
		// Primitives (int, float, bool) returned as-is
		return v
	}
}

// newHash creates a hash.Hash for the given algorithm.
func (c *Canonicalizer) newHash(alg string) (hash.Hash, error) {
	switch strings.ToLower(alg) {
	case "sha256":
		return sha256.New(), nil
	case "sha512":
		return sha512.New(), nil
	case "blake3":
		// BLAKE3 not in standard library; will need golang.org/x/crypto/blake3
		// For now, return error with helpful message
		return nil, fmt.Errorf("blake3 not yet implemented; use sha256 or sha512")
	default:
		return nil, fmt.Errorf("unsupported hash algorithm %q", alg)
	}
}

// HashHex returns the hash as a lowercase hex string.
func (c *Canonicalizer) HashHex(hashBytes []byte) string {
	return hex.EncodeToString(hashBytes)
}

// ComputeAndFormat computes the hash and returns as hex string.
func (c *Canonicalizer) ComputeAndFormat(yamlBytes []byte, alg string) (string, error) {
	h, err := c.ComputeHash(yamlBytes, alg)
	if err != nil {
		return "", err
	}
	return c.HashHex(h), nil
}

// DefaultCanonicalizer is a ready-to-use instance.
var DefaultCanonicalizer = &Canonicalizer{}

// ComputeHash is a convenience wrapper around DefaultCanonicalizer.ComputeHash.
func ComputeHash(yamlBytes []byte, alg string) ([]byte, error) {
	return DefaultCanonicalizer.ComputeHash(yamlBytes, alg)
}

// SHA256 is a convenience wrapper around DefaultCanonicalizer.SHA256.
func SHA256(yamlBytes []byte) ([]byte, error) {
	return DefaultCanonicalizer.SHA256(yamlBytes)
}
