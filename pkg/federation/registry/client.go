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

package registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/federation"
)

// AgentCapability describes a remote agent's interface and trust information.
type AgentCapability struct {
	URN         string   `json:"urn"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Endpoint    string   `json:"endpoint"` // URL to invoke
	AuthMethods []string `json:"authMethods"`
	TrustLevel  string   `json:"trustLevel"` // self-attested, verified, certified
	PublicKey   string   `json:"publicKey"`   // PEM format
	RateLimit   int      `json:"rateLimit"`

	// Capabilities list (actionId, schemas)
	Capabilities []Capability `json:"capabilities"`
}

// Capability describes a single action that the agent can perform.
type Capability struct {
	ActionID       string      `json:"actionId"`
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	InputSchemaRef string      `json:"inputSchemaRef"` // JSON Schema URL or local ref
	OutputSchemaRef string     `json:"outputSchemaRef"`
	InputExample   interface{} `json:"inputExample,omitempty"`
	OutputExample  interface{} `json:"outputExample,omitempty"`
}

// Client is a UAF registry client with caching.
type Client struct {
	baseURL    string
	httpClient *http.Client
	cache      map[string]*cachedEntry
	mu         sync.RWMutex
	ttl        time.Duration
}

type cachedEntry struct {
	capability *AgentCapability
	expiresAt  time.Time
}

// NewClient creates a new registry client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			},
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]*cachedEntry),
		ttl:   5 * time.Minute,
	}
}

// WithCacheTTL sets the cache TTL (default 5 minutes).
func (c *Client) WithCacheTTL(ttl time.Duration) *Client {
	c.ttl = ttl
	return c
}

// ResolveURN resolves a URN to an AgentCapability.
// It uses the cache if entry is fresh.
func (c *Client) ResolveURN(ctx context.Context, urnStr string) (*AgentCapability, error) {
	urn, err := federation.Parse(urnStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URN: %w", err)
	}
	urnKey := urn.String()

	// Check cache
	c.mu.RLock()
	if entry, ok := c.cache[urnKey]; ok && time.Now().Before(entry.expiresAt) {
		c.mu.RUnlock()
		return entry.capability, nil
	}
	c.mu.RUnlock()

	// Determine resolution method
	var endpoint string
	if c.isLocalhost(urn.Authority) {
		// Direct endpoint: authority is host:port
		endpoint = fmt.Sprintf("https://%s/.uaf/v1/invoke", urn.Authority)
	} else {
		// Query registry
		var err error
		endpoint, err = c.lookupEndpoint(ctx, urn)
		if err != nil {
			return nil, err
		}
	}

	// Fetch capability from the endpoint (or from well-known)
	cap, err := c.fetchCapability(ctx, urn, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch capability: %w", err)
	}

	// Cache it
	c.mu.Lock()
	c.cache[urnKey] = &cachedEntry{
		capability: cap,
		expiresAt:  time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return cap, nil
}

// isLocalhost checks if the authority indicates a direct endpoint (localhost or IP with port).
func (c *Client) isLocalhost(authority string) bool {
	// Simple heuristic: contains ':' (port) and is localhost or 127.0.0.1 or [::1]
	return authority == "localhost" || authority == "127.0.0.1" || authority == "[::1]"
}

// lookupEndpoint queries the registry to find the agent's endpoint.
func (c *Client) lookupEndpoint(ctx context.Context, urn *federation.URN) (string, error) {
	// Construct registry lookup URL
	// For now, simple GET /v1/agents/{urn-encoded}
	url := fmt.Sprintf("%s/v1/agents/%s", c.baseURL, urn.String())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var result struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.Endpoint == "" {
		return "", fmt.Errorf("agent not found or no endpoint")
	}
	return result.Endpoint, nil
}

// fetchCapability retrieves the agent's capability description from its endpoint.
func (c *Client) fetchCapability(ctx context.Context, urn *federation.URN, endpoint string) (*AgentCapability, error) {
	// Try the well-known endpoint first: /.well-known/agent/{urn-encoded}
	wellKnownURL := fmt.Sprintf("%s/.well-known/agent/%s", endpoint, urn.String())

	req, err := http.NewRequestWithContext(ctx, "GET", wellKnownURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Could fall back to registry's capability endpoint
		return nil, fmt.Errorf("well-known request failed: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var cap AgentCapability
	if err := json.Unmarshal(body, &cap); err != nil {
		return nil, err
	}

	// Verify URN matches
	if cap.URN != urn.String() {
		return nil, fmt.Errorf("capability URN mismatch")
	}

	return &cap, nil
}

// InvalidateCache removes a specific URN from the cache.
func (c *Client) InvalidateCache(urnStr string) {
	c.mu.Lock()
	delete(c.cache, urnStr)
	c.mu.Unlock()
}

// ClearCache empties the entire cache.
func (c *Client) ClearCache() {
	c.mu.Lock()
	c.cache = make(map[string]*cachedEntry)
	c.mu.Unlock()
}
