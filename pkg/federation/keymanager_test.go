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

package federation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateKeypair(t *testing.T) {
	private, public, err := GenerateKeypair()
	assert.NoError(t, err)
	assert.NotNil(t, private)
	assert.NotNil(t, public)
	assert.Len(t, private.Seed(), 32) // Ed25519 private key seed is 32 bytes
	assert.Len(t, public, 32)         // Ed25519 public key is 32 bytes
}

func TestKeyManagerSignVerify(t *testing.T) {
	private, public, err := GenerateKeypair()
	assert.NoError(t, err)

	km := NewKeyManager(private)

	msg := []byte("test message")
	sig, err := km.Sign(msg)
	assert.NoError(t, err)
	assert.Len(t, sig, 64) // Ed25519 signature is 64 bytes

	// Verify with correct public key
	valid := Verify(public, msg, sig)
	assert.True(t, valid)

	// Verify with wrong message fails
	invalidMsg := []byte("wrong message")
	valid = Verify(public, invalidMsg, sig)
	assert.False(t, valid)

	// Verify with wrong key fails
	_, otherPublic, _ := GenerateKeypair()
	valid = Verify(otherPublic, msg, sig)
	assert.False(t, valid)
}

func TestKeyManagerSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := tmpDir + "/test.key"

	// Generate and save
	private, _, err := GenerateKeypair()
	assert.NoError(t, err)
	km := NewKeyManager(private)
	km.keyPath = keyPath
	err = km.Save()
	assert.NoError(t, err)
	assert.FileExists(t, keyPath)

	// Load
	km2, err := LoadKey(keyPath)
	assert.NoError(t, err)
	assert.Equal(t, km.publicKey, km2.publicKey)

	// Sign with loaded key and verify
	msg := []byte("hello")
	sig, err := km2.Sign(msg)
	assert.NoError(t, err)
	assert.True(t, Verify(km2.publicKey, msg, sig))
}

func TestKeyManagerLoadOrCreate(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := tmpDir + "/loca.key"

	// First call creates new key
	km1, err := LoadOrCreate(keyPath)
	assert.NoError(t, err)
	assert.FileExists(t, keyPath)

	// Second call loads existing
	km2, err := LoadOrCreate(keyPath)
	assert.NoError(t, err)
	assert.Equal(t, km1.publicKey, km2.publicKey)
}

func TestPublicKeyPEM(t *testing.T) {
	_, public, err := GenerateKeypair()
	assert.NoError(t, err)

	km := &KeyManager{publicKey: public}

	pemData, err := km.PublicKeyPEM()
	assert.NoError(t, err)
	assert.Contains(t, string(pemData), "ED25519 PUBLIC KEY")

	// Also save to file
	tmpDir := t.TempDir()
	pubPath := tmpDir + "/pub.pem"
	err = km.SavePublicKey(pubPath)
	assert.NoError(t, err)
	assert.FileExists(t, pubPath)
}

func TestSignMessage(t *testing.T) {
	private, public, err := GenerateKeypair()
	assert.NoError(t, err)

	km := NewKeyManager(private)
	sm, err := km.SignMessage([]byte("data"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("data"), sm.Data)
	assert.Len(t, sm.Signature, 64)
	assert.Equal(t, public, sm.PublicKey)
	assert.True(t, sm.Verify())
}

func TestKeyManagerPublicKey(t *testing.T) {
	private, public, err := GenerateKeypair()
	assert.NoError(t, err)

	km := NewKeyManager(private)
	assert.Equal(t, public, km.PublicKey())
}

func TestWriteAndReadKeyToFile(t *testing.T) {
	private, _, err := GenerateKeypair()
	assert.NoError(t, err)

	dir := t.TempDir()
	path := filepath.Join(dir, "test.pem")

	// Write.
	assert.NoError(t, WriteKeyToFile(path, private))

	// Read back and compare.
	loaded, err := ReadKeyFromFile(path)
	assert.NoError(t, err)
	assert.Equal(t, private, loaded)
}

func TestReadKeyFromFile_NotFound(t *testing.T) {
	_, err := ReadKeyFromFile("/nonexistent/path/key.pem")
	assert.Error(t, err)
}

func TestReadKeyFromFile_InvalidPEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.pem")
	assert.NoError(t, os.WriteFile(path, []byte("not a pem block"), 0600))

	_, err := ReadKeyFromFile(path)
	assert.Error(t, err)
}

func BenchmarkGenerateKeypair(b *testing.B) {
	for range b.N {
		_, _, _ = GenerateKeypair()
	}
}

func BenchmarkSignVerify(b *testing.B) {
	private, public, _ := GenerateKeypair()
	km := NewKeyManager(private)
	msg := []byte("benchmark message")
	sig, _ := km.Sign(msg)

	b.ResetTimer()
	for range b.N {
		_ = Verify(public, msg, sig)
	}
}
