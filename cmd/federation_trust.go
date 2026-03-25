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

	"github.com/spf13/cobra"
)

const (
	pubKeyExtension = ".pub"
)

// trustCmd represents the `kdeps federation trust` command with subcommands.
func newFederationTrustCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "trust",
		Short: "Manage local trust anchors",
		Long: `Manage the set of trusted registry public keys.

Trust anchors are the root public keys used to verify signatures from registries.
Use 'trust add' to add a registry's public key, 'trust list' to show all, and 'trust remove' to delete.`,
	}

	cmd.AddCommand(newFederationTrustAddCmd())
	cmd.AddCommand(newFederationTrustListCmd())
	cmd.AddCommand(newFederationTrustRemoveCmd())

	return cmd
}

// newFederationTrustAddCmd creates `kdeps federation trust add`.
func newFederationTrustAddCmd() *cobra.Command {
	var (
		registryURL   string
		publicKeyPath string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a trust anchor",
		Long: `Add a registry's public key as a trust anchor.

The public key file will be copied to the local trust store:
  ~/.config/kdeps/trust/<registry-hostname>.pub

Examples:
  kdeps federation trust add --registry https://registry.kdeps.io --public-key ./registry.pub`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if registryURL == "" {
				return errors.New("registry URL is required (use --registry)")
			}
			if publicKeyPath == "" {
				return errors.New("public key path is required (use --public-key)")
			}

			// Read public key file
			pubData, err := os.ReadFile(publicKeyPath)
			if err != nil {
				return fmt.Errorf("failed to read public key: %w", err)
			}

			// Determine trust store location
			trustDir, err := getTrustDir()
			if err != nil {
				return fmt.Errorf("failed to get trust directory: %w", err)
			}
			err = os.MkdirAll(trustDir, 0750)
			if err != nil {
				return fmt.Errorf("failed to create trust directory: %w", err)
			}

			// Derive filename from registry URL host
			host := extractRegistryHost(registryURL)
			if host == "" {
				host = "unknown"
			}
			destPath := filepath.Join(trustDir, host+pubKeyExtension)

			// Write with 0644
			err = os.WriteFile(destPath, pubData, 0644) //nolint:gosec // public key file should be world-readable
			if err != nil {
				return fmt.Errorf("failed to write trust anchor: %w", err)
			}

			fmt.Fprintf(os.Stdout, "Added trust anchor for %s at %s\n", host, destPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&registryURL, "registry", "", "Registry URL (required)")
	cmd.Flags().
		StringVar(&publicKeyPath, "public-key", "", "Path to registry public key PEM (required)")

	return cmd
}

// newFederationTrustListCmd creates `kdeps federation trust list`.
func newFederationTrustListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List trust anchors",
		Long:  "List all trusted registry public keys in the local trust store.",
		RunE: func(_ *cobra.Command, _ []string) error {
			trustDir, err := getTrustDir()
			if err != nil {
				return fmt.Errorf("failed to get trust directory: %w", err)
			}

			if _, statErr := os.Stat(trustDir); os.IsNotExist(statErr) {
				fmt.Fprintln(os.Stdout, "No trust anchors found (trust directory does not exist).")
				return nil
			}

			entries, err := os.ReadDir(trustDir)
			if err != nil {
				return fmt.Errorf("failed to read trust directory: %w", err)
			}

			if len(entries) == 0 {
				fmt.Fprintln(os.Stdout, "No trust anchors configured.")
				return nil
			}

			fmt.Fprintln(os.Stdout, "Trusted registries:")
			for _, e := range entries {
				if !e.IsDir() && filepath.Ext(e.Name()) == pubKeyExtension {
					fmt.Fprintf(os.Stdout, "  - %s\n", e.Name())
				}
			}
			return nil
		},
	}

	return cmd
}

// removeAllTrustAnchors removes all .pub files from the trust directory.
func removeAllTrustAnchors(trustDir string) error {
	entries, err := os.ReadDir(trustDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read trust directory: %w", err)
	}
	removed := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == pubKeyExtension {
			path := filepath.Join(trustDir, e.Name())
			removeErr := os.Remove(path)
			if removeErr != nil {
				fmt.Fprintf(os.Stdout, "Failed to remove %s: %v\n", e.Name(), removeErr)
			} else {
				removed++
			}
		}
	}
	fmt.Fprintf(os.Stdout, "Removed %d trust anchor(s).\n", removed)
	return nil
}

// newFederationTrustRemoveCmd creates `kdeps federation trust remove`.
func newFederationTrustRemoveCmd() *cobra.Command {
	var (
		registryHost string
		all          bool
	)

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a trust anchor",
		Long: `Remove a trust anchor by registry hostname.

Examples:
  kdeps federation trust remove --host registry.kdeps.io
  kdeps federation trust remove --all  # remove all trust anchors`,
		RunE: func(_ *cobra.Command, _ []string) error {
			trustDir, err := getTrustDir()
			if err != nil {
				return fmt.Errorf("failed to get trust directory: %w", err)
			}

			if all {
				return removeAllTrustAnchors(trustDir)
			}

			if registryHost == "" {
				return errors.New(
					"registry host is required (use --host) or use --all to remove all",
				)
			}

			destPath := filepath.Join(trustDir, registryHost+pubKeyExtension)
			err = os.Remove(destPath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("no trust anchor found for %s", registryHost)
				}
				return fmt.Errorf("failed to remove trust anchor: %w", err)
			}

			fmt.Fprintf(os.Stdout, "Removed trust anchor for %s\n", registryHost)
			return nil
		},
	}

	cmd.Flags().StringVar(&registryHost, "host", "", "Registry hostname (e.g., registry.kdeps.io)")
	cmd.Flags().BoolVar(&all, "all", false, "Remove all trust anchors")

	return cmd
}

// getTrustDir returns the trust store directory: ~/.config/kdeps/trust.
func getTrustDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".config", "kdeps", "trust"), nil
}

// extractRegistryHost extracts the hostname from a registry URL.
func extractRegistryHost(urlStr string) string {
	// Very simple: strip scheme and path, keep host:port if present
	// For MVP, use regex or strings. Could use net/url but avoid import for simplicity
	// Strip common schemes
	s := urlStr
	if len(s) > 7 && s[:7] == "http://" {
		s = s[7:]
	} else if len(s) > 8 && s[:8] == "https://" {
		s = s[8:]
	}
	// Remove trailing slash and path
	if idx := filepath.Dir(s); idx != "." && idx != "/" {
		// Not perfect; better to find first slash
		for i, ch := range s {
			if ch == '/' || ch == ':' {
				// Actually we might want to keep port, so split on '/' only.
				if ch == '/' {
					s = s[:i]
					break
				}
			}
		}
	}
	// Remove path if any slash remains
	for i, ch := range s {
		if ch == '/' {
			s = s[:i]
			break
		}
	}
	return s
}
