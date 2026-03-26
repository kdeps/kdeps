package cmd

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/federation"
)

const receiptTestHash = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func buildReceiptCalleeURN() string {
	return "urn:agent:callee.example.com/test:receipt-agent@v1.0.0#sha256:" + receiptTestHash
}

func buildReceiptCallerURN() string {
	return "urn:agent:caller.example.com/test:calling-agent@v1.0.0#sha256:" + receiptTestHash
}

func makeReceiptEnvelopeWithKM(
	t *testing.T,
	km *federation.KeyManager,
	calleeURNStr, callerURNStr string,
) string {
	t.Helper()

	calleeURN, err := federation.Parse(calleeURNStr)
	require.NoError(t, err)
	callerURN, err := federation.Parse(callerURNStr)
	require.NoError(t, err)

	rec := federation.Receipt{
		MessageID: uuid.New(),
		Timestamp: time.Now().UTC(),
		Callee:    *calleeURN,
		Caller:    *callerURN,
		Execution: federation.ExecutionResult{
			Status:  "success",
			Outputs: map[string]interface{}{"result": "test-output"},
		},
	}

	receiptJSON, err := json.Marshal(&rec)
	require.NoError(t, err)

	sig, err := km.Sign(receiptJSON)
	require.NoError(t, err)

	envelope := struct {
		ReceiptB64 string `json:"receipt"`
		Signature  string `json:"signature"`
	}{
		ReceiptB64: base64.StdEncoding.EncodeToString(receiptJSON),
		Signature:  hex.EncodeToString(sig),
	}

	envelopeJSON, err := json.Marshal(envelope)
	require.NoError(t, err)
	return string(envelopeJSON)
}

func writePubKeyFile(t *testing.T, km *federation.KeyManager, dir string) string {
	t.Helper()
	pubPath := filepath.Join(dir, "callee.pub")
	err := km.SavePublicKey(pubPath)
	require.NoError(t, err)
	return pubPath
}

func TestFederationReceiptVerify_Success(t *testing.T) {
	tmpDir := t.TempDir()

	privKey, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	km := federation.NewKeyManager(privKey)

	calleeURN := buildReceiptCalleeURN()
	callerURN := buildReceiptCallerURN()

	envelopeJSON := makeReceiptEnvelopeWithKM(t, km, calleeURN, callerURN)

	receiptFile := filepath.Join(tmpDir, "receipt.json")
	err = os.WriteFile(receiptFile, []byte(envelopeJSON), 0644)
	require.NoError(t, err)

	pubKeyFile := writePubKeyFile(t, km, tmpDir)

	cmd := newFederationReceiptVerifyCmd()
	cmd.SetArgs([]string{
		"--receipt", receiptFile,
		"--callee-urn", calleeURN,
		"--caller-urn", callerURN,
		"--public-key", pubKeyFile,
	})
	err = cmd.Execute()
	require.NoError(t, err)
}

func TestFederationReceiptVerify_BadSignature(t *testing.T) {
	tmpDir := t.TempDir()

	// Create receipt signed with one key
	privKey, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	km := federation.NewKeyManager(privKey)

	calleeURN := buildReceiptCalleeURN()
	callerURN := buildReceiptCallerURN()
	envelopeJSON := makeReceiptEnvelopeWithKM(t, km, calleeURN, callerURN)

	receiptFile := filepath.Join(tmpDir, "receipt.json")
	err = os.WriteFile(receiptFile, []byte(envelopeJSON), 0644)
	require.NoError(t, err)

	// Verify with DIFFERENT key
	wrongPriv, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	wrongKM := federation.NewKeyManager(wrongPriv)
	wrongPubFile := writePubKeyFile(t, wrongKM, tmpDir)

	cmd := newFederationReceiptVerifyCmd()
	cmd.SetArgs([]string{
		"--receipt", receiptFile,
		"--callee-urn", calleeURN,
		"--caller-urn", callerURN,
		"--public-key", wrongPubFile,
	})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")
}

func TestFederationReceiptVerify_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	receiptFile := filepath.Join(tmpDir, "bad.json")
	err := os.WriteFile(receiptFile, []byte("not valid json {{{"), 0644)
	require.NoError(t, err)

	privKey, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	km := federation.NewKeyManager(privKey)
	pubKeyFile := writePubKeyFile(t, km, tmpDir)

	cmd := newFederationReceiptVerifyCmd()
	cmd.SetArgs([]string{
		"--receipt", receiptFile,
		"--callee-urn", buildReceiptCalleeURN(),
		"--caller-urn", buildReceiptCallerURN(),
		"--public-key", pubKeyFile,
	})
	err = cmd.Execute()
	require.Error(t, err)
}

func TestFederationReceiptVerify_MissingReceipt(t *testing.T) {
	tmpDir := t.TempDir()

	privKey, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	km := federation.NewKeyManager(privKey)
	pubKeyFile := writePubKeyFile(t, km, tmpDir)

	cmd := newFederationReceiptVerifyCmd()
	cmd.SetArgs([]string{
		"--callee-urn", buildReceiptCalleeURN(),
		"--caller-urn", buildReceiptCallerURN(),
		"--public-key", pubKeyFile,
	})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "receipt file is required")
}

func TestFederationReceiptVerify_MissingCalleeURN(t *testing.T) {
	tmpDir := t.TempDir()
	receiptFile := filepath.Join(tmpDir, "r.json")
	_ = os.WriteFile(receiptFile, []byte("{}"), 0644)

	cmd := newFederationReceiptVerifyCmd()
	cmd.SetArgs([]string{
		"--receipt", receiptFile,
		"--caller-urn", buildReceiptCallerURN(),
		"--public-key", filepath.Join(tmpDir, "pub.pem"),
	})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "callee URN is required")
}

func TestFederationReceiptVerify_URNMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	privKey, _, err := federation.GenerateKeypair()
	require.NoError(t, err)
	km := federation.NewKeyManager(privKey)

	calleeURN := buildReceiptCalleeURN()
	callerURN := buildReceiptCallerURN()
	envelopeJSON := makeReceiptEnvelopeWithKM(t, km, calleeURN, callerURN)

	receiptFile := filepath.Join(tmpDir, "receipt.json")
	err = os.WriteFile(receiptFile, []byte(envelopeJSON), 0644)
	require.NoError(t, err)

	pubKeyFile := writePubKeyFile(t, km, tmpDir)

	// Use different callee URN than what's in receipt
	differentHash := "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
	differentCalleeURN := "urn:agent:different.example.com/test:agent@v1.0.0#sha256:" + differentHash

	cmd := newFederationReceiptVerifyCmd()
	cmd.SetArgs([]string{
		"--receipt", receiptFile,
		"--callee-urn", differentCalleeURN,
		"--caller-urn", callerURN,
		"--public-key", pubKeyFile,
	})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "callee URN mismatch")
}

func TestParseEd25519PublicKey_Valid(t *testing.T) {
	_, pub, err := federation.GenerateKeypair()
	require.NoError(t, err)

	block := &pem.Block{
		Type:  "ED25519 PUBLIC KEY",
		Bytes: pub,
	}
	pemData := pem.EncodeToMemory(block)

	key, err := parseEd25519PublicKey(pemData)
	require.NoError(t, err)
	assert.Len(t, key, 32)
}

func TestParseEd25519PublicKey_InvalidPEM(t *testing.T) {
	_, err := parseEd25519PublicKey([]byte("not-a-pem"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PEM block")
}

func TestParseEd25519PublicKey_WrongType(t *testing.T) {
	block := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: make([]byte, 32),
	}
	pemData := pem.EncodeToMemory(block)
	_, err := parseEd25519PublicKey(pemData)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PEM block")
}
