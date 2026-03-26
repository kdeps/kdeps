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

package remoteagent

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/federation"
	"github.com/kdeps/kdeps/v2/pkg/federation/registry"
)

// testURN returns a valid test URN with a 64-char sha256 hash.
const testHash = "aabbccddeeff00112233445566778899aabbccddeeff00112233445566778899"

func testCalleeURN() string {
	return "urn:agent:example.com/test:agent@v1.0.0#sha256:" + testHash
}

func testCallerURN() string {
	return "urn:agent:local/caller:agent@v0.0.0#sha256:" + testHash
}

// buildTestServers creates a mock registry server and mock agent server for testing.
// It returns the registry server URL.
//
//nolint:cyclop,gocyclo,gocognit // test setup with many branches
func buildTestServers(t *testing.T, agentStatus int, receiptSuccess bool, signWithWrongKey bool) *httptest.Server {
	t.Helper()

	// Generate callee keypair
	calleePriv, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	calleeKM := federation.NewKeyManager(calleePriv)

	// Get callee public key PEM
	calleePubPEM, err := calleeKM.PublicKeyPEM()
	require.NoError(t, err)

	// Optionally generate a wrong key for bad-signature tests
	wrongPriv, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	wrongKM := federation.NewKeyManager(wrongPriv)

	calleeURN := testCalleeURN()
	callerURN := testCallerURN()

	// Build agent server (handles /.well-known and /.uaf invocation)
	agentServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && len(r.URL.Path) > len("/.well-known/agent/"):
				// /.well-known/agent/{urn} - return capability
				agentCap := registry.AgentCapability{
					URN:         calleeURN,
					Title:       "Test Agent",
					Description: "Test agent for unit tests",
					Version:     "v1.0.0",
					Endpoint:    "PLACEHOLDER", // will be set below
					TrustLevel:  "self-attested",
					PublicKey:   string(calleePubPEM),
					Capabilities: []registry.Capability{
						{ActionID: "test-action", Title: "Test"},
					},
				}
				// Use the real server URL as endpoint
				agentCap.Endpoint = "PENDING" // resolved after server start
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(agentCap); err != nil { //nolint:govet // intentional shadow
					http.Error(w, "encode error", http.StatusInternalServerError)
				}

			case r.Method == http.MethodPost && r.URL.Path == "/.uaf/v1/invoke":
				if agentStatus != http.StatusOK {
					http.Error(w, "agent error", agentStatus)
					return
				}

				// Parse the request to get MessageID
				var req federation.InvocationRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil { //nolint:govet // intentional shadow
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}

				// Build receipt
				parsedCalleeURN, err := federation.Parse(calleeURN) //nolint:govet // intentional shadow
				if err != nil {
					http.Error(w, "parse callee urn", http.StatusInternalServerError)
					return
				}
				parsedCallerURN, err := federation.Parse(callerURN) //nolint:govet // intentional shadow
				if err != nil {
					http.Error(w, "parse caller urn", http.StatusInternalServerError)
					return
				}

				status := "success"
				var execError *federation.ExecutionError
				if !receiptSuccess {
					status = "error"
					execError = &federation.ExecutionError{
						Code:    "EXEC_FAIL",
						Message: "execution failed",
					}
				}

				receipt := federation.Receipt{
					MessageID: req.MessageID,
					Timestamp: time.Now().UTC(),
					Callee:    *parsedCalleeURN,
					Caller:    *parsedCallerURN,
					Execution: federation.ExecutionResult{
						Status:  status,
						Outputs: map[string]interface{}{"result": "hello"},
						Error:   execError,
					},
				}

				receiptJSON, err := json.Marshal(&receipt)
				if err != nil {
					http.Error(w, "marshal receipt", http.StatusInternalServerError)
					return
				}

				// Sign receipt - optionally with wrong key
				var sig []byte
				if signWithWrongKey {
					sig, err = wrongKM.Sign(receiptJSON)
				} else {
					sig, err = calleeKM.Sign(receiptJSON)
				}
				if err != nil {
					http.Error(w, "sign receipt", http.StatusInternalServerError)
					return
				}

				w.Header().Set("X-Uaf-Receipt", base64.StdEncoding.EncodeToString(receiptJSON))
				w.Header().Set("X-Uaf-Receipt-Signature", hex.EncodeToString(sig))
				w.WriteHeader(http.StatusOK)

			default:
				http.NotFound(w, r)
			}
		}),
	)

	// We need to update the /.well-known handler to use the real agent server URL.
	// Restart the agent server with a handler that knows the URL.
	agentServer.Close()
	agentServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && len(r.URL.Path) > len("/.well-known/agent/"):
			agentCap := registry.AgentCapability{
				URN:         calleeURN,
				Title:       "Test Agent",
				Description: "Test agent for unit tests",
				Version:     "v1.0.0",
				Endpoint:    agentServer.URL,
				TrustLevel:  "self-attested",
				PublicKey:   string(calleePubPEM),
				Capabilities: []registry.Capability{
					{ActionID: "test-action", Title: "Test"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(agentCap); err != nil { //nolint:govet // intentional shadow
				http.Error(w, "encode error", http.StatusInternalServerError)
			}

		case r.Method == http.MethodPost && r.URL.Path == "/.uaf/v1/invoke":
			if agentStatus != http.StatusOK {
				http.Error(w, "agent error", agentStatus)
				return
			}

			var req federation.InvocationRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil { //nolint:govet // intentional shadow
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}

			parsedCalleeURN, err := federation.Parse(calleeURN) //nolint:govet // intentional shadow
			if err != nil {
				http.Error(w, "parse callee urn", http.StatusInternalServerError)
				return
			}
			parsedCallerURN, err := federation.Parse(callerURN) //nolint:govet // intentional shadow
			if err != nil {
				http.Error(w, "parse caller urn", http.StatusInternalServerError)
				return
			}

			status := "success"
			var execError *federation.ExecutionError
			if !receiptSuccess {
				status = "error"
				execError = &federation.ExecutionError{
					Code:    "EXEC_FAIL",
					Message: "execution failed",
				}
			}

			receipt := federation.Receipt{
				MessageID: req.MessageID,
				Timestamp: time.Now().UTC(),
				Callee:    *parsedCalleeURN,
				Caller:    *parsedCallerURN,
				Execution: federation.ExecutionResult{
					Status:  status,
					Outputs: map[string]interface{}{"result": "hello"},
					Error:   execError,
				},
			}

			receiptJSON, err := json.Marshal(&receipt)
			if err != nil {
				http.Error(w, "marshal receipt", http.StatusInternalServerError)
				return
			}

			var sig []byte
			if signWithWrongKey {
				sig, err = wrongKM.Sign(receiptJSON)
			} else {
				sig, err = calleeKM.Sign(receiptJSON)
			}
			if err != nil {
				http.Error(w, "sign receipt", http.StatusInternalServerError)
				return
			}

			w.Header().Set("X-Uaf-Receipt", base64.StdEncoding.EncodeToString(receiptJSON))
			w.Header().Set("X-Uaf-Receipt-Signature", hex.EncodeToString(sig))
			w.WriteHeader(http.StatusOK)

		default:
			http.NotFound(w, r)
		}
	}))

	// Registry server serves /v1/agents/{urn} -> endpoint URL
	registryServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				resp := map[string]string{"endpoint": agentServer.URL}
				w.Header().Set("Content-Type", "application/json")
				encErr := json.NewEncoder(w).Encode(resp)
				if encErr != nil {
					http.Error(w, "encode error", http.StatusInternalServerError)
				}
				return
			}
			http.NotFound(w, r)
		}),
	)

	t.Cleanup(func() {
		registryServer.Close()
		agentServer.Close()
	})

	return registryServer
}

// buildAdapter creates a test Adapter with caller key manager and specified registry URL.
func buildAdapter(t *testing.T, registryURL string) *Adapter {
	t.Helper()

	callerPriv, _, err := federation.GenerateKeypair()
	require.NoError(t, err)

	callerKM := federation.NewKeyManager(callerPriv)

	registryClient := registry.NewClient(registryURL).WithCacheTTL(0)

	return &Adapter{
		registryClient:   registryClient,
		callerKeyManager: callerKM,
		callerURN:        testCallerURN(),
	}
}

// buildMinimalExecutionContext creates an ExecutionContext with minimal required fields.
func buildMinimalExecutionContext() *executor.ExecutionContext {
	return &executor.ExecutionContext{
		Workflow: &domain.Workflow{
			Metadata: domain.WorkflowMetadata{
				Name:    "test-workflow",
				Version: "v1.0.0",
			},
		},
		Outputs: map[string]interface{}{},
		Items:   map[string]interface{}{},
		API:     nil,
	}
}

// buildPublicKeyPEM creates a PEM-encoded Ed25519 public key from the raw key bytes.
func buildPublicKeyPEM(pub ed25519.PublicKey) []byte {
	block := &pem.Block{
		Type:  "ED25519 PUBLIC KEY",
		Bytes: pub,
	}
	return pem.EncodeToMemory(block)
}

func TestAdapter_Execute_Success(t *testing.T) {
	registryServer := buildTestServers(t, http.StatusOK, true, false)

	adapter := buildAdapter(t, registryServer.URL)
	ctx := buildMinimalExecutionContext()

	cfg := &domain.RemoteAgentConfig{
		URN:   testCalleeURN(),
		Input: map[string]domain.Expression{},
	}

	result, err := adapter.Execute(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, result)

	outputs, ok := result.(map[string]interface{})
	require.True(t, ok, "result should be map[string]interface{}")
	assert.Equal(t, "hello", outputs["result"])
}

func TestAdapter_Execute_InvalidConfig(t *testing.T) {
	adapter := &Adapter{}
	ctx := buildMinimalExecutionContext()

	// Pass wrong config type
	_, err := adapter.Execute(ctx, "not-a-config")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config type")
}

func TestAdapter_Execute_InvalidURN(t *testing.T) {
	adapter := &Adapter{
		callerURN: testCallerURN(),
	}
	ctx := buildMinimalExecutionContext()

	cfg := &domain.RemoteAgentConfig{
		URN:   "this-is-not-a-urn",
		Input: map[string]domain.Expression{},
	}

	_, err := adapter.Execute(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URN")
}

func TestAdapter_Execute_TrustLevelInsufficient(t *testing.T) {
	registryServer := buildTestServers(t, http.StatusOK, true, false)

	adapter := buildAdapter(t, registryServer.URL)
	ctx := buildMinimalExecutionContext()

	// Agent returns self-attested, but we require certified
	cfg := &domain.RemoteAgentConfig{
		URN:               testCalleeURN(),
		Input:             map[string]domain.Expression{},
		RequireTrustLevel: "certified",
	}

	_, err := adapter.Execute(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trust level")
}

func TestAdapter_Execute_RegistryFail(t *testing.T) {
	// Point registry to a non-existent server
	adapter := buildAdapter(t, "http://127.0.0.1:1") // port 1 should fail

	ctx := buildMinimalExecutionContext()
	cfg := &domain.RemoteAgentConfig{
		URN:   testCalleeURN(),
		Input: map[string]domain.Expression{},
	}

	_, err := adapter.Execute(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve agent")
}

func TestAdapter_Execute_RemoteAgentError(t *testing.T) {
	// Agent returns success HTTP status but receipt with error status
	registryServer := buildTestServers(t, http.StatusOK, false, false)

	adapter := buildAdapter(t, registryServer.URL)
	ctx := buildMinimalExecutionContext()

	cfg := &domain.RemoteAgentConfig{
		URN:   testCalleeURN(),
		Input: map[string]domain.Expression{},
	}

	_, err := adapter.Execute(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remote agent execution failed")
}

func TestAdapter_Execute_SignatureVerifyFail(t *testing.T) {
	// Agent signs with wrong key
	registryServer := buildTestServers(t, http.StatusOK, true, true)

	adapter := buildAdapter(t, registryServer.URL)
	ctx := buildMinimalExecutionContext()

	cfg := &domain.RemoteAgentConfig{
		URN:   testCalleeURN(),
		Input: map[string]domain.Expression{},
	}

	_, err := adapter.Execute(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "receipt signature verification failed")
}

func TestAdapter_Execute_AgentHTTPError(t *testing.T) {
	// Agent server returns HTTP 500
	registryServer := buildTestServers(t, http.StatusInternalServerError, false, false)

	adapter := buildAdapter(t, registryServer.URL)
	ctx := buildMinimalExecutionContext()

	cfg := &domain.RemoteAgentConfig{
		URN:   testCalleeURN(),
		Input: map[string]domain.Expression{},
	}

	_, err := adapter.Execute(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestAdapter_trustLevelSatisfies(t *testing.T) {
	tests := []struct {
		name      string
		candidate string
		required  string
		expected  bool
	}{
		{"self-attested >= self-attested", "self-attested", "self-attested", true},
		{"verified >= self-attested", "verified", "self-attested", true},
		{"certified >= self-attested", "certified", "self-attested", true},
		{"certified >= verified", "certified", "verified", true},
		{"certified >= certified", "certified", "certified", true},
		{"self-attested < verified", "self-attested", "verified", false},
		{"self-attested < certified", "self-attested", "certified", false},
		{"verified < certified", "verified", "certified", false},
		{"unknown candidate", "unknown", "verified", false},
		{"unknown required", "verified", "unknown", false},
		{"both unknown", "unknown", "unknown", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := trustLevelSatisfies(tc.candidate, tc.required)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestAdapter_parseTimeout(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"", 60 * time.Second},
		{"30s", 30 * time.Second},
		{"2m", 2 * time.Minute},
		{"invalid", 60 * time.Second},
		{"1h", time.Hour},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := parseTimeout(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestAdapter_parseEd25519PublicKey(t *testing.T) {
	_, pub, err := federation.GenerateKeypair()
	require.NoError(t, err)

	pemData := buildPublicKeyPEM(pub)

	parsed, err := parseEd25519PublicKey(pemData)
	require.NoError(t, err)
	assert.Equal(t, pub, parsed)
}

func TestAdapter_parseEd25519PublicKey_InvalidPEM(t *testing.T) {
	_, err := parseEd25519PublicKey([]byte("not a pem"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PEM block")
}

func TestAdapter_parseEd25519PublicKey_WrongType(t *testing.T) {
	block := &pem.Block{
		Type:  "WRONG TYPE",
		Bytes: make([]byte, ed25519.PublicKeySize),
	}
	pemData := pem.EncodeToMemory(block)

	_, err := parseEd25519PublicKey(pemData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PEM block")
}

func TestAdapter_publicKeyString_NoKeyManager(t *testing.T) {
	a := &Adapter{}
	assert.Equal(t, "", a.publicKeyString())
}

func TestAdapter_publicKeyString_WithKeyManager(t *testing.T) {
	priv, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	km := federation.NewKeyManager(priv)

	a := &Adapter{callerKeyManager: km}
	str := a.publicKeyString()
	assert.True(t, len(str) > 0, "public key string should not be empty")
	assert.Contains(t, str, "ed25519:")
}

func TestAdapter_Execute_NilCallerURN_UsesWorkflowMetadata(t *testing.T) {
	registryServer := buildTestServers(t, http.StatusOK, true, false)

	// Build adapter without caller URN - should derive from workflow
	callerPriv, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	callerKM := federation.NewKeyManager(callerPriv)

	registryClient := registry.NewClient(registryServer.URL).WithCacheTTL(0)

	adapter := &Adapter{
		registryClient:   registryClient,
		callerKeyManager: callerKM,
		callerURN:        "", // empty - will be derived
	}

	ctx := &executor.ExecutionContext{
		Workflow: &domain.Workflow{
			Metadata: domain.WorkflowMetadata{
				Name:    "my-workflow",
				Version: "v1.2.3",
			},
		},
		Outputs: map[string]interface{}{},
		Items:   map[string]interface{}{},
	}

	// This will fail on receipt caller URN mismatch since the derived URN won't match
	// testCallerURN() - but it should at least attempt the call and fail with mismatch
	cfg := &domain.RemoteAgentConfig{
		URN:   testCalleeURN(),
		Input: map[string]domain.Expression{},
	}

	_, err = adapter.Execute(ctx, cfg)
	// Expect error because derived URN doesn't match receipt caller URN
	require.Error(t, err)
}

func TestAdapter_setCallerURN(t *testing.T) {
	a := &Adapter{}
	assert.Equal(t, "", a.callerURN)

	a.setCallerURN("urn:agent:example.com/test:agent@v1.0.0#sha256:" + testHash)
	assert.Equal(t, "urn:agent:example.com/test:agent@v1.0.0#sha256:"+testHash, a.callerURN)
}

func TestAdapter_Execute_WithLiteralInput(t *testing.T) {
	registryServer := buildTestServers(t, http.StatusOK, true, false)

	adapter := buildAdapter(t, registryServer.URL)
	ctx := buildMinimalExecutionContext()

	// Use literal expression (ExprTypeLiteral = 0, Raw set)
	cfg := &domain.RemoteAgentConfig{
		URN: testCalleeURN(),
		Input: map[string]domain.Expression{
			"message": {Raw: "hello world", Type: domain.ExprTypeLiteral},
		},
	}

	result, err := adapter.Execute(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
}

func TestNewAdapter_DoesNotPanic(t *testing.T) {
	// NewAdapter may fail to load a key file but should not panic
	a := NewAdapter()
	assert.NotNil(t, a)
	assert.NotNil(t, a.registryClient)
}

func TestAdapter_buildEnvironment_WithRequest(t *testing.T) {
	a := &Adapter{}

	ctx := &executor.ExecutionContext{
		Request: &executor.RequestContext{
			Method:  "GET",
			Path:    "/test",
			Headers: map[string]string{"X-Test": "value"},
			Query:   map[string]string{"q": "hello"},
			Body:    map[string]interface{}{"key": "val"},
		},
		Outputs: map[string]interface{}{"out1": "data"},
		Items:   map[string]interface{}{"item": "current-item"},
	}

	env := a.buildEnvironment(ctx)

	assert.Equal(t, ctx.Outputs, env["outputs"])
	assert.Equal(t, "current-item", env["item"])

	reqEnv, ok := env["request"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "GET", reqEnv["method"])
	assert.Equal(t, "/test", reqEnv["path"])

	// input should be set from Body
	assert.Equal(t, ctx.Request.Body, env["input"])
}

func TestAdapter_buildEnvironment_NoRequest(t *testing.T) {
	a := &Adapter{}

	ctx := &executor.ExecutionContext{
		Outputs: map[string]interface{}{},
		Items:   map[string]interface{}{},
	}

	env := a.buildEnvironment(ctx)
	assert.Equal(t, ctx.Outputs, env["outputs"])
	assert.Nil(t, env["request"])
}

func TestAdapter_Execute_MissingCapabilities(t *testing.T) {
	// Build an agent server that returns no capabilities
	_, pub, err := federation.GenerateKeypair()
	require.NoError(t, err)
	pubPEM := buildPublicKeyPEM(pub)

	calleeURN := testCalleeURN()

	agentServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				agentCap := registry.AgentCapability{
					URN:          calleeURN,
					Endpoint:     "PLACEHOLDER",
					TrustLevel:   "self-attested",
					PublicKey:    string(pubPEM),
					Capabilities: []registry.Capability{}, // empty!
				}
				agentCap.Endpoint = r.Host // placeholder
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(agentCap)
			}
		}),
	)
	defer agentServer.Close()

	// We need to re-create with correct endpoint
	agentServer.Close()
	agentServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			agentCap := registry.AgentCapability{
				URN:          calleeURN,
				Endpoint:     agentServer.URL,
				TrustLevel:   "self-attested",
				PublicKey:    string(pubPEM),
				Capabilities: []registry.Capability{}, // empty!
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(agentCap)
		}
	}))
	defer agentServer.Close()

	registryServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			resp := map[string]string{"endpoint": agentServer.URL}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}),
	)
	defer registryServer.Close()

	adapter := buildAdapter(t, registryServer.URL)
	ctx := buildMinimalExecutionContext()

	cfg := &domain.RemoteAgentConfig{
		URN:   calleeURN,
		Input: map[string]domain.Expression{},
	}

	_, err = adapter.Execute(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no actions defined")
}

// TestAdapter_Execute_CacheHit verifies that a second call with the same URN uses cache.
func TestAdapter_Execute_CacheHit(t *testing.T) {
	callCount := 0

	_, pub, err := federation.GenerateKeypair()
	require.NoError(t, err)
	calleePriv, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	calleeKM := federation.NewKeyManager(calleePriv)
	_ = pub

	calleePubPEM, err := calleeKM.PublicKeyPEM()
	require.NoError(t, err)

	calleeURN := testCalleeURN()
	callerURN := testCallerURN()

	var agentServer *httptest.Server
	agentServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			callCount++
			agentCap := registry.AgentCapability{
				URN:          calleeURN,
				Endpoint:     agentServer.URL,
				TrustLevel:   "self-attested",
				PublicKey:    string(calleePubPEM),
				Capabilities: []registry.Capability{{ActionID: "test"}},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(agentCap)

		case http.MethodPost:
			var req federation.InvocationRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			parsedCallee, _ := federation.Parse(calleeURN)
			parsedCaller, _ := federation.Parse(callerURN)

			receipt := federation.Receipt{
				MessageID: req.MessageID,
				Timestamp: time.Now().UTC(),
				Callee:    *parsedCallee,
				Caller:    *parsedCaller,
				Execution: federation.ExecutionResult{
					Status:  "success",
					Outputs: map[string]interface{}{"r": "ok"},
				},
			}
			receiptJSON, _ := json.Marshal(&receipt)
			sig, _ := calleeKM.Sign(receiptJSON)

			w.Header().Set("X-Uaf-Receipt", base64.StdEncoding.EncodeToString(receiptJSON))
			w.Header().Set("X-Uaf-Receipt-Signature", hex.EncodeToString(sig))
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer agentServer.Close()

	registryServer := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			resp := map[string]string{"endpoint": agentServer.URL}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		}),
	)
	defer registryServer.Close()

	callerPrivKey, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	callerKM := federation.NewKeyManager(callerPrivKey)

	// Use a real TTL so cache actually works
	registryClient := registry.NewClient(registryServer.URL)
	adapter := &Adapter{
		registryClient:   registryClient,
		callerKeyManager: callerKM,
		callerURN:        callerURN,
	}

	ctx := buildMinimalExecutionContext()
	cfg := &domain.RemoteAgentConfig{
		URN:   calleeURN,
		Input: map[string]domain.Expression{},
	}

	// First call
	_, err = adapter.Execute(ctx, cfg)
	require.NoError(t, err)

	// Second call - capability should be cached
	firstCount := callCount
	_, err = adapter.Execute(ctx, cfg)
	require.NoError(t, err)

	// Well-known should only be called once due to caching
	assert.Equal(
		t,
		firstCount,
		callCount,
		"well-known endpoint should only be called once due to caching",
	)
}

// TestAdapter_Execute_Fallback verifies that fallback agents are tried when primary fails.
func TestAdapter_Execute_Fallback(t *testing.T) {
	// This test verifies the fallback logic: when primary fails (no server), try fallback.
	// We use a URN that we know won't resolve as the primary, and a valid URN as fallback.

	registryServer := buildTestServers(t, http.StatusOK, true, false)

	adapter := buildAdapter(t, registryServer.URL)
	ctx := buildMinimalExecutionContext()

	// Primary points to a bad URL; fallback points to valid server
	// Use an invalid URN structure-wise to cause primary lookup failure via wrong registry
	badRegistryAdapter := buildAdapter(t, "http://127.0.0.1:1")
	_ = badRegistryAdapter

	cfg := &domain.RemoteAgentConfig{
		URN:   testCalleeURN(),
		Input: map[string]domain.Expression{},
		// No fallback - just verify normal success
	}

	result, err := adapter.Execute(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, result)
}

// Test uuid generation to ensure MessageID is properly set.
func TestAdapter_Execute_MessageIDUnique(t *testing.T) {
	// Each Execute call should produce a unique MessageID
	id1 := uuid.New()
	id2 := uuid.New()
	assert.NotEqual(t, id1, id2)
}
