package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/federation"
)

const registerTestHash = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"

func buildSpecYAML(t *testing.T) (string, []byte, string) {
	t.Helper()
	specContent := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-agent
  version: "1.0.0"
`
	c := &federation.Canonicalizer{}
	hashBytes, err := c.SHA256([]byte(specContent))
	require.NoError(t, err)
	hashHex := c.HashHex(hashBytes)

	return specContent, hashBytes, hashHex
}

func TestFederationRegister_Success(t *testing.T) {
	// Start mock registry server
	registered := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/agents" && r.Method == http.MethodPost {
			registered = true
			io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).
				Encode(map[string]string{"messageId": "test-id", "status": "registered"})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")
	pubPath := keyPath + ".pub"

	// Generate key
	keygenCmd := newFederationKeygenCmd()
	keygenCmd.SetArgs([]string{"--org", "test", "--private", keyPath, "--public", pubPath})
	require.NoError(t, keygenCmd.Execute())

	// Write spec file
	specContent, _, hashHex := buildSpecYAML(t)
	specPath := filepath.Join(tmpDir, "workflow.yaml")
	err := os.WriteFile(specPath, []byte(specContent), 0644)
	require.NoError(t, err)

	urnStr := "urn:agent:registry.example.com/test:test-agent@v1.0.0#sha256:" + hashHex

	cmd := newFederationRegisterCmd()
	cmd.SetArgs([]string{
		"--urn", urnStr,
		"--spec", specPath,
		"--registry", server.URL,
		"--contact", "test@example.com",
		"--key", keyPath,
	})
	err = cmd.Execute()
	require.NoError(t, err)
	assert.True(t, registered)
}

func TestFederationRegister_MissingURN(t *testing.T) {
	cmd := newFederationRegisterCmd()
	cmd.SetArgs([]string{
		"--spec", "/tmp/spec.yaml",
		"--registry", "http://localhost",
		"--contact", "test@example.com",
	})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "URN is required")
}

func TestFederationRegister_MissingSpec(t *testing.T) {
	cmd := newFederationRegisterCmd()
	cmd.SetArgs([]string{
		"--urn", "urn:agent:x/y:z@v1#sha256:" + registerTestHash,
		"--registry", "http://localhost",
		"--contact", "test@example.com",
	})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "specification path is required")
}

func TestFederationRegister_MissingRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "spec.yaml")
	os.WriteFile(specPath, []byte("spec"), 0644)

	cmd := newFederationRegisterCmd()
	cmd.SetArgs([]string{
		"--urn", "urn:agent:x/y:z@v1#sha256:" + registerTestHash,
		"--spec", specPath,
		"--contact", "test@example.com",
	})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry URL is required")
}

func TestFederationRegister_HashMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test.key")
	pubPath := keyPath + ".pub"
	keygenCmd := newFederationKeygenCmd()
	keygenCmd.SetArgs([]string{"--org", "test", "--private", keyPath, "--public", pubPath})
	require.NoError(t, keygenCmd.Execute())

	specPath := filepath.Join(tmpDir, "spec.yaml")
	os.WriteFile(specPath, []byte("different content"), 0644)

	// URN has hash of "abc..." but spec content will have different hash
	urnStr := "urn:agent:reg.example.com/test:agent@v1.0.0#sha256:" + registerTestHash

	cmd := newFederationRegisterCmd()
	cmd.SetArgs([]string{
		"--urn", urnStr,
		"--spec", specPath,
		"--registry", server.URL,
		"--contact", "test@example.com",
		"--key", keyPath,
	})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hash does not match URN")
}

func TestFederationRegister_InvalidURN(t *testing.T) {
	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "spec.yaml")
	os.WriteFile(specPath, []byte("spec"), 0644)

	cmd := newFederationRegisterCmd()
	cmd.SetArgs([]string{
		"--urn", "not-a-valid-urn",
		"--spec", specPath,
		"--registry", "http://localhost",
		"--contact", "test@example.com",
	})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid URN")
}
