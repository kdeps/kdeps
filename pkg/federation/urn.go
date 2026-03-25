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
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// URN represents a Universal Agent Federation identifier.
// Format: urn:agent:<authority>/<namespace>:<name>@<version>#<algorithm>:<hash>.
type URN struct {
	Authority   string // hostname[:port]
	Namespace   string // org/entity identifier
	Name        string // agent name
	Version     string // semantic version
	HashAlg     string // sha256, sha512, blake3
	ContentHash []byte // raw hash bytes
}

var urnRegex = regexp.MustCompile(
	`^urn:agent:([^/]+)/([^:]+):([^@]+)@([^#]+)#([^:]+):([a-fA-F0-9]+)$`,
)

// Parse parses a URN string into its structured components.
// URNs are case-insensitive in authority, namespace, name, and version,
// but the content hash must be lowercase hex.
func Parse(urnStr string) (*URN, error) {
	urnStr = strings.TrimSpace(urnStr)
	matches := urnRegex.FindStringSubmatch(urnStr)
	if matches == nil {
		return nil, fmt.Errorf("%w: malformed URN %q", ErrInvalidURN, urnStr)
	}

	// Normalize: lowercase for all string components except hash bytes
	authority := strings.ToLower(matches[1])
	namespace := strings.ToLower(matches[2])
	name := strings.ToLower(matches[3])
	version := strings.ToLower(matches[4])
	hashAlg := strings.ToLower(matches[5])
	hashHex := strings.ToLower(matches[6])

	// Validate hash algorithm
	if hashAlg != "sha256" && hashAlg != "sha512" && hashAlg != "blake3" {
		return nil, fmt.Errorf("%w: unsupported hash algorithm %q", ErrInvalidURN, hashAlg)
	}

	// Decode hash
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid hex hash: %w", ErrInvalidURN, err)
	}

	// Validate hash length
	expectedLen := 0
	switch hashAlg {
	case "sha256":
		expectedLen = 32
	case "sha512":
		expectedLen = 64
	case "blake3":
		expectedLen = 32
	}
	if len(hashBytes) != expectedLen {
		return nil, fmt.Errorf("%w: hash length mismatch for %s (expected %d, got %d)",
			ErrInvalidURN, hashAlg, expectedLen, len(hashBytes))
	}

	return &URN{
		Authority:   authority,
		Namespace:   namespace,
		Name:        name,
		Version:     version,
		HashAlg:     hashAlg,
		ContentHash: hashBytes,
	}, nil
}

// String returns the canonical string representation of the URN.
// The content hash is always lowercase hex.
func (u *URN) String() string {
	return fmt.Sprintf("urn:agent:%s/%s:%s@%s#%s:%s",
		u.Authority,
		u.Namespace,
		u.Name,
		u.Version,
		u.HashAlg,
		hex.EncodeToString(u.ContentHash))
}

// Equals checks if two URNs are identical (byte-for-byte).
func (u *URN) Equals(other *URN) bool {
	if u == nil || other == nil {
		return u == other
	}
	return u.Authority == other.Authority &&
		u.Namespace == other.Namespace &&
		u.Name == other.Name &&
		u.Version == other.Version &&
		u.HashAlg == other.HashAlg &&
		hex.EncodeToString(u.ContentHash) == hex.EncodeToString(other.ContentHash)
}

// Component returns a specific URN component by name.
// Valid components: "authority", "namespace", "name", "version", "hashalg", "hash".
func (u *URN) Component(component string) (string, error) {
	switch strings.ToLower(component) {
	case "authority":
		return u.Authority, nil
	case "namespace":
		return u.Namespace, nil
	case "name":
		return u.Name, nil
	case "version":
		return u.Version, nil
	case "hashalg":
		return u.HashAlg, nil
	case "hash":
		return hex.EncodeToString(u.ContentHash), nil
	default:
		return "", fmt.Errorf("unknown component %q", component)
	}
}

// Validate checks if the URN meets all structural requirements.
// Returns nil if valid, error otherwise.
func (u *URN) Validate() error {
	if u.Authority == "" {
		return errors.New("authority cannot be empty")
	}
	if u.Namespace == "" {
		return errors.New("namespace cannot be empty")
	}
	if u.Name == "" {
		return errors.New("name cannot be empty")
	}
	if u.Version == "" {
		return errors.New("version cannot be empty")
	}
	if u.HashAlg == "" {
		return errors.New("hash algorithm cannot be empty")
	}
	if len(u.ContentHash) == 0 {
		return errors.New("content hash cannot be empty")
	}
	return nil
}

// MarshalJSON converts URN to JSON as its canonical string representation.
// NOTE: Pass *URN or &struct to json.Marshal to ensure this is called.
// In Go, pointer-receiver methods on value fields are only invoked when
// the containing struct is marshaled via a pointer (e.g. json.Marshal(&receipt)).
func (u *URN) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

// UnmarshalJSON parses a URN from a JSON string.
func (u *URN) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*u = *parsed
	return nil
}
