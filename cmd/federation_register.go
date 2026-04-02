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

package cmd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/federation"
)

// resolvePrivateKey determines which private key manager to use based on flags.
func resolvePrivateKey(
	keyPath, publicKeyPath, urnNamespace string,
) (*federation.KeyManager, error) {
	kdeps_debug.Log("enter: resolvePrivateKey")
	if keyPath != "" {
		return federation.LoadKey(keyPath)
	}
	if publicKeyPath == "" {
		return resolveKeyFromNamespace(urnNamespace)
	}
	return resolveKeyFromPublicKeyPath(publicKeyPath)
}

// resolveKeyFromNamespace loads the default key for the given URN namespace.
func resolveKeyFromNamespace(urnNamespace string) (*federation.KeyManager, error) {
	kdeps_debug.Log("enter: resolveKeyFromNamespace")
	keyDir, err := getDefaultKeyDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get default key directory: %w", err)
	}
	privKeyPath := filepath.Join(keyDir, fmt.Sprintf("%s.key", urnNamespace))
	km, err := federation.LoadKey(privKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key (default %s): %w", privKeyPath, err)
	}
	return km, nil
}

// resolveKeyFromPublicKeyPath derives the private key path from the public key path.
func resolveKeyFromPublicKeyPath(publicKeyPath string) (*federation.KeyManager, error) {
	kdeps_debug.Log("enter: resolveKeyFromPublicKeyPath")
	privKeyPath := publicKeyPath[:len(publicKeyPath)-4] // remove ".pub"
	if _, statErr := os.Stat(privKeyPath); os.IsNotExist(statErr) {
		return nil, fmt.Errorf(
			"private key not found at %s (expected from public key path)",
			privKeyPath,
		)
	}
	km, err := federation.LoadKey(privKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}
	return km, nil
}

// resolvePublicKeyPEM returns the PEM-encoded public key, either from file or derived from km.
func resolvePublicKeyPEM(publicKeyPath string, km *federation.KeyManager) ([]byte, error) {
	kdeps_debug.Log("enter: resolvePublicKeyPEM")
	if publicKeyPath != "" {
		pubData, err := os.ReadFile(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key: %w", err)
		}
		return pubData, nil
	}
	pubData, err := km.PublicKeyPEM()
	if err != nil {
		return nil, fmt.Errorf("failed to encode public key: %w", err)
	}
	return pubData, nil
}

// newFederationRegisterCmd creates the `kdeps federation register` command.
func newFederationRegisterCmd() *cobra.Command {
	kdeps_debug.Log("enter: newFederationRegisterCmd")
	var (
		urnStr        string
		specPath      string
		publicKeyPath string
		registryURL   string
		contactEmail  string
		keyPath       string
	)

	cmd := &cobra.Command{
		Use:   "register [flags]",
		Short: "Register this agent in a UAF registry",
		Long: `Register an agent specification in a UAF registry.

This command creates a signed registration request containing:
  - The agent URN (must match the spec content hash)
  - The agent's public key
  - The specification YAML (or a URL to it)
  - Contact information
  - Trust level and validity period

The registration is signed with the private key corresponding to the public key.

Examples:
  # Register an agent using default keys and a spec file
  kdeps federation register --urn urn:agent:example.com/myorg:myagent@v1.0.0#sha256:abcd1234 \\
    --spec ./workflow.yaml --registry https://registry.kdeps.io --contact admin@example.com

  # Register using a specific key pair
  kdeps federation register --urn ... --spec ... --public-key ./myorg.key.pub --key ./myorg.key`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runFederationRegister(
				cmd,
				urnStr,
				specPath,
				publicKeyPath,
				registryURL,
				contactEmail,
				keyPath,
			)
		},
	}

	cmd.Flags().StringVar(&urnStr, "urn", "", "Agent URN (required)")
	cmd.Flags().StringVar(&specPath, "spec", "", "Path to agent specification YAML (required)")
	cmd.Flags().
		StringVar(&publicKeyPath, "public-key", "", "Path to public key PEM (default: derived from URN namespace)")
	cmd.Flags().StringVar(&registryURL, "registry", "", "Registry endpoint URL (required)")
	cmd.Flags().StringVar(&contactEmail, "contact", "", "Contact email for the agent (required)")
	cmd.Flags().
		StringVar(&keyPath, "key", "", "Path to private key (default: derived from public-key or URN namespace)")

	return cmd
}

// runFederationRegister executes the register command logic.
func runFederationRegister(
	cmd *cobra.Command,
	urnStr, specPath, publicKeyPath, registryURL, contactEmail, keyPath string,
) error {
	kdeps_debug.Log("enter: runFederationRegister")
	// Validate required flags
	if err := validateRegisterFlags(urnStr, specPath, registryURL, contactEmail); err != nil {
		return err
	}

	// Parse URN
	urn, err := federation.Parse(urnStr)
	if err != nil {
		return fmt.Errorf("invalid URN: %w", err)
	}

	// Compute and verify spec hash
	computedHash, err := computeAndVerifySpecHash(specPath, urn)
	if err != nil {
		return err
	}

	// Determine which key to use
	km, err := resolvePrivateKey(keyPath, publicKeyPath, urn.Namespace)
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}

	// Get public key (either from file or derive from private)
	pubKeyPEM, err := resolvePublicKeyPEM(publicKeyPath, km)
	if err != nil {
		return err
	}

	// Build registration payload
	regJSON, err := buildRegistrationPayload(urn, computedHash, contactEmail, string(pubKeyPEM))
	if err != nil {
		return fmt.Errorf("failed to build registration payload: %w", err)
	}

	// Sign the registration payload and send to registry
	receipt, respBody, err := signAndSendRegistration(cmd, km, regJSON, registryURL)
	if err != nil {
		return err
	}

	// Process response
	return processRegistrationResponse(respBody, receipt)
}

// validateRegisterFlags checks that all required flags are provided.
func validateRegisterFlags(urnStr, specPath, registryURL, contactEmail string) error {
	kdeps_debug.Log("enter: validateRegisterFlags")
	if urnStr == "" {
		return errors.New("URN is required (use --urn)")
	}
	if specPath == "" {
		return errors.New("specification path is required (use --spec)")
	}
	if registryURL == "" {
		return errors.New("registry URL is required (use --registry)")
	}
	if contactEmail == "" {
		return errors.New("contact email is required (use --contact)")
	}
	return nil
}

// computeAndVerifySpecHash reads the spec file, computes its canonical hash,
// and verifies it matches the URN's content hash.
func computeAndVerifySpecHash(specPath string, urn *federation.URN) ([]byte, error) {
	kdeps_debug.Log("enter: computeAndVerifySpecHash")
	specBytes, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}
	canonicalizer := &federation.Canonicalizer{}
	computedHash, err := canonicalizer.ComputeHash(specBytes, urn.HashAlg)
	if err != nil {
		return nil, fmt.Errorf("failed to compute spec hash: %w", err)
	}
	if !bytes.Equal(computedHash, urn.ContentHash) {
		return nil, fmt.Errorf(
			"spec content hash does not match URN: expected %x, got %x",
			urn.ContentHash,
			computedHash,
		)
	}
	return computedHash, nil
}

// registrationPayload represents the registration request payload.
type registrationPayload struct {
	URN          string    `json:"urn"`
	PublicKey    string    `json:"publicKey"`
	ContentHash  string    `json:"contentHash"`
	SpecURL      string    `json:"specUrl,omitempty"`
	Contact      string    `json:"contact"`
	TrustLevel   string    `json:"trustLevel"`
	RegisteredAt time.Time `json:"registeredAt"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty"`
}

// buildRegistrationPayload constructs and marshals the registration JSON.
func buildRegistrationPayload(
	urn *federation.URN,
	computedHash []byte,
	contactEmail, pubKeyPEM string,
) ([]byte, error) {
	kdeps_debug.Log("enter: buildRegistrationPayload")
	reg := registrationPayload{
		URN:          urn.String(),
		PublicKey:    pubKeyPEM,
		ContentHash:  fmt.Sprintf("%s:%x", urn.HashAlg, computedHash),
		Contact:      contactEmail,
		TrustLevel:   "self-attested",
		RegisteredAt: time.Now().UTC(),
	}
	return json.MarshalIndent(reg, "", "  ")
}

// signedRegistration represents the signed envelope.
type signedRegistration struct {
	Registration json.RawMessage `json:"registration"`
	Signature    string          `json:"signature"`
}

// signAndSendRegistration signs the payload and performs the HTTP request.
func signAndSendRegistration(
	cmd *cobra.Command,
	km *federation.KeyManager,
	regJSON []byte,
	registryURL string,
) (registrationReceipt, []byte, error) {
	kdeps_debug.Log("enter: signAndSendRegistration")
	var (
		receipt  registrationReceipt
		respBody []byte
	)
	// Sign the registration payload
	signature, err := km.Sign(regJSON)
	if err != nil {
		return receipt, nil, fmt.Errorf("failed to sign registration: %w", err)
	}

	// Create signed envelope
	signedReg := signedRegistration{
		Registration: regJSON,
		Signature:    "ed25519:" + hex.EncodeToString(signature),
	}
	signedJSON, err := json.Marshal(signedReg)
	if err != nil {
		return receipt, nil, fmt.Errorf("failed to marshal signed registration: %w", err)
	}

	// Build HTTP request
	ctx := cmd.Context()
	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		registryURL+"/v1/agents",
		bytes.NewReader(signedJSON),
	)
	if err != nil {
		return receipt, nil, fmt.Errorf("failed to create registry request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return receipt, nil, fmt.Errorf("registry request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return receipt, nil, fmt.Errorf(
			"registration failed with status %d: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	// Read response body
	respBody, err = io.ReadAll(resp.Body)
	if err != nil {
		return receipt, nil, fmt.Errorf("failed to read registry response: %w", err)
	}

	// Decode receipt if present
	if err := json.Unmarshal(respBody, &receipt); err != nil { //nolint:govet // intentional shadow
		// Non-JSON response, leave receipt empty
		receipt = registrationReceipt{}
	}
	return receipt, respBody, nil
}

// registrationReceipt represents the success response from registry.
type registrationReceipt struct {
	MessageID string `json:"messageId"`
	Status    string `json:"status"`
}

// processRegistrationResponse outputs the result to the user.
func processRegistrationResponse(respBody []byte, receipt registrationReceipt) error {
	kdeps_debug.Log("enter: processRegistrationResponse")
	if receipt.MessageID == "" {
		// Non-JSON response
		fmt.Fprintf(os.Stdout, "Registration accepted. Response: %s\n", string(respBody))
	} else {
		fmt.Fprintln(os.Stdout, "Registration successful!")
		fmt.Fprintf(os.Stdout, "  Message ID: %s\n", receipt.MessageID)
		fmt.Fprintf(os.Stdout, "  Status: %s\n", receipt.Status)
	}
	return nil
}
