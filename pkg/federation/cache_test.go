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
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCapability returns a simple Capability for use in cache tests.
func testCapability(urn string) *Capability {
	return &Capability{
		URN:   urn,
		Title: "Test Agent",
		Capabilities: []Action{
			{
				ActionID:    "action-1",
				Title:       "Do Thing",
				Description: "Does a thing",
				InputSchema: JSONSchema{
					Type: "object",
					Properties: map[string]Property{
						"input": {Type: "string"},
					},
				},
				OutputSchema: JSONSchema{
					Type: "object",
					Properties: map[string]Property{
						"output": {Type: "string"},
					},
				},
			},
		},
		TrustLevel: "self-attested",
		Endpoint:   "https://example.com/agent",
	}
}

func TestRegistryCacheSetGet(t *testing.T) {
	rc := NewRegistryCache(5 * time.Minute)
	defer rc.Stop()

	const urn = "urn:agent:example.com/ns:agent@v1.0.0#sha256:0000000000000000000000000000000000000000000000000000000000000001"
	capability := testCapability(urn)

	rc.Set(urn, capability)

	got, err := rc.Get(urn)
	require.NoError(t, err)
	assert.Equal(t, capability, got)
	assert.Equal(t, urn, got.URN)
	assert.Equal(t, "Test Agent", got.Title)
}

func TestRegistryCacheMiss(t *testing.T) {
	rc := NewRegistryCache(5 * time.Minute)
	defer rc.Stop()

	const urn = "urn:agent:example.com/ns:missing@v1.0.0#sha256:0000000000000000000000000000000000000000000000000000000000000002"

	got, err := rc.Get(urn)
	assert.ErrorIs(t, err, ErrCacheMiss)
	assert.Nil(t, got)
}

func TestRegistryCacheTTLExpiry(t *testing.T) {
	rc := NewRegistryCache(50 * time.Millisecond)
	defer rc.Stop()

	const urn = "urn:agent:example.com/ns:expiry@v1.0.0#sha256:0000000000000000000000000000000000000000000000000000000000000003"
	capability := testCapability(urn)

	rc.Set(urn, capability)

	// Should be present immediately after set.
	got, err := rc.Get(urn)
	require.NoError(t, err)
	assert.Equal(t, capability, got)

	// Wait for TTL to elapse.
	time.Sleep(100 * time.Millisecond)

	// Should now be a cache miss.
	got, err = rc.Get(urn)
	assert.ErrorIs(t, err, ErrCacheMiss)
	assert.Nil(t, got)
}

func TestRegistryCacheInvalidate(t *testing.T) {
	rc := NewRegistryCache(5 * time.Minute)
	defer rc.Stop()

	const urn = "urn:agent:example.com/ns:invalidate@v1.0.0#sha256:0000000000000000000000000000000000000000000000000000000000000004"
	capability := testCapability(urn)

	rc.Set(urn, capability)

	// Confirm it is present.
	_, err := rc.Get(urn)
	require.NoError(t, err)

	// Invalidate and confirm miss.
	rc.Invalidate(urn)

	got, err := rc.Get(urn)
	assert.ErrorIs(t, err, ErrCacheMiss)
	assert.Nil(t, got)
}

func TestRegistryCacheInvalidateNonExistent(t *testing.T) {
	rc := NewRegistryCache(5 * time.Minute)
	defer rc.Stop()

	const urn = "urn:agent:example.com/ns:nothere@v1.0.0#sha256:0000000000000000000000000000000000000000000000000000000000000005"

	// Invalidating a key that was never set must not panic.
	assert.NotPanics(t, func() {
		rc.Invalidate(urn)
	})

	got, err := rc.Get(urn)
	assert.ErrorIs(t, err, ErrCacheMiss)
	assert.Nil(t, got)
}

func TestRegistryCacheClear(t *testing.T) {
	rc := NewRegistryCache(5 * time.Minute)
	defer rc.Stop()

	urns := []string{
		"urn:agent:example.com/ns:alpha@v1.0.0#sha256:0000000000000000000000000000000000000000000000000000000000000006",
		"urn:agent:example.com/ns:beta@v1.0.0#sha256:0000000000000000000000000000000000000000000000000000000000000007",
		"urn:agent:example.com/ns:gamma@v1.0.0#sha256:0000000000000000000000000000000000000000000000000000000000000008",
	}

	for _, urn := range urns {
		rc.Set(urn, testCapability(urn))
	}

	// Confirm all entries are present before clearing.
	for _, urn := range urns {
		_, err := rc.Get(urn)
		require.NoError(t, err, "expected hit before clear for %s", urn)
	}

	rc.Clear()

	// Confirm all entries are gone after clearing.
	for _, urn := range urns {
		got, err := rc.Get(urn)
		assert.ErrorIs(t, err, ErrCacheMiss, "expected miss after clear for %s", urn)
		assert.Nil(t, got)
	}
}

func TestRegistryCacheOverwrite(t *testing.T) {
	rc := NewRegistryCache(5 * time.Minute)
	defer rc.Stop()

	const urn = "urn:agent:example.com/ns:overwrite@v1.0.0#sha256:0000000000000000000000000000000000000000000000000000000000000009"

	first := testCapability(urn)
	first.Title = "First"

	second := testCapability(urn)
	second.Title = "Second"

	rc.Set(urn, first)
	rc.Set(urn, second)

	got, err := rc.Get(urn)
	require.NoError(t, err)
	assert.Equal(t, "Second", got.Title)
}

func TestRegistryCacheConcurrent(t *testing.T) {
	t.Parallel()
	rc := NewRegistryCache(5 * time.Minute)
	defer rc.Stop()

	const workers = 10
	const iterations = 100

	// Pre-populate keys so readers have something to find.
	for i := range workers {
		urn := fmt.Sprintf(
			"urn:agent:example.com/ns:concurrent%d@v1.0.0#sha256:%064x",
			i, i+1,
		)
		rc.Set(urn, testCapability(urn))
	}

	var wg sync.WaitGroup

	// Writers.
	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			urn := fmt.Sprintf(
				"urn:agent:example.com/ns:concurrent%d@v1.0.0#sha256:%064x",
				idx, idx+1,
			)
			capability := testCapability(urn)
			for range iterations {
				rc.Set(urn, capability)
			}
		}(i)
	}

	// Readers.
	for i := range workers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			urn := fmt.Sprintf(
				"urn:agent:example.com/ns:concurrent%d@v1.0.0#sha256:%064x",
				idx, idx+1,
			)
			for range iterations {
				// A hit or miss is both valid; we only care there is no data race.
				rc.Get(urn) //nolint:errcheck
			}
		}(i)
	}

	wg.Wait()
}

func TestRegistryCacheStop(t *testing.T) {
	rc := NewRegistryCache(5 * time.Minute)

	// First Stop should not panic.
	assert.NotPanics(t, func() { rc.Stop() })

	// Second Stop must also not panic (channel already closed).
	assert.NotPanics(t, func() { rc.Stop() })
}
