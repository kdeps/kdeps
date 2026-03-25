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

package registry

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testHash   = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	testURNStr = "urn:agent:agents.example.com/myns:myagent@v1.0.0#sha256:" + testHash
)

// newMuxServer creates an httptest.Server with a ServeMux. It returns both the
// server and the mux so individual test routes can be registered.
func newMuxServer(t *testing.T) (*httptest.Server, *http.ServeMux) {
	t.Helper()
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, mux
}

// buildCapability returns an AgentCapability whose URN matches urnStr.
func buildCapability(urnStr string) AgentCapability {
	return AgentCapability{
		URN:        urnStr,
		Title:      "Test Agent",
		Endpoint:   "http://agents.example.com",
		TrustLevel: "verified",
		PublicKey:  "ed25519:testkey",
		Capabilities: []Capability{
			{
				ActionID:    "say-hello",
				Title:       "Say Hello",
				Description: "Greets the caller",
			},
		},
	}
}

// TestResolveURN_Success verifies the full happy path: registry lookup then
// well-known capability fetch, with the resolved capability returned.
func TestResolveURN_Success(t *testing.T) {
	agentCap := buildCapability(testURNStr)

	ts, mux := newMuxServer(t)

	// Registry endpoint: GET /v1/agents/{urn} — returns the endpoint URL.
	mux.HandleFunc("/v1/agents/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"endpoint": ts.URL,
		}); err != nil {
			t.Errorf("failed to encode endpoint response: %v", err)
		}
	})

	// Well-known endpoint: GET /.well-known/agent/{urn} — returns full capability.
	mux.HandleFunc("/.well-known/agent/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(agentCap); err != nil {
			t.Errorf("failed to encode capability response: %v", err)
		}
	})

	client := NewClient(ts.URL)

	got, err := client.ResolveURN(context.Background(), testURNStr)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, agentCap.URN, got.URN)
	assert.Equal(t, agentCap.Title, got.Title)
	assert.Equal(t, agentCap.Endpoint, got.Endpoint)
	assert.Equal(t, agentCap.TrustLevel, got.TrustLevel)
	assert.Equal(t, agentCap.PublicKey, got.PublicKey)
	require.Len(t, got.Capabilities, 1)
	assert.Equal(t, "say-hello", got.Capabilities[0].ActionID)
}

// TestResolveURN_InvalidURN verifies that a malformed URN string causes an
// error before any network call is made.
func TestResolveURN_InvalidURN(t *testing.T) {
	client := NewClient("http://unused.example.com")

	_, err := client.ResolveURN(context.Background(), "not-a-valid-urn")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URN")
}

// TestResolveURN_CacheHit verifies that a second call with the same URN uses
// the cache and does not hit the server again.
func TestResolveURN_CacheHit(t *testing.T) {
	agentCap := buildCapability(testURNStr)

	var hitCount int32

	ts, mux := newMuxServer(t)

	mux.HandleFunc("/v1/agents/", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"endpoint": ts.URL,
		}); err != nil {
			t.Errorf("failed to encode endpoint response: %v", err)
		}
	})

	mux.HandleFunc("/.well-known/agent/", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(agentCap); err != nil {
			t.Errorf("failed to encode capability response: %v", err)
		}
	})

	client := NewClient(ts.URL).WithCacheTTL(5 * time.Minute)

	// First call — should hit the server (both registry and well-known = 2 hits).
	got1, err := client.ResolveURN(context.Background(), testURNStr)
	require.NoError(t, err)
	require.NotNil(t, got1)

	hitsAfterFirst := atomic.LoadInt32(&hitCount)
	assert.EqualValues(t, 2, hitsAfterFirst, "expected 2 server hits on first resolve")

	// Second call — cache should be hot; server hit count must not increase.
	got2, err := client.ResolveURN(context.Background(), testURNStr)
	require.NoError(t, err)
	require.NotNil(t, got2)

	hitsAfterSecond := atomic.LoadInt32(&hitCount)
	assert.Equal(t, hitsAfterFirst, hitsAfterSecond, "server should not be hit on cache hit")
	assert.Equal(t, got1.URN, got2.URN)
}

// TestResolveURN_RegistryError verifies that a non-OK response from the registry
// lookup endpoint causes ResolveURN to return an error.
func TestResolveURN_RegistryError(t *testing.T) {
	ts, mux := newMuxServer(t)

	mux.HandleFunc("/v1/agents/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	})

	client := NewClient(ts.URL)

	_, err := client.ResolveURN(context.Background(), testURNStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// TestResolveURN_URNMismatch verifies that when the well-known endpoint returns a
// capability whose URN does not match the requested URN, ResolveURN returns an error.
func TestResolveURN_URNMismatch(t *testing.T) {
	const mismatchedURN = "urn:agent:other.example.com/otherns:otheragent@v9.9.9#sha256:" + testHash

	ts, mux := newMuxServer(t)

	mux.HandleFunc("/v1/agents/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"endpoint": ts.URL,
		}); err != nil {
			t.Errorf("failed to encode endpoint response: %v", err)
		}
	})

	// Return a capability whose URN does not match the requested one.
	mux.HandleFunc("/.well-known/agent/", func(w http.ResponseWriter, _ *http.Request) {
		mismatchedCap := buildCapability(mismatchedURN)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(mismatchedCap); err != nil {
			t.Errorf("failed to encode capability response: %v", err)
		}
	})

	client := NewClient(ts.URL)

	_, err := client.ResolveURN(context.Background(), testURNStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
}

// TestClearCache verifies that after ClearCache, a subsequent ResolveURN call
// hits the server again rather than using the stale cache.
func TestClearCache(t *testing.T) {
	agentCap := buildCapability(testURNStr)

	var hitCount int32

	ts, mux := newMuxServer(t)

	mux.HandleFunc("/v1/agents/", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"endpoint": ts.URL,
		}); err != nil {
			t.Errorf("failed to encode endpoint response: %v", err)
		}
	})

	mux.HandleFunc("/.well-known/agent/", func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hitCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(agentCap); err != nil {
			t.Errorf("failed to encode capability response: %v", err)
		}
	})

	client := NewClient(ts.URL).WithCacheTTL(5 * time.Minute)

	// First resolve — populates cache.
	_, err := client.ResolveURN(context.Background(), testURNStr)
	require.NoError(t, err)

	hitsAfterFirst := atomic.LoadInt32(&hitCount)
	assert.EqualValues(t, 2, hitsAfterFirst, "expected 2 server hits on first resolve")

	// Clear the cache.
	client.ClearCache()

	// Second resolve — cache is empty, server must be hit again.
	got, err := client.ResolveURN(context.Background(), testURNStr)
	require.NoError(t, err)
	require.NotNil(t, got)

	hitsAfterSecond := atomic.LoadInt32(&hitCount)
	assert.EqualValues(t, 4, hitsAfterSecond, "expected 4 total server hits after cache clear")
	assert.Equal(t, testURNStr, got.URN)
}
