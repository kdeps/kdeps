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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/federation"
)

// newFederationKeygenCmd creates the `kdeps federation keygen` command.
func newFederationKeygenCmd() *cobra.Command {
	kdeps_debug.Log("enter: newFederationKeygenCmd")
	var (
		outputPriv string
		outputPub  string
		orgName    string
		overwrite  bool
	)

	cmd := &cobra.Command{
		Use:   "keygen [flags]",
		Short: "Generate an Ed25519 keypair for federation identity",
		Long: `Generate a new Ed25519 keypair for use in federation operations.

The private key is written to a file with 0600 permissions.
The public key is written to a separate file with 0644 permissions.

If no output paths are specified, keys are written to:
  ~/.config/kdeps/keys/<org>.key
  ~/.config/kdeps/keys/<org>.key.pub

Examples:
  # Generate keys for organization "my-org" to default location
  kdeps federation keygen --org my-org

  # Generate keys to specific paths
  kdeps federation keygen --org my-org --private /path/to/priv.key --public /path/to/pub.key

  # Overwrite existing keys
  kdeps federation keygen --org my-org --overwrite`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runFederationKeygen(orgName, outputPriv, outputPub, overwrite)
		},
	}

	// Flags
	cmd.Flags().
		StringVar(&outputPriv, "private", "", "Path for private key file (default: ~/.config/kdeps/keys/<org>.key)")
	cmd.Flags().
		StringVar(&outputPub, "public", "", "Path for public key file (default: ~/.config/kdeps/keys/<org>.key.pub)")
	cmd.Flags().
		StringVar(&orgName, "org", "", "Organization name (used in key filenames and URN namespace) (required)")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite existing key files")

	return cmd
}

// runFederationKeygen executes the key generation logic.
func runFederationKeygen(orgName, outputPriv, outputPub string, overwrite bool) error {
	kdeps_debug.Log("enter: runFederationKeygen")
	// Validate orgName
	if orgName == "" {
		return errors.New("organization name is required (use --org)")
	}
	// Resolve output paths
	privPath, pubPath, err := resolveOutputPaths(orgName, outputPriv, outputPub)
	if err != nil {
		return err
	}
	// Check if files exist and overwrite not allowed
	if err := checkKeyFilesExist(privPath, pubPath, overwrite); err != nil { //nolint:govet // intentional shadow
		return err
	}
	// Generate and write keys
	if err := generateAndWriteKeys(privPath, pubPath); err != nil { //nolint:govet // intentional shadow
		return err
	}
	// Output success message
	fmt.Fprintf(os.Stdout, "Generated Ed25519 keypair for %s:\n", orgName)
	fmt.Fprintf(os.Stdout, "  Private: %s (0600)\n", privPath)
	fmt.Fprintf(os.Stdout, "  Public:  %s (0644)\n", pubPath)
	return nil
}

// resolveOutputPaths determines the private and public key file paths.
func resolveOutputPaths(
	orgName, outputPriv, outputPub string,
) (string, string, error) {
	kdeps_debug.Log("enter: resolveOutputPaths")
	if outputPriv != "" && outputPub != "" {
		return outputPriv, outputPub, nil
	}
	keyDir, err := getDefaultKeyDir()
	if err != nil {
		return "", "", err
	}
	if outputPriv == "" {
		outputPriv = filepath.Join(keyDir, fmt.Sprintf("%s.key", orgName))
	}
	if outputPub == "" {
		outputPub = filepath.Join(keyDir, fmt.Sprintf("%s.key.pub", orgName))
	}
	return outputPriv, outputPub, nil
}

// checkKeyFilesExist verifies that key files do not exist unless overwrite is true.
func checkKeyFilesExist(privPath, pubPath string, overwrite bool) error {
	kdeps_debug.Log("enter: checkKeyFilesExist")
	if overwrite {
		return nil
	}
	if _, err := os.Stat(privPath); err == nil {
		return fmt.Errorf(
			"private key file already exists at %s (use --overwrite to replace)",
			privPath,
		)
	}
	if _, err := os.Stat(pubPath); err == nil {
		return fmt.Errorf(
			"public key file already exists at %s (use --overwrite to replace)",
			pubPath,
		)
	}
	return nil
}

// generateAndWriteKeys creates a new Ed25519 keypair and writes them to files.
func generateAndWriteKeys(privPath, pubPath string) error {
	kdeps_debug.Log("enter: generateAndWriteKeys")
	// Generate keypair
	privKey, _, err := federation.GenerateKeypair()
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}
	// Ensure private key directory exists with 0700
	privDir := filepath.Dir(privPath)
	if err := os.MkdirAll(privDir, 0700); err != nil { //nolint:govet // intentional shadow
		return fmt.Errorf("failed to create key directory: %w", err)
	}
	if err := federation.WriteKeyToFile(privPath, privKey); err != nil { //nolint:govet // intentional shadow
		return fmt.Errorf("failed to write private key: %w", err)
	}
	// Ensure public key directory exists with 0755
	pubDir := filepath.Dir(pubPath)
	if err := os.MkdirAll(pubDir, 0750); err != nil { //nolint:govet // intentional shadow
		return fmt.Errorf("failed to create public key directory: %w", err)
	}
	km := federation.NewKeyManager(privKey)
	if err := km.SavePublicKey(pubPath); err != nil { //nolint:govet // intentional shadow
		return fmt.Errorf("failed to write public key: %w", err)
	}
	return nil
}
