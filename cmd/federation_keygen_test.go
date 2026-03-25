package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFederationKeygen_Success(t *testing.T) {
	tmpDir := t.TempDir()
	privPath := filepath.Join(tmpDir, "test.key")
	pubPath := filepath.Join(tmpDir, "test.key.pub")

	cmd := newFederationKeygenCmd()
	cmd.SetArgs([]string{
		"--org", "test-org",
		"--private", privPath,
		"--public", pubPath,
	})
	err := cmd.Execute()
	require.NoError(t, err)

	assert.FileExists(t, privPath)
	assert.FileExists(t, pubPath)

	privInfo, err := os.Stat(privPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), privInfo.Mode().Perm())

	pubInfo, err := os.Stat(pubPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), pubInfo.Mode().Perm())

	privContent, err := os.ReadFile(privPath)
	require.NoError(t, err)
	assert.Contains(t, string(privContent), "ED25519 PRIVATE KEY")

	pubContent, err := os.ReadFile(pubPath)
	require.NoError(t, err)
	assert.Contains(t, string(pubContent), "ED25519 PUBLIC KEY")
}

func TestFederationKeygen_MissingOrg(t *testing.T) {
	tmpDir := t.TempDir()
	cmd := newFederationKeygenCmd()
	cmd.SetArgs([]string{
		"--private", filepath.Join(tmpDir, "test.key"),
		"--public", filepath.Join(tmpDir, "test.key.pub"),
	})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "organization name is required")
}

func TestFederationKeygen_OverwriteProtection(t *testing.T) {
	tmpDir := t.TempDir()
	privPath := filepath.Join(tmpDir, "test.key")
	pubPath := filepath.Join(tmpDir, "test.key.pub")

	// Create the files first
	err := os.WriteFile(privPath, []byte("existing"), 0600)
	require.NoError(t, err)

	cmd := newFederationKeygenCmd()
	cmd.SetArgs([]string{
		"--org", "test-org",
		"--private", privPath,
		"--public", pubPath,
	})
	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestFederationKeygen_OverwriteFlag(t *testing.T) {
	tmpDir := t.TempDir()
	privPath := filepath.Join(tmpDir, "test.key")
	pubPath := filepath.Join(tmpDir, "test.key.pub")

	// Create the files first
	err := os.WriteFile(privPath, []byte("old"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(pubPath, []byte("old"), 0644)
	require.NoError(t, err)

	cmd := newFederationKeygenCmd()
	cmd.SetArgs([]string{
		"--org", "test-org",
		"--private", privPath,
		"--public", pubPath,
		"--overwrite",
	})
	err = cmd.Execute()
	require.NoError(t, err)

	// Files should be new (not "old")
	privContent, err := os.ReadFile(privPath)
	require.NoError(t, err)
	assert.Contains(t, string(privContent), "ED25519 PRIVATE KEY")
}

func TestFederationKeygen_DefaultPaths(t *testing.T) {
	// Override HOME to avoid polluting the real home directory
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	cmd := newFederationKeygenCmd()
	cmd.SetArgs([]string{"--org", "myorg"})
	err := cmd.Execute()
	require.NoError(t, err)

	expectedPriv := filepath.Join(tmpHome, ".config", "kdeps", "keys", "myorg.key")
	expectedPub := filepath.Join(tmpHome, ".config", "kdeps", "keys", "myorg.key.pub")
	assert.FileExists(t, expectedPriv)
	assert.FileExists(t, expectedPub)
}
