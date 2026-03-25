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

package federation

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsFederationError_KnownErrors(t *testing.T) {
	knownErrors := []struct {
		name string
		err  error
	}{
		{"ErrInvalidURN", ErrInvalidURN},
		{"ErrURNMismatch", ErrURNMismatch},
		{"ErrMissingAuthority", ErrMissingAuthority},
		{"ErrInvalidNamespace", ErrInvalidNamespace},
		{"ErrInvalidName", ErrInvalidName},
		{"ErrInvalidVersion", ErrInvalidVersion},
		{"ErrInvalidHash", ErrInvalidHash},
		{"ErrInvalidHashAlgo", ErrInvalidHashAlgo},
		{"ErrRegistryUnreachable", ErrRegistryUnreachable},
		{"ErrAgentNotFound", ErrAgentNotFound},
		{"ErrCapabilityInvalid", ErrCapabilityInvalid},
		{"ErrSignatureInvalid", ErrSignatureInvalid},
		{"ErrReceiptInvalid", ErrReceiptInvalid},
		{"ErrMissingPublicKey", ErrMissingPublicKey},
		{"ErrAuthFailed", ErrAuthFailed},
		{"ErrUnauthorized", ErrUnauthorized},
		{"ErrRateLimited", ErrRateLimited},
		{"ErrTimeout", ErrTimeout},
		{"ErrSchemaMismatch", ErrSchemaMismatch},
		{"ErrMissingInput", ErrMissingInput},
		{"ErrCacheMiss", ErrCacheMiss},
		{"ErrTrustLevel", ErrTrustLevel},
		{"ErrKeyNotFound", ErrKeyNotFound},
		{"ErrKeyRotation", ErrKeyRotation},
	}

	for _, tc := range knownErrors {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, IsFederationError(tc.err),
				"IsFederationError(%s) should return true", tc.name)
		})
	}
}

func TestIsFederationError_WrappedErrors(t *testing.T) {
	wrappedErrors := []struct {
		name string
		err  error
	}{
		{
			"wrapped ErrTimeout",
			fmt.Errorf("operation failed: %w", ErrTimeout),
		},
		{
			"wrapped ErrInvalidURN",
			fmt.Errorf("parse error: %w", ErrInvalidURN),
		},
		{
			"wrapped ErrAuthFailed",
			fmt.Errorf("wrap: %w", ErrAuthFailed),
		},
		{
			"wrapped ErrCacheMiss",
			fmt.Errorf("lookup failed: %w", ErrCacheMiss),
		},
		{
			"double-wrapped ErrAgentNotFound",
			fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", ErrAgentNotFound)),
		},
		{
			"wrapped ErrRateLimited",
			fmt.Errorf("wrap: %w", ErrRateLimited),
		},
		{
			"wrapped ErrKeyRotation",
			fmt.Errorf("key op: %w", ErrKeyRotation),
		},
		{
			"wrapped ErrSignatureInvalid",
			fmt.Errorf("verify: %w", ErrSignatureInvalid),
		},
	}

	for _, tc := range wrappedErrors {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, IsFederationError(tc.err),
				"IsFederationError(%s) should return true for wrapped federation error", tc.name)
		})
	}
}

func TestIsFederationError_UnknownErrors(t *testing.T) {
	unknownErrors := []struct {
		name string
		err  error
	}{
		{
			"plain stdlib error",
			errors.New("something went wrong"),
		},
		{
			"io EOF",
			fmt.Errorf("read failed: %w", errors.New("unexpected EOF")),
		},
		{
			"generic context error",
			errors.New("context deadline exceeded"),
		},
		{
			"unrelated wrapped error",
			fmt.Errorf("outer: %w", errors.New("inner unrelated")),
		},
		{
			"error with federation-like message but not a federation error",
			errors.New("invalid URN format"),
		},
	}

	for _, tc := range unknownErrors {
		t.Run(tc.name, func(t *testing.T) {
			assert.False(t, IsFederationError(tc.err),
				"IsFederationError(%s) should return false for non-federation error", tc.name)
		})
	}
}

func TestIsFederationError_Nil(t *testing.T) {
	assert.False(t, IsFederationError(nil), "IsFederationError(nil) should return false")
}
