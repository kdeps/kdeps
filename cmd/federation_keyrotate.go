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
	"time"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/federation"
)

// newFederationKeyRotateCmd creates the `kdeps federation key-rotate` command.
func newFederationKeyRotateCmd() *cobra.Command {
	var (
		orgName        string
		privateKeyPath string
		backup         bool
	)

	cmd := &cobra.Command{
		Use:   "key-rotate",
		Short: "Rotate Ed25519 keys (dual-key period)",
		Long: `Rotate the federation keys for an organization.

This command generates a new keypair and optionally backs up the old one.
During a rotation period, both old and new public keys may be accepted
by the registry to ensure continuity.

Examples:
  # Rotate keys for organization "my-org" (uses default key locations)
  kdeps federation key-rotate --org my-org

  # Rotate with explicit private key path and backup old key
  kdeps federation key-rotate --key /path/to/my-org.key --backup`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runFederationKeyRotate(orgName, privateKeyPath, backup)
		},
	}

	cmd.Flags().StringVar(&orgName, "org", "", "Organization name (used to locate default key)")
	cmd.Flags().StringVar(&privateKeyPath, "key", "", "Path to private key to rotate")
	cmd.Flags().BoolVar(&backup, "backup", true, "Backup old key with timestamp")

	return cmd
}

// runFederationKeyRotate executes the key rotation logic.
func runFederationKeyRotate(orgName, privateKeyPath string, backup bool) error {
	// Validate input
	if orgName == "" && privateKeyPath == "" {
		return errors.New("organization name or key path is required (use --org or --key)")
	}
	// Determine the key path
	keyPath, err := determineKeyPath(orgName, privateKeyPath)
	if err != nil {
		return err
	}
	// Check that existing key exists and load it
	existingKM, err := federation.LoadKey(keyPath)
	if err != nil {
		if os.IsNotExist(errors.Unwrap(err)) {
			return fmt.Errorf("no existing key found at %s", keyPath)
		}
		return fmt.Errorf("failed to load existing key: %w", err)
	}
	existingPub := existingKM.PublicKey()
	// Backup old key if requested
	if backup {
		if err := backupOldKey(keyPath); err != nil { //nolint:govet // intentional shadow
			return err
		}
	}
	// Generate new keypair and write to disk
	if err := rotateKeyPair(keyPath); err != nil { //nolint:govet // intentional shadow
		return err
	}
	// Output success
	printRotationSuccess(orgName, keyPath, existingPub)
	return nil
}

// determineKeyPath resolves the private key path from flags.
func determineKeyPath(orgName, privateKeyPath string) (string, error) {
	if privateKeyPath != "" {
		return privateKeyPath, nil
	}
	keyDir, err := getDefaultKeyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(keyDir, fmt.Sprintf("%s.key", orgName)), nil
}

// backupOldKey renames the existing key file with a timestamp suffix.
func backupOldKey(keyPath string) error {
	backupPath := fmt.Sprintf("%s.backup-%s", keyPath, time.Now().Format("20060102-150405"))
	if err := os.Rename(keyPath, backupPath); err != nil {
		return fmt.Errorf("failed to backup old key: %w", err)
	}
	// Also backup public key if exists
	pubPath := keyPath + ".pub"
	if _, err := os.Stat(pubPath); err == nil {
		pubBackupPath := fmt.Sprintf("%s.backup-%s", pubPath, time.Now().Format("20060102-150405"))
		if err := os.Rename(pubPath, pubBackupPath); err != nil { //nolint:govet // intentional shadow
			// Log but not fatal
			fmt.Fprintf(os.Stderr, "warning: failed to backup public key: %v\n", err)
		} else {
			fmt.Fprintf(os.Stdout, "Backed up old keys to: %s and %s\n", backupPath, pubBackupPath)
		}
	} else {
		fmt.Fprintf(os.Stdout, "Backed up old key to: %s\n", backupPath)
	}
	return nil
}

// rotateKeyPair generates a new Ed25519 keypair and writes private and public keys.
func rotateKeyPair(keyPath string) error {
	// Generate new keypair
	privKey, _, err := federation.GenerateKeypair()
	if err != nil {
		return fmt.Errorf("failed to generate new keypair: %w", err)
	}
	// Write new private key
	if err := federation.WriteKeyToFile(keyPath, privKey); err != nil { //nolint:govet // intentional shadow
		return fmt.Errorf("failed to write new private key: %w", err)
	}
	// Write new public key
	pubPath := keyPath + ".pub"
	km := federation.NewKeyManager(privKey)
	if err := km.SavePublicKey(pubPath); err != nil { //nolint:govet // intentional shadow
		return fmt.Errorf("failed to write new public key: %w", err)
	}
	return nil
}

// printRotationSuccess outputs the rotation result to the user.
func printRotationSuccess(orgName, keyPath string, existingPub []byte) {
	fmt.Fprintf(os.Stdout, "Key rotation completed for %s\n", orgName)
	fmt.Fprintf(os.Stdout, "  New private: %s\n", keyPath)
	fmt.Fprintf(os.Stdout, "  New public:  %s\n", keyPath+".pub")
	fmt.Fprintf(os.Stdout, "  Previous public key fingerprint: %x\n", existingPub[:8])
	fmt.Fprintln(
		os.Stdout,
		"\nNOTE: You should re-register any agents that used the old public key with the registry.",
	)
}
