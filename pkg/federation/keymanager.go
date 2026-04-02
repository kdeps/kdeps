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
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// KeyManager handles Ed25519 key operations for federation.
type KeyManager struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	keyPath    string
}

// NewKeyManager creates a new KeyManager from an existing private key.
func NewKeyManager(privateKey ed25519.PrivateKey) *KeyManager {
	kdeps_debug.Log("enter: NewKeyManager")
	pub, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		panic("ed25519.PrivateKey.Public() did not return ed25519.PublicKey")
	}
	return &KeyManager{
		privateKey: privateKey,
		publicKey:  pub,
	}
}

// GenerateKeypair generates a new Ed25519 keypair.
func GenerateKeypair() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	kdeps_debug.Log("enter: GenerateKeypair")
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate keypair: %w", err)
	}
	return privateKey, publicKey, nil
}

// LoadOrCreate loads a private key from path, or creates a new one if not exists.
func LoadOrCreate(keyPath string) (*KeyManager, error) {
	kdeps_debug.Log("enter: LoadOrCreate")
	// Try to load existing key
	if _, statErr := os.Stat(keyPath); statErr == nil {
		km, loadErr := LoadKey(keyPath)
		if loadErr != nil {
			return nil, fmt.Errorf("failed to load existing key: %w", loadErr)
		}
		return km, nil
	}

	// Generate new keypair
	privateKey, publicKey, err := GenerateKeypair()
	if err != nil {
		return nil, err
	}

	km := &KeyManager{
		privateKey: privateKey,
		publicKey:  publicKey,
		keyPath:    keyPath,
	}

	// Save to disk
	err = km.Save()
	if err != nil {
		return nil, fmt.Errorf("failed to save new key: %w", err)
	}

	return km, nil
}

// LoadKey loads a private key from a PEM file.
func LoadKey(path string) (*KeyManager, error) {
	kdeps_debug.Log("enter: LoadKey")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "ED25519 PRIVATE KEY" {
		return nil, errors.New("invalid or missing PEM block")
	}

	privateKey := ed25519.PrivateKey(block.Bytes)
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key length")
	}

	pub, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("ed25519.PrivateKey.Public() did not return ed25519.PublicKey")
	}
	km := &KeyManager{
		privateKey: privateKey,
		publicKey:  pub,
		keyPath:    path,
	}

	return km, nil
}

// Save writes the private key to disk in PEM format (0600 permissions).
func (km *KeyManager) Save() error {
	kdeps_debug.Log("enter: Save")
	if km.keyPath == "" {
		return errors.New("no key path configured")
	}

	// Ensure directory exists
	dir := filepath.Dir(km.keyPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Marshal to PEM
	der := []byte(km.privateKey) // full private key bytes (64)
	block := &pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: der,
	}
	pemData := pem.EncodeToMemory(block)

	// Write with secure permissions
	tmpPath := km.keyPath + ".tmp"
	if err := os.WriteFile(tmpPath, pemData, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}
	return os.Rename(tmpPath, km.keyPath)
}

// PublicKey returns the Ed25519 public key (32 bytes).
func (km *KeyManager) PublicKey() ed25519.PublicKey {
	kdeps_debug.Log("enter: PublicKey")
	return km.publicKey
}

// PublicKeyPEM returns the public key in PEM format.
func (km *KeyManager) PublicKeyPEM() ([]byte, error) {
	kdeps_debug.Log("enter: PublicKeyPEM")
	block := &pem.Block{
		Type:  "ED25519 PUBLIC KEY",
		Bytes: km.publicKey[:],
	}
	return pem.EncodeToMemory(block), nil
}

// Sign signs the data with the private key.
func (km *KeyManager) Sign(data []byte) ([]byte, error) {
	kdeps_debug.Log("enter: Sign")
	signature := ed25519.Sign(km.privateKey, data)
	return signature, nil
}

// Verify verifies a signature against the public key.
func Verify(publicKey ed25519.PublicKey, data, signature []byte) bool {
	kdeps_debug.Log("enter: Verify")
	return ed25519.Verify(publicKey, data, signature)
}

// SavePublicKey writes the public key to a separate file (0644 permissions).
func (km *KeyManager) SavePublicKey(pubKeyPath string) error {
	kdeps_debug.Log("enter: SavePublicKey")
	pemData, err := km.PublicKeyPEM()
	if err != nil {
		return fmt.Errorf("failed to encode public key: %w", err)
	}
	if len(pemData) == 0 {
		return errors.New("empty public key data")
	}

	// Ensure directory exists
	dir := filepath.Dir(pubKeyPath)
	err = os.MkdirAll(dir, 0700)
	if err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write with readable permissions
	tmpPath := pubKeyPath + ".tmp"
	err = os.WriteFile(tmpPath, pemData, 0644) //nolint:gosec // public key is intentionally world-readable
	if err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}
	return os.Rename(tmpPath, pubKeyPath)
}

// SignMessage creates a signed message envelope.
// This is a convenience that combines the data and signature.
func (km *KeyManager) SignMessage(data []byte) (*SignedMessage, error) {
	kdeps_debug.Log("enter: SignMessage")
	sig, err := km.Sign(data)
	if err != nil {
		return nil, err
	}
	return &SignedMessage{
		Data:      data,
		Signature: sig,
		PublicKey: km.publicKey,
	}, nil
}

// SignedMessage represents a signed data payload.
type SignedMessage struct {
	Data      []byte
	Signature []byte
	PublicKey ed25519.PublicKey
}

// Verify verifies the signed message.
func (sm *SignedMessage) Verify() bool {
	kdeps_debug.Log("enter: Verify")
	return ed25519.Verify(sm.PublicKey, sm.Data, sm.Signature)
}

// WriteKeyToFile writes the private key to the given path, creating the file
// with secure permissions (0600). Used for CLI commands.
func WriteKeyToFile(path string, privateKey ed25519.PrivateKey) error {
	kdeps_debug.Log("enter: WriteKeyToFile")
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	der := []byte(privateKey) // full 64-byte key (consistent with KeyManager.Save)
	block := &pem.Block{
		Type:  "ED25519 PRIVATE KEY",
		Bytes: der,
	}
	pemData := pem.EncodeToMemory(block)

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, pemData, 0600); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}
	return os.Rename(tmpPath, path)
}

// ReadKeyFromFile reads an Ed25519 private key from a PEM file.
func ReadKeyFromFile(path string) (ed25519.PrivateKey, error) {
	kdeps_debug.Log("enter: ReadKeyFromFile")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "ED25519 PRIVATE KEY" {
		return nil, errors.New("invalid PEM block")
	}
	return ed25519.PrivateKey(block.Bytes), nil
}
