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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseURN(t *testing.T) {
	tests := []struct {
		name        string
		urn         string
		expected    *URN
		shouldError bool
	}{
		{
			name: "valid SHA256 URN",
			urn:  "urn:agent:acme-corp.example.com/compliance:checker@v2.1.0#sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: &URN{
				Authority: "acme-corp.example.com",
				Namespace: "compliance",
				Name:      "checker",
				Version:   "v2.1.0",
				HashAlg:   "sha256",
				ContentHash: mustDecodeHex(
					"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				),
			},
			shouldError: false,
		},
		{
			name: "valid SHA512 URN",
			urn: "urn:agent:localhost:8080/test:agent@v1.0.0#sha512:" + strings.Repeat(
				"0123456789abcdef",
				8,
			), // 128 hex digits
			expected: &URN{
				Authority:   "localhost:8080",
				Namespace:   "test",
				Name:        "agent",
				Version:     "v1.0.0",
				HashAlg:     "sha512",
				ContentHash: mustDecodeHex(strings.Repeat("0123456789abcdef", 8)),
			},
			shouldError: false,
		},
		{
			name: "valid BLAKE3 URN",
			urn: "urn:agent:my-internal.net/org:name@v0.0.1#blake3:" + strings.Repeat(
				"fedcba9876543210",
				4,
			), // 64 hex digits
			expected: &URN{
				Authority:   "my-internal.net",
				Namespace:   "org",
				Name:        "name",
				Version:     "v0.0.1",
				HashAlg:     "blake3",
				ContentHash: mustDecodeHex(strings.Repeat("fedcba9876543210", 4)),
			},
			shouldError: false,
		},
		{
			name:        "empty string",
			urn:         "",
			shouldError: true,
		},
		{
			name:        "missing urn prefix",
			urn:         "agent:test/name@v1.0#sha256:abc123",
			shouldError: true,
		},
		{
			name:        "missing hash",
			urn:         "urn:agent:test/name@v1.0#sha256",
			shouldError: true,
		},
		{
			name:        "invalid hash hex",
			urn:         "urn:agent:test/name@v1.0#sha256:zzzzzz",
			shouldError: true,
		},
		{
			name:        "wrong hash length",
			urn:         "urn:agent:test/name@v1.0#sha256:abc",
			shouldError: true,
		},
		{
			name:        "unsupported hash algorithm",
			urn:         "urn:agent:test/name@v1.0#md5:abc123",
			shouldError: true,
		},
		{
			name:        "missing namespace colon",
			urn:         "urn:agent:test/name@v1.0#sha256:abc123",
			shouldError: true,
		},
		{
			name:        "missing version at symbol",
			urn:         "urn:agent:test/namev1.0#sha256:abc123",
			shouldError: true,
		},
		{
			name: "case insensitivity in components",
			urn:  "urn:agent:ACMECorp.Example.COM/ACME-CORP:Name@V2.1.0#SHA256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: &URN{
				Authority: "acmecorp.example.com",
				Namespace: "acme-corp",
				Name:      "name",
				Version:   "v2.1.0",
				HashAlg:   "sha256",
				ContentHash: mustDecodeHex(
					"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				),
			},
			shouldError: false,
		},
		{
			name: "colon in name",
			urn:  "urn:agent:test.com/my-namespace:my/name@v1.0#sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			expected: &URN{
				Authority: "test.com",
				Namespace: "my-namespace",
				Name:      "my/name",
				Version:   "v1.0",
				HashAlg:   "sha256",
				ContentHash: mustDecodeHex(
					"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				),
			},
			shouldError: false,
		},
		{
			name: "whitespace trimming",
			urn:  "  urn:agent:test.com/ns:name@v1.0#sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef  ",
			expected: &URN{
				Authority: "test.com",
				Namespace: "ns",
				Name:      "name",
				Version:   "v1.0",
				HashAlg:   "sha256",
				ContentHash: mustDecodeHex(
					"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				),
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.urn)
			if tt.shouldError {
				assert.Error(t, err, "expected error for URN: %s", tt.urn)
				return
			}
			require.NoError(t, err, "unexpected error for URN: %s", tt.urn)
			assert.Equal(t, tt.expected, got, "parsed URN mismatch")
			assert.Equal(t, got.String(), tt.expected.String(), "String() round-trip")
		})
	}
}

func TestURNValidate(t *testing.T) {
	tests := []struct {
		name    string
		urn     *URN
		isValid bool
	}{
		{
			name: "complete URN",
			urn: &URN{
				Authority:   "test.com",
				Namespace:   "ns",
				Name:        "agent",
				Version:     "v1.0.0",
				HashAlg:     "sha256",
				ContentHash: make([]byte, 32),
			},
			isValid: true,
		},
		{
			name: "missing authority",
			urn: &URN{
				Namespace:   "ns",
				Name:        "agent",
				Version:     "v1.0.0",
				HashAlg:     "sha256",
				ContentHash: make([]byte, 32),
			},
			isValid: false,
		},
		{
			name: "missing namespace",
			urn: &URN{
				Authority:   "test.com",
				Name:        "agent",
				Version:     "v1.0.0",
				HashAlg:     "sha256",
				ContentHash: make([]byte, 32),
			},
			isValid: false,
		},
		{
			name: "missing name",
			urn: &URN{
				Authority:   "test.com",
				Namespace:   "ns",
				Version:     "v1.0.0",
				HashAlg:     "sha256",
				ContentHash: make([]byte, 32),
			},
			isValid: false,
		},
		{
			name: "missing version",
			urn: &URN{
				Authority:   "test.com",
				Namespace:   "ns",
				Name:        "agent",
				HashAlg:     "sha256",
				ContentHash: make([]byte, 32),
			},
			isValid: false,
		},
		{
			name: "missing hash alg",
			urn: &URN{
				Authority:   "test.com",
				Namespace:   "ns",
				Name:        "agent",
				Version:     "v1.0.0",
				ContentHash: make([]byte, 32),
			},
			isValid: false,
		},
		{
			name: "missing content hash",
			urn: &URN{
				Authority: "test.com",
				Namespace: "ns",
				Name:      "agent",
				Version:   "v1.0.0",
				HashAlg:   "sha256",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.urn.Validate()
			if tt.isValid {
				assert.NoError(t, err, "expected valid URN")
			} else {
				assert.Error(t, err, "expected invalid URN")
			}
		})
	}
}

func TestURNComponent(t *testing.T) {
	urn := &URN{
		Authority: "test.com",
		Namespace: "my-ns",
		Name:      "my-agent",
		Version:   "v1.2.3",
		HashAlg:   "sha256",
		ContentHash: mustDecodeHex(
			"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		),
	}

	tests := []struct {
		component string
		expected  string
		shouldErr bool
	}{
		{"authority", "test.com", false},
		{"namespace", "my-ns", false},
		{"name", "my-agent", false},
		{"version", "v1.2.3", false},
		{"hashalg", "sha256", false},
		{"hash", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.component, func(t *testing.T) {
			got, err := urn.Component(tt.component)
			if tt.shouldErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func mustDecodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(fmt.Errorf("failed to decode hex string %q: %w", s, err))
	}
	return b
}

func TestURNEquals(t *testing.T) {
	hashHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	hash := mustDecodeHex(hashHex)

	base := &URN{
		Authority:   "agents.example.com",
		Namespace:   "myns",
		Name:        "myagent",
		Version:     "v1.0.0",
		HashAlg:     hashAlgorithmSHA256,
		ContentHash: hash,
	}

	// Equal to itself.
	assert.True(t, base.Equals(base)) //nolint:gocritic // intentional: testing reflexivity of Equals

	// Equal copy.
	other := &URN{
		Authority:   "agents.example.com",
		Namespace:   "myns",
		Name:        "myagent",
		Version:     "v1.0.0",
		HashAlg:     hashAlgorithmSHA256,
		ContentHash: mustDecodeHex(hashHex),
	}
	assert.True(t, base.Equals(other))

	// Different authority.
	diff := *base
	diff.Authority = "other.example.com"
	assert.False(t, base.Equals(&diff))

	// Different name.
	diff2 := *base
	diff2.Name = "different"
	assert.False(t, base.Equals(&diff2))

	// Different hash.
	diff3 := *base
	diff3.ContentHash = mustDecodeHex("abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789")
	assert.False(t, base.Equals(&diff3))

	// Nil receiver.
	var nilURN *URN
	assert.False(t, nilURN.Equals(base))

	// Both nil.
	assert.True(t, nilURN.Equals(nil))

	// Nil argument.
	assert.False(t, base.Equals(nil))
}
