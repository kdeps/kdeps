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
//

package remote_agent

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/federation"
	"github.com/kdeps/kdeps/v2/pkg/federation/registry"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

// Adapter adapts the remote agent resource to the ResourceExecutor interface.
type Adapter struct {
	// registryClient for resolving agents
	registryClient *registry.Client
	// callerKeyManager for signing requests
	callerKeyManager *federation.KeyManager
	// callerURN identifies this kdeps installation as an agent
	callerURN string
}

// NewAdapter creates a new remote agent adapter.
func NewAdapter() *Adapter {
	// Determine key path
	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".config", "kdeps", "keys", "installation.key")

	// Load or create caller identity key
	km, err := federation.LoadOrCreate(keyPath)
	if err != nil {
		// Log but continue; some operations may still work if not requiring signatures
		// In production we'd require this, but for now we'll allow nil
		km = nil
	}

	// Default registry client will resolve URNs via direct endpoint or kdeps.io
	client := registry.NewClient("https://kdeps.io")

	return &Adapter{
		registryClient:   client,
		callerKeyManager: km,
		// callerURN will be set later when workflow metadata is available
	}
}

// setCallerURN sets the URN for this caller (can be derived from workflow).
func (a *Adapter) setCallerURN(urn string) {
	a.callerURN = urn
}

// Execute implements ResourceExecutor.
//
//nolint:funlen,cyclop,gocyclo,gocognit // complex but linear workflow
func (a *Adapter) Execute(ctx *executor.ExecutionContext, config interface{}) (interface{}, error) {
	cfg, ok := config.(*domain.RemoteAgentConfig)
	if !ok {
		return nil, errors.New("invalid config type for RemoteAgent executor")
	}

	// 1. Parse URN
	urn, err := federation.Parse(cfg.URN)
	if err != nil {
		return nil, fmt.Errorf("invalid URN: %w", err)
	}

	// 2. Resolve agent capability (endpoint, public key, trust level, schemas)
	agentCap, err := a.registryClient.ResolveURN(context.Background(), cfg.URN)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve agent: %w", err)
	}

	// 3. Check trust level if required
	if cfg.RequireTrustLevel != "" {
		// Simple comparison: trust levels are ordered self-attested < verified < certified
		// For now, require cap.TrustLevel >= required
		if !trustLevelSatisfies(agentCap.TrustLevel, cfg.RequireTrustLevel) {
			return nil, fmt.Errorf(
				"agent trust level %s does not meet required %s",
				agentCap.TrustLevel,
				cfg.RequireTrustLevel,
			)
		}
	}

	// 4. Validate and evaluate input payload against remote agent's input schema
	// For now, we only check required fields presence; detailed schema validation can be added later.
	if agentCap.Capabilities == nil || len(agentCap.Capabilities) == 0 {
		return nil, errors.New("agent capability has no actions defined")
	}
	// Evaluate input expressions
	evaluator := expression.NewEvaluator(ctx.API)
	env := a.buildEnvironment(ctx)
	inputs := make(map[string]interface{})
	for key, expr := range cfg.Input {
		// Take address of expr to pass to Evaluate (expects *domain.Expression)
		val, evalErr := evaluator.Evaluate(&expr, env)
		if evalErr != nil {
			return nil, fmt.Errorf("failed to evaluate input %s: %w", key, evalErr)
		}
		inputs[key] = val
	}

	// TODO: Input validation against remote agent's schema (when available)
	// For now, skip as Capability may only contain schema refs.

	// 5. Build signed UAF InvocationRequest
	msgID := uuid.New()
	callerURNStr := a.callerURN
	if callerURNStr == "" {
		// Derive a caller URN from workflow metadata if available
		callerURNStr = fmt.Sprintf("urn:agent:local/%s:%s@v0.0.0#sha256:unknown",
			ctx.Workflow.Metadata.Name, ctx.Workflow.Metadata.Version)
	}
	req := federation.InvocationRequest{
		MessageID: msgID,
		Timestamp: time.Now().UTC(),
		Caller: federation.CallerIdentity{
			URN:       callerURNStr,
			PublicKey: a.publicKeyString(),
		},
		Callee: federation.CalleeIdentity{
			URN: urn.String(),
		},
		Payload: federation.RequestPayload{
			Inputs: inputs,
			Context: federation.InvocationContext{
				RequestID:     uuid.New().String(),
				CorrelationID: "", // optional
				CallerIP:      "", // could fill from request
			},
		},
	}

	// Marshal request JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Sign request if we have a private key
	var signature []byte
	if a.callerKeyManager != nil {
		signature, err = a.callerKeyManager.Sign(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to sign request: %w", err)
		}
	}

	// 6. Send HTTP/2 POST to endpoint
	endpointURL := fmt.Sprintf("%s/.uaf/v1/invoke", agentCap.Endpoint)
	httpReq, err := http.NewRequestWithContext(
		context.Background(),
		"POST",
		endpointURL,
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Uaf-Version", "1.0")
	httpReq.Header.Set("X-Uaf-Message-Id", msgID.String())
	httpReq.Header.Set("X-Uaf-Caller-Urn", callerURNStr)
	httpReq.Header.Set("X-Uaf-Caller-Public-Key", a.publicKeyString())
	if signature != nil {
		httpReq.Header.Set("X-Uaf-Signature", hex.EncodeToString(signature))
	}

	// Parse timeout
	timeoutDur := parseTimeout(cfg.Timeout)
	// Configure HTTP/2 client
	httpClient := &http.Client{
		Timeout: timeoutDur,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
	}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		// Try fallback agents if defined
		if cfg.Fallback != nil && len(cfg.Fallback) > 0 {
			return a.tryFallbacks(cfg.Fallback, cfg, ctx)
		}
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 7. Verify receipt
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"remote agent returned status %d: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	// Parse receipt from header
	receiptB64 := resp.Header.Get("X-Uaf-Receipt")
	sigB64 := resp.Header.Get("X-Uaf-Receipt-Signature")
	if receiptB64 == "" || sigB64 == "" {
		return nil, errors.New("missing receipt or signature in response")
	}
	receiptJSON, err := base64.StdEncoding.DecodeString(receiptB64)
	if err != nil {
		return nil, fmt.Errorf("invalid receipt encoding: %w", err)
	}
	signature, err = hex.DecodeString(sigB64)
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}

	// Parse receipt struct
	var rec federation.Receipt
	err = json.Unmarshal(receiptJSON, &rec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse receipt: %w", err)
	}

	// Verify signature using callee's public key from capability
	calleePubKeyPEM := agentCap.PublicKey
	// Convert PEM to ed25519.PublicKey
	pubKey, err := parseEd25519PublicKey([]byte(calleePubKeyPEM))
	if err != nil {
		return nil, fmt.Errorf("failed to parse callee public key: %w", err)
	}
	if !ed25519.Verify(pubKey, receiptJSON, signature) {
		return nil, errors.New("receipt signature verification failed")
	}

	// Check receipt matches request
	if rec.MessageID != msgID {
		return nil, errors.New("receipt message ID mismatch")
	}
	if rec.Caller.String() != callerURNStr {
		return nil, fmt.Errorf(
			"receipt caller URN mismatch: expected %s, got %s",
			callerURNStr,
			rec.Caller.String(),
		)
	}
	if rec.Callee.String() != urn.String() {
		return nil, fmt.Errorf(
			"receipt callee URN mismatch: expected %s, got %s",
			urn.String(),
			rec.Callee.String(),
		)
	}

	// 8. Return outputs
	if rec.Execution.Status != "success" {
		errMsg := "remote agent execution failed"
		if rec.Execution.Error != nil {
			errMsg = fmt.Sprintf("%s: %s", errMsg, rec.Execution.Error.Message)
		}
		return nil, errors.New(errMsg)
	}

	return rec.Execution.Outputs, nil
}

// tryFallbacks attempts each fallback agent in order until one succeeds.
func (a *Adapter) tryFallbacks(
	fallbacks []domain.FallbackConfig,
	primary *domain.RemoteAgentConfig,
	ctx *executor.ExecutionContext,
) (interface{}, error) {
	var lastErr error
	for _, fb := range fallbacks {
		timeout := fb.Timeout
		if timeout == "" {
			timeout = primary.Timeout
		}
		fallbackCfg := &domain.RemoteAgentConfig{
			URN:               fb.URN,
			Input:             primary.Input,
			Timeout:           timeout,
			RequireTrustLevel: primary.RequireTrustLevel,
		}
		result, err := a.Execute(ctx, fallbackCfg)
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, fmt.Errorf("all fallback agents failed, last error: %w", lastErr)
	}
	return nil, errors.New("no fallback agents configured")
}

// buildEnvironment creates an evaluation environment from the execution context.
func (a *Adapter) buildEnvironment(ctx *executor.ExecutionContext) map[string]interface{} {
	env := make(map[string]interface{})

	// Include request context if available
	if ctx.Request != nil {
		env["request"] = map[string]interface{}{
			"method":  ctx.Request.Method,
			"path":    ctx.Request.Path,
			"headers": ctx.Request.Headers,
			"query":   ctx.Request.Query,
			"body":    ctx.Request.Body,
		}
		if ctx.Request.Body != nil {
			env["input"] = ctx.Request.Body
		}
	}

	// Include previously computed outputs
	env["outputs"] = ctx.Outputs

	// Include items iteration context
	if item, ok := ctx.Items["item"]; ok {
		env["item"] = item
	}

	return env
}

// publicKeyString returns the caller's public key in the format "ed25519:<hex>".
func (a *Adapter) publicKeyString() string {
	if a.callerKeyManager == nil {
		return ""
	}
	pub := a.callerKeyManager.PublicKey()
	return "ed25519:" + hex.EncodeToString(pub[:])
}

const (
	trustLevelVerified  = 2
	trustLevelCertified = 3
	defaultTimeoutSecs  = 60
)

// trustLevelSatisfies returns true if candidate >= required in trust hierarchy.
func trustLevelSatisfies(candidate, required string) bool {
	levels := map[string]int{
		"self-attested": 1,
		"verified":      trustLevelVerified,
		"certified":     trustLevelCertified,
	}
	cand, ok1 := levels[candidate]
	req, ok2 := levels[required]
	if !ok1 || !ok2 {
		return false
	}
	return cand >= req
}

// parseEd25519PublicKey decodes a PEM-encoded Ed25519 public key.
func parseEd25519PublicKey(pemData []byte) (ed25519.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "ED25519 PUBLIC KEY" {
		return nil, errors.New("invalid PEM block for public key")
	}
	if len(block.Bytes) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key length")
	}
	return ed25519.PublicKey(block.Bytes), nil
}

// parseTimeout converts a timeout string (e.g., "60s") to time.Duration.
// Defaults to 60 seconds if empty or invalid.
func parseTimeout(timeout string) time.Duration {
	if timeout == "" {
		return defaultTimeoutSecs * time.Second
	}
	d, err := time.ParseDuration(timeout)
	if err != nil {
		return defaultTimeoutSecs * time.Second
	}
	return d
}
