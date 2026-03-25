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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// newFederationCmd creates the federation command group.
func newFederationCmd() *cobra.Command {
	federationCmd := &cobra.Command{
		Use:   "federation",
		Short: "Federation commands for UAF (Universal Agent Federation)",
		Long: `Manage federation identity, registry integration, and cross-agent communication.

The federation commands allow you to:
  • Generate and manage Ed25519 keys for agent identity
  • Register agents in a UAF registry
  • Manage trust anchors for verifying remote agents
  • Publish and discover agents in the mesh
  • Verify signed receipts from remote invocations`,
	}

	// Add subcommands
	federationCmd.AddCommand(newFederationKeygenCmd())
	federationCmd.AddCommand(newFederationRegisterCmd())
	federationCmd.AddCommand(newFederationKeyRotateCmd())
	federationCmd.AddCommand(newFederationTrustCmd())
	federationCmd.AddCommand(newFederationMeshCmd())
	federationCmd.AddCommand(newFederationReceiptCmd())

	return federationCmd
}

// getDefaultKeyDir returns the default keys directory: ~/.config/kdeps/keys.
func getDefaultKeyDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".config", "kdeps", "keys"), nil
}

// getDefaultKeyPaths returns the default private and public key paths for an organization.
func getDefaultKeyPaths(org string) (privPath, pubPath string, err error) {
	keyDir, err := getDefaultKeyDir()
	if err != nil {
		return "", "", err
	}
	privPath = filepath.Join(keyDir, fmt.Sprintf("%s.key", org))
	pubPath = filepath.Join(keyDir, fmt.Sprintf("%s.key.pub", org))
	return privPath, pubPath, nil
}
