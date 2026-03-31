package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFederationKeyRotate_Success(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "myorg.key")
	pubPath := keyPath + ".pub"

	// Create initial keypair via keygen
	keygenCmd := newFederationKeygenCmd()
	keygenCmd.SetArgs([]string{
		"--org", "myorg",
		"--private", keyPath,
		"--public", pubPath,
	})
	require.NoError(t, keygenCmd.Execute())

	// Read original public key
	origPubContent, err := os.ReadFile(pubPath)
	require.NoError(t, err)

	// Rotate keys
	rotateCmd := newFederationKeyRotateCmd()
	rotateCmd.SetArgs([]string{
		"--key", keyPath,
		"--org", "myorg",
	})
	err = rotateCmd.Execute()
	require.NoError(t, err)

	// Verify new key files exist
	assert.FileExists(t, keyPath)
	assert.FileExists(t, pubPath)

	// Verify new public key is different from original
	newPubContent, err := os.ReadFile(pubPath)
	require.NoError(t, err)
	assert.NotEqual(t, string(origPubContent), string(newPubContent))

	// New key should still be valid PEM
	assert.Contains(t, string(newPubContent), "ED25519 PUBLIC KEY")
}

func TestFederationKeyRotate_MissingKeyFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "nonexistent.key")

	rotateCmd := newFederationKeyRotateCmd()
	rotateCmd.SetArgs([]string{"--key", keyPath})
	err := rotateCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no existing key found at")
}

func TestFederationKeyRotate_MissingArgs(t *testing.T) {
	rotateCmd := newFederationKeyRotateCmd()
	rotateCmd.SetArgs([]string{})
	err := rotateCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "organization name or key path is required")
}

func TestFederationKeyRotate_WithBackup(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "myorg.key")
	pubPath := keyPath + ".pub"

	// Create initial keypair via keygen
	keygenCmd := newFederationKeygenCmd()
	keygenCmd.SetArgs([]string{
		"--org", "myorg",
		"--private", keyPath,
		"--public", pubPath,
	})
	require.NoError(t, keygenCmd.Execute())

	// Rotate with backup
	rotateCmd := newFederationKeyRotateCmd()
	rotateCmd.SetArgs([]string{
		"--key", keyPath,
		"--backup",
	})
	err := rotateCmd.Execute()
	require.NoError(t, err)

	// Verify original key file still exists (new key) and backup was created
	assert.FileExists(t, keyPath)

	// At least one backup file should exist
	entries, err := os.ReadDir(tmpDir)
	require.NoError(t, err)
	backupFound := false
	for _, e := range entries {
		if len(e.Name()) > len("myorg.key.backup") &&
			e.Name()[:len("myorg.key.backup")] == "myorg.key.backup" {
			backupFound = true
			break
		}
	}
	assert.True(t, backupFound, "backup file should have been created")
}

func TestFederationKeyRotate_DefaultKeyDir(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// First generate a key at the default location
	keygenCmd := newFederationKeygenCmd()
	keygenCmd.SetArgs([]string{"--org", "testorg"})
	require.NoError(t, keygenCmd.Execute())

	// Now rotate using --org
	rotateCmd := newFederationKeyRotateCmd()
	rotateCmd.SetArgs([]string{"--org", "testorg"})
	err := rotateCmd.Execute()
	require.NoError(t, err)

	keyDir := filepath.Join(tmpHome, ".config", "kdeps", "keys")
	assert.FileExists(t, filepath.Join(keyDir, "testorg.key"))
	assert.FileExists(t, filepath.Join(keyDir, "testorg.key.pub"))
}
