// Package federation_demo_e2e contains end-to-end tests for the UAF federation demo.
// These tests spin up mock agent-b HTTP servers in-process and verify
// the full UAF request/receipt/signature flow.
package federation_demo_e2e_test

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/federation"
	"github.com/kdeps/kdeps/v2/pkg/federation/registry"
)

// e2eHash is the SHA256 of the canonical form of agent-b/workflow.yaml.
const e2eHash = "58109b25020227c53310847b65400e7cc162a27c8c0559fd1296f85ae247b211"

func agentBURN(authority string) string {
	return "urn:agent:" + authority + "/demo:federation-agent-b@1.0.0#sha256:" + e2eHash
}

func agentAURN() string {
	return "urn:agent:localhost/demo:federation-agent-a@1.0.0#sha256:" + e2eHash
}

// startAgentB creates a mock agent-b server implementing the UAF protocol.
func startAgentB(t *testing.T) (*httptest.Server, *federation.KeyManager) {
	t.Helper()

	privKey, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	calleeKM := federation.NewKeyManager(privKey)

	calleePubPEM, err := calleeKM.PublicKeyPEM()
	require.NoError(t, err)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := listener.Addr().String()
	calleeURNStr := agentBURN(addr)

	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/agent/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		agentCap := registry.AgentCapability{
			URN:          calleeURNStr,
			Title:        "Federation Agent B",
			Version:      "1.0.0",
			Endpoint:     "http://" + addr,
			TrustLevel:   "self-attested",
			PublicKey:    string(calleePubPEM),
			Capabilities: []registry.Capability{{ActionID: "echoResource", Title: "Echo"}},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(agentCap)
	})

	mux.HandleFunc("/.uaf/v1/invoke", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req federation.InvocationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { //nolint:govet // intentional shadow
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		message, _ := req.Payload.Inputs["message"].(string)

		calleeURN, _ := federation.Parse(calleeURNStr)
		callerURN, _ := federation.Parse(req.Caller.URN)

		receipt := federation.Receipt{
			MessageID: req.MessageID,
			Timestamp: time.Now().UTC(),
			Callee:    *calleeURN,
			Caller:    *callerURN,
			Execution: federation.ExecutionResult{
				Status:  "success",
				Outputs: map[string]interface{}{"echoed": message, "agentId": "federation-agent-b"},
			},
		}

		receiptJSON, err := json.Marshal(&receipt) //nolint:govet // intentional shadow
		if err != nil {
			http.Error(w, "marshal error", http.StatusInternalServerError)
			return
		}
		sig, err := calleeKM.Sign(receiptJSON)
		if err != nil {
			http.Error(w, "sign error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("X-Uaf-Receipt", base64.StdEncoding.EncodeToString(receiptJSON))
		w.Header().Set("X-Uaf-Receipt-Signature", hex.EncodeToString(sig))
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewUnstartedServer(mux)
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)

	return server, calleeKM
}

func startRegistry(t *testing.T, agentBURL string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"endpoint": agentBURL})
	}))
	t.Cleanup(server.Close)
	return server
}

func parsePubKeyFromPEM(t *testing.T, pemBytes []byte) []byte {
	t.Helper()
	block, _ := pem.Decode(pemBytes)
	require.NotNil(t, block, "PEM block should not be nil")
	require.Equal(t, "ED25519 PUBLIC KEY", block.Type)
	return block.Bytes
}

// TestE2E_AgentACallsAgentB verifies the full UAF invocation flow.
func TestE2E_AgentACallsAgentB(t *testing.T) {
	agentBServer, _ := startAgentB(t)
	agentBAddr := agentBServer.Listener.Addr().String()
	agentBURL := "http://" + agentBAddr

	registryServer := startRegistry(t, agentBURL)

	// Generate caller (agent-a) keypair
	callerPriv, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	callerKM := federation.NewKeyManager(callerPriv)

	calleeURNStr := agentBURN(agentBAddr)
	callerURNStr := agentAURN()

	// Resolve capability via registry
	regClient := registry.NewClient(registryServer.URL)
	agentCap, err := regClient.ResolveURN(t.Context(), calleeURNStr)
	require.NoError(t, err)
	assert.Equal(t, "self-attested", agentCap.TrustLevel)
	assert.NotEmpty(t, agentCap.PublicKey)

	// Build and sign InvocationRequest
	msgID := uuid.New()
	req := federation.InvocationRequest{
		MessageID: msgID,
		Timestamp: time.Now().UTC(),
		Caller: federation.CallerIdentity{
			URN:       callerURNStr,
			PublicKey: "ed25519:" + hex.EncodeToString(callerKM.PublicKey()),
		},
		Callee: federation.CalleeIdentity{URN: calleeURNStr},
		Payload: federation.RequestPayload{
			Inputs:  map[string]interface{}{"message": "Hello from E2E!"},
			Context: federation.InvocationContext{RequestID: uuid.New().String()},
		},
	}

	reqBody, err := json.Marshal(req)
	require.NoError(t, err)
	reqSig, err := callerKM.Sign(reqBody)
	require.NoError(t, err)

	// Send to agent-b
	httpReq, err := http.NewRequest(http.MethodPost, agentBURL+"/.uaf/v1/invoke", bytes.NewReader(reqBody))
	require.NoError(t, err)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Uaf-Version", "1.0")
	httpReq.Header.Set("X-Uaf-Message-Id", msgID.String())
	httpReq.Header.Set("X-Uaf-Caller-Urn", callerURNStr)
	httpReq.Header.Set("X-Uaf-Signature", hex.EncodeToString(reqSig))

	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(httpReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Decode and verify receipt
	receiptB64 := resp.Header.Get("X-Uaf-Receipt")
	sigHex := resp.Header.Get("X-Uaf-Receipt-Signature")
	require.NotEmpty(t, receiptB64)
	require.NotEmpty(t, sigHex)

	receiptJSON, err := base64.StdEncoding.DecodeString(receiptB64)
	require.NoError(t, err)
	receiptSig, err := hex.DecodeString(sigHex)
	require.NoError(t, err)

	// Verify signature with agent-b's public key
	calleePubBytes := parsePubKeyFromPEM(t, []byte(agentCap.PublicKey))
	assert.True(t, federation.Verify(calleePubBytes, receiptJSON, receiptSig))

	// Parse receipt and validate fields
	var rec federation.Receipt
	require.NoError(t, json.Unmarshal(receiptJSON, &rec))
	assert.Equal(t, msgID, rec.MessageID)
	assert.Equal(t, "success", rec.Execution.Status)
	assert.Equal(t, "Hello from E2E!", rec.Execution.Outputs["echoed"])
}

// TestE2E_TamperingDetected verifies that tampering with a receipt invalidates the signature.
func TestE2E_TamperingDetected(t *testing.T) {
	privKey, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	km := federation.NewKeyManager(privKey)

	calleeURN, _ := federation.Parse(agentBURN("localhost:16396"))
	callerURN, _ := federation.Parse(agentAURN())

	receipt := federation.Receipt{
		MessageID: uuid.New(),
		Timestamp: time.Now().UTC(),
		Callee:    *calleeURN,
		Caller:    *callerURN,
		Execution: federation.ExecutionResult{
			Status:  "success",
			Outputs: map[string]interface{}{"echoed": "hello"},
		},
	}

	receiptJSON, err := json.Marshal(&receipt) //nolint:govet // intentional shadow
	require.NoError(t, err)
	sig, err := km.Sign(receiptJSON)
	require.NoError(t, err)

	// Valid signature
	assert.True(t, federation.Verify(km.PublicKey(), receiptJSON, sig))

	// Tampered receipt
	tampered := bytes.ReplaceAll(receiptJSON, []byte("hello"), []byte("hacked"))
	assert.False(t, federation.Verify(km.PublicKey(), tampered, sig))

	// Wrong key
	wrongPriv, _, _ := federation.GenerateKeypair()
	wrongKM := federation.NewKeyManager(wrongPriv)
	assert.False(t, federation.Verify(wrongKM.PublicKey(), receiptJSON, sig))
}

// TestE2E_URNHashBinding verifies the e2eHash constant matches agent-b/workflow.yaml.
func TestE2E_URNHashBinding(t *testing.T) {
	workflowYAML, err := os.ReadFile("agent-b/workflow.yaml")
	if err != nil {
		t.Skip("workflow.yaml not found; run from examples/federation-demo/")
	}

	c := &federation.Canonicalizer{}
	hashBytes, err := c.SHA256(workflowYAML)
	require.NoError(t, err)
	hashHex := c.HashHex(hashBytes)

	assert.Equal(t, e2eHash, hashHex,
		"e2eHash constant should match SHA256(canonical(agent-b/workflow.yaml))")
}
