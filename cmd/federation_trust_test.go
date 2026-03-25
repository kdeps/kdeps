package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPubKeyPEM = `-----BEGIN ED25519 PUBLIC KEY-----
MCowBQYDK2VwAyEAtest12345678901234567890123456789012345678901234
-----END ED25519 PUBLIC KEY-----
`

func writeTempPubKey(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.pub")
	require.NoError(t, err)
	_, err = f.WriteString(testPubKeyPEM)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestFederationTrustAdd_Success(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	pubKeyPath := writeTempPubKey(t)

	cmd := newFederationTrustAddCmd()
	cmd.SetArgs([]string{
		"--registry", "https://registry.example.com",
		"--public-key", pubKeyPath,
	})
	err := cmd.Execute()
	require.NoError(t, err)

	trustDir := filepath.Join(tmpHome, ".config", "kdeps", "trust")
	entries, err := os.ReadDir(trustDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "registry.example.com.pub", entries[0].Name())
}

func TestFederationTrustAdd_MissingRegistry(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	pubKeyPath := writeTempPubKey(t)

	cmd := newFederationTrustAddCmd()
	cmd.SetArgs([]string{"--public-key", pubKeyPath})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registry URL is required")
}

func TestFederationTrustAdd_MissingPublicKey(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cmd := newFederationTrustAddCmd()
	cmd.SetArgs([]string{"--registry", "https://registry.example.com"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "public key path is required")
}

func TestFederationTrustList_Empty(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Do not create trust dir - it should say no anchors
	cmd := newFederationTrustListCmd()
	cmd.SetArgs([]string{})

	var output string
	cmd.SetOut(nil)
	err := cmd.Execute()
	require.NoError(t, err)
	_ = output
}

func TestFederationTrustList_WithAnchors(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Add an anchor first
	pubKeyPath := writeTempPubKey(t)
	addCmd := newFederationTrustAddCmd()
	addCmd.SetArgs([]string{
		"--registry", "https://registry.example.com",
		"--public-key", pubKeyPath,
	})
	require.NoError(t, addCmd.Execute())

	// Now list
	listCmd := newFederationTrustListCmd()
	listCmd.SetArgs([]string{})
	err := listCmd.Execute()
	require.NoError(t, err)

	// Verify the file exists in the trust dir
	trustDir := filepath.Join(tmpHome, ".config", "kdeps", "trust")
	assert.FileExists(t, filepath.Join(trustDir, "registry.example.com.pub"))
}

func TestFederationTrustRemove_Success(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Add an anchor
	pubKeyPath := writeTempPubKey(t)
	addCmd := newFederationTrustAddCmd()
	addCmd.SetArgs([]string{
		"--registry", "https://myregistry.io",
		"--public-key", pubKeyPath,
	})
	require.NoError(t, addCmd.Execute())

	trustDir := filepath.Join(tmpHome, ".config", "kdeps", "trust")
	assert.FileExists(t, filepath.Join(trustDir, "myregistry.io.pub"))

	// Remove it
	removeCmd := newFederationTrustRemoveCmd()
	removeCmd.SetArgs([]string{"--host", "myregistry.io"})
	err := removeCmd.Execute()
	require.NoError(t, err)

	assert.NoFileExists(t, filepath.Join(trustDir, "myregistry.io.pub"))
}

func TestFederationTrustRemove_NotFound(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create trust dir but no entries
	trustDir := filepath.Join(tmpHome, ".config", "kdeps", "trust")
	require.NoError(t, os.MkdirAll(trustDir, 0755))

	removeCmd := newFederationTrustRemoveCmd()
	removeCmd.SetArgs([]string{"--host", "nonexistent.io"})
	err := removeCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no trust anchor found")
}

func TestFederationTrustRemove_All(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Add multiple anchors
	for _, host := range []string{"https://reg1.io", "https://reg2.io", "https://reg3.io"} {
		pubKeyPath := writeTempPubKey(t)
		addCmd := newFederationTrustAddCmd()
		addCmd.SetArgs([]string{"--registry", host, "--public-key", pubKeyPath})
		require.NoError(t, addCmd.Execute())
	}

	trustDir := filepath.Join(tmpHome, ".config", "kdeps", "trust")
	entries, _ := os.ReadDir(trustDir)
	assert.Len(t, entries, 3)

	removeCmd := newFederationTrustRemoveCmd()
	removeCmd.SetArgs([]string{"--all"})
	err := removeCmd.Execute()
	require.NoError(t, err)

	entries, _ = os.ReadDir(trustDir)
	pubEntries := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".pub" {
			pubEntries++
		}
	}
	assert.Equal(t, 0, pubEntries)
}

func TestExtractRegistryHost(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://registry.example.com", "registry.example.com"},
		{"https://registry.example.com/path", "registry.example.com"},
		{"http://localhost:8080", "localhost:8080"},
		{"https://api.kdeps.io/v1", "api.kdeps.io"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := extractRegistryHost(tt.url)
			assert.Equal(t, tt.expected, got)
		})
	}
}
