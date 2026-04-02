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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSHA256Hash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// redirectTransport rewrites all requests to point at a specific httptest.Server,
// changing scheme to http. This allows tests to intercept the https:// calls
// that WellKnownClient makes.
type redirectTransport struct {
	target *url.URL
}

func (r *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = r.target.Host
	return http.DefaultTransport.RoundTrip(req2)
}

// newTestWellKnownClient creates a WellKnownClient whose HTTP calls are redirected
// to the given httptest.Server.
func newTestWellKnownClient(ts *httptest.Server) *WellKnownClient {
	parsed, err := url.Parse(ts.URL)
	if err != nil {
		panic(err)
	}
	wkc := NewWellKnownClient()
	wkc.httpClient = &http.Client{
		Transport: &redirectTransport{target: parsed},
	}
	return wkc
}

// TestWellKnownDiscover_Success verifies that a valid JSON response is correctly
// parsed and returned when the server responds with 200 OK.
func TestWellKnownDiscover_Success(t *testing.T) {
	const urnStr = "urn:agent:agents.example.com/myns:myagent@v1.0.0#sha256:" + testSHA256Hash

	urn, err := Parse(urnStr)
	require.NoError(t, err)

	expected := WellKnownResponse{
		URN:        urnStr,
		Endpoint:   "https://agents.example.com/invoke",
		PublicKey:  "ed25519:test",
		TrustLevel: "verified",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		encErr := json.NewEncoder(w).Encode(expected)
		if encErr != nil {
			t.Errorf("failed to encode response: %v", encErr)
		}
	}))
	defer ts.Close()

	wkc := newTestWellKnownClient(ts)

	got, err := wkc.Discover(context.Background(), urn)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, expected.URN, got.URN)
	assert.Equal(t, expected.Endpoint, got.Endpoint)
	assert.Equal(t, expected.PublicKey, got.PublicKey)
	assert.Equal(t, expected.TrustLevel, got.TrustLevel)
}

// TestWellKnownDiscover_NotFound verifies that a 404 response produces an error
// wrapping ErrAgentNotFound.
func TestWellKnownDiscover_NotFound(t *testing.T) {
	const urnStr = "urn:agent:agents.example.com/myns:myagent@v1.0.0#sha256:" + testSHA256Hash

	urn, err := Parse(urnStr)
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer ts.Close()

	wkc := newTestWellKnownClient(ts)

	_, err = wkc.Discover(context.Background(), urn)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrAgentNotFound), "expected ErrAgentNotFound, got: %v", err)
}

// TestWellKnownDiscover_InvalidJSON verifies that malformed JSON in the response
// body results in a parse error.
func TestWellKnownDiscover_InvalidJSON(t *testing.T) {
	const urnStr = "urn:agent:agents.example.com/myns:myagent@v1.0.0#sha256:" + testSHA256Hash

	urn, err := Parse(urnStr)
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer ts.Close()

	wkc := newTestWellKnownClient(ts)

	_, err = wkc.Discover(context.Background(), urn)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "parse"), "expected parse error, got: %v", err)
}

// TestWellKnownDiscover_MissingURN verifies that a JSON response with an empty
// "urn" field produces a validation error.
func TestWellKnownDiscover_MissingURN(t *testing.T) {
	const urnStr = "urn:agent:agents.example.com/myns:myagent@v1.0.0#sha256:" + testSHA256Hash

	urn, err := Parse(urnStr)
	require.NoError(t, err)

	payload := WellKnownResponse{
		URN:      "",
		Endpoint: "https://agents.example.com/invoke",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		encErr := json.NewEncoder(w).Encode(payload)
		if encErr != nil {
			t.Errorf("failed to encode response: %v", encErr)
		}
	}))
	defer ts.Close()

	wkc := newTestWellKnownClient(ts)

	_, err = wkc.Discover(context.Background(), urn)
	require.Error(t, err)
	assert.True(
		t,
		strings.Contains(err.Error(), "urn"),
		"expected error mentioning urn field, got: %v",
		err,
	)
}

// TestWellKnownDiscover_MissingEndpoint verifies that a JSON response with an
// empty "endpoint" field produces a validation error.
func TestWellKnownDiscover_MissingEndpoint(t *testing.T) {
	const urnStr = "urn:agent:agents.example.com/myns:myagent@v1.0.0#sha256:" + testSHA256Hash

	urn, err := Parse(urnStr)
	require.NoError(t, err)

	payload := WellKnownResponse{
		URN:      urnStr,
		Endpoint: "",
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		encErr := json.NewEncoder(w).Encode(payload)
		if encErr != nil {
			t.Errorf("failed to encode response: %v", encErr)
		}
	}))
	defer ts.Close()

	wkc := newTestWellKnownClient(ts)

	_, err = wkc.Discover(context.Background(), urn)
	require.Error(t, err)
	assert.True(
		t,
		strings.Contains(err.Error(), "endpoint"),
		"expected error mentioning endpoint field, got: %v",
		err,
	)
}

// TestWellKnownDiscoverFromAuthority_Success verifies that the directory listing
// endpoint returns a slice of WellKnownResponse values.
func TestWellKnownDiscoverFromAuthority_Success(t *testing.T) {
	agents := []*WellKnownResponse{
		{
			URN:       "urn:agent:agents.example.com/myns:agentA@v1.0.0#sha256:" + testSHA256Hash,
			Endpoint:  "https://agents.example.com/a",
			PublicKey: "ed25519:keyA",
		},
		{
			URN:       "urn:agent:agents.example.com/myns:agentB@v2.0.0#sha256:" + testSHA256Hash,
			Endpoint:  "https://agents.example.com/b",
			PublicKey: "ed25519:keyB",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/.well-known/agent", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(agents); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer ts.Close()

	wkc := newTestWellKnownClient(ts)

	got, err := wkc.DiscoverFromAuthority(context.Background(), "agents.example.com")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, agents[0].URN, got[0].URN)
	assert.Equal(t, agents[0].Endpoint, got[0].Endpoint)
	assert.Equal(t, agents[1].URN, got[1].URN)
	assert.Equal(t, agents[1].Endpoint, got[1].Endpoint)
}

// TestWellKnownDiscoverFromAuthority_Error verifies that a non-OK status code
// from the directory endpoint produces an error.
func TestWellKnownDiscoverFromAuthority_Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	wkc := newTestWellKnownClient(ts)

	_, err := wkc.DiscoverFromAuthority(context.Background(), "agents.example.com")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "500"), "expected HTTP 500 in error, got: %v", err)
}

// TestNewWellKnownClient_CheckRedirect verifies that the redirect limit is enforced.
func TestNewWellKnownClient_CheckRedirect(t *testing.T) {
	// Build a chain of redirect servers longer than wellKnownMaxRedirects (5).
	const redirectCount = 10
	servers := make([]*httptest.Server, redirectCount)

	for i := redirectCount - 1; i >= 0; i-- {
		idx := i
		// Capture servers slice so closures reference the final slice.
		localServers := servers
		servers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if idx+1 < len(localServers) && localServers[idx+1] != nil {
				http.Redirect(w, r, localServers[idx+1].URL, http.StatusFound)
			} else {
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer servers[i].Close()
	}

	// Use a real NewWellKnownClient (not the test redirect wrapper) to exercise CheckRedirect.
	wkc := NewWellKnownClient()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, servers[0].URL, nil)
	require.NoError(t, err)

	resp, err := wkc.httpClient.Do(req)
	if resp != nil {
		resp.Body.Close()
	}
	// After wellKnownMaxRedirects hops the client should refuse with an error.
	assert.Error(t, err)
}
