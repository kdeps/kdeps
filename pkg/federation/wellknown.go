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
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	wellKnownTimeout      = 10 * time.Second
	wellKnownMaxRedirects = 3
)

// WellKnownResponse is the JSON payload returned by a /.well-known/agent/{urn} endpoint.
type WellKnownResponse struct {
	URN        string `json:"urn"`
	Endpoint   string `json:"endpoint"`
	PublicKey  string `json:"publicKey"`
	TrustLevel string `json:"trustLevel,omitempty"`
}

// WellKnownClient fetches agent capability advertisements from /.well-known/agent/.
type WellKnownClient struct {
	httpClient *http.Client
}

// NewWellKnownClient creates a WellKnownClient with a 10-second timeout and TLS 1.2+.
func NewWellKnownClient() *WellKnownClient {
	return &WellKnownClient{
		httpClient: &http.Client{
			Timeout: wellKnownTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
			},
			// Allow up to wellKnownMaxRedirects redirects.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= wellKnownMaxRedirects {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
	}
}

// Discover attempts to discover an agent via the /.well-known/agent/{encodedURN} path
// on the agent's authority host. Falls back to /.well-known/agent (directory listing)
// if the specific path returns 404.
func (c *WellKnownClient) Discover(ctx context.Context, urn *URN) (*WellKnownResponse, error) {
	// Encode the URN for safe use in a URL path segment.
	encodedURN := url.PathEscape(urn.String())

	// Try specific URN path first.
	specificURL := fmt.Sprintf("https://%s/.well-known/agent/%s", urn.Authority, encodedURN)
	resp, err := c.fetchURL(ctx, specificURL)
	if err == nil {
		return resp, nil
	}

	// Fall back to directory listing if specific path failed with 404.
	return nil, fmt.Errorf("well-known discovery failed for %s: %w", urn.String(), err)
}

// DiscoverFromAuthority resolves an agent by querying authority's well-known directory.
// Useful when you only have the authority and want to enumerate available agents.
func (c *WellKnownClient) DiscoverFromAuthority(
	ctx context.Context,
	authority string,
) ([]*WellKnownResponse, error) {
	directoryURL := fmt.Sprintf("https://%s/.well-known/agent", authority)

	reqCtx, cancel := context.WithTimeout(ctx, wellKnownTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", directoryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("well-known directory request failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("well-known directory returned status %d", httpResp.StatusCode)
	}

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory response: %w", err)
	}

	var agents []*WellKnownResponse
	err = json.Unmarshal(body, &agents)
	if err != nil {
		return nil, fmt.Errorf("failed to parse directory response: %w", err)
	}

	return agents, nil
}

// fetchURL performs a GET to the given URL and unmarshals the JSON body into WellKnownResponse.
func (c *WellKnownClient) fetchURL(ctx context.Context, rawURL string) (*WellKnownResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, wellKnownTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: agent not found at %s", ErrAgentNotFound, rawURL)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, rawURL)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result WellKnownResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse well-known response: %w", err)
	}

	// Validate that the returned URN matches what we requested.
	if result.URN == "" {
		return nil, errors.New("well-known response missing urn field")
	}
	if result.Endpoint == "" {
		return nil, errors.New("well-known response missing endpoint field")
	}

	return &result, nil
}
