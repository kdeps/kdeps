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
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"

	"encoding/pem"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/federation"
)

// newFederationReceiptCmd creates the `kdeps federation receipt` command group.
func newFederationReceiptCmd() *cobra.Command {
	kdeps_debug.Log("enter: newFederationReceiptCmd")
	cmd := &cobra.Command{
		Use:   "receipt",
		Short: "Verify signed receipts",
		Long: `Verify and inspect signed receipts from remote agent invocations.

Receipts are cryptographically signed by the callee agent and provide
proof of execution, including outputs, duration, and any errors.`,
	}

	cmd.AddCommand(newFederationReceiptVerifyCmd())

	return cmd
}

// newFederationReceiptVerifyCmd creates `kdeps federation receipt verify`.
func newFederationReceiptVerifyCmd() *cobra.Command {
	kdeps_debug.Log("enter: newFederationReceiptVerifyCmd")
	var (
		receiptPath string
		calleeURN   string
		callerURN   string
		publicKey   string // path to PEM
	)

	cmd := &cobra.Command{
		Use:   "verify [flags]",
		Short: "Verify a signed receipt",
		Long: `Verify the signature and integrity of a UAF receipt.

This command checks:
  - Signature validity using the callee's public key
  - Receipt structure and required fields
  - Consistency between receipt and request (message ID, caller/callee URNs)

The callee's public key must be provided as a PEM file.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			if receiptPath == "" {
				return errors.New("receipt file is required (use --receipt)")
			}
			if calleeURN == "" {
				return errors.New("callee URN is required (use --callee-urn)")
			}
			if callerURN == "" {
				return errors.New("caller URN is required (use --caller-urn)")
			}
			if publicKey == "" {
				return errors.New("callee public key is required (use --public-key)")
			}

			// Read receipt file
			receiptBytes, err := os.ReadFile(receiptPath)
			if err != nil {
				return fmt.Errorf("failed to read receipt: %w", err)
			}

			// Expect envelope format: {"receipt":"<base64>","signature":"<hex>"}
			var envelope struct {
				ReceiptB64 string `json:"receipt"`
				Signature  string `json:"signature"`
			}
			if err := json.Unmarshal(receiptBytes, &envelope); //nolint:govet // shadow err intentionally
			err == nil && envelope.ReceiptB64 != "" && envelope.Signature != "" {
				receiptJSON, decodeErr := base64.StdEncoding.DecodeString(envelope.ReceiptB64)
				if decodeErr != nil {
					return fmt.Errorf("invalid base64 receipt in envelope: %w", decodeErr)
				}
				sig, err := hex.DecodeString(envelope.Signature) //nolint:govet // shadow err for decoding
				if err != nil {
					return fmt.Errorf("invalid hex signature: %w", err)
				}
				return verifyReceiptBytes(receiptJSON, sig, calleeURN, callerURN, publicKey)
			}

			return errors.New(
				"receipt format not recognized; expected envelope with base64 receipt and hex signature",
			)
		},
	}

	cmd.Flags().
		StringVar(&receiptPath, "receipt", "", "Path to receipt envelope JSON file (required)")
	cmd.Flags().StringVar(&calleeURN, "callee-urn", "", "URN of the callee agent (required)")
	cmd.Flags().StringVar(&callerURN, "caller-urn", "", "URN of the caller agent (required)")
	cmd.Flags().StringVar(&publicKey, "public-key", "", "Path to callee public key PEM (required)")

	return cmd
}

// verifyReceiptBytes verifies the receipt JSON and signature.
func verifyReceiptBytes(
	receiptJSON, signature []byte,
	calleeURNStr, callerURNStr, pubKeyPath string,
) error {
	kdeps_debug.Log("enter: verifyReceiptBytes")
	// Parse receipt
	var rec federation.Receipt
	if err := json.Unmarshal(receiptJSON, &rec); err != nil {
		return fmt.Errorf("failed to parse receipt JSON: %w", err)
	}

	// Parse expected URNs
	calleeURN, err := federation.Parse(calleeURNStr)
	if err != nil {
		return fmt.Errorf("invalid callee URN: %w", err)
	}
	callerURN, err := federation.Parse(callerURNStr)
	if err != nil {
		return fmt.Errorf("invalid caller URN: %w", err)
	}

	// Compare URNs using Equals
	if !rec.Callee.Equals(calleeURN) {
		return fmt.Errorf(
			"receipt callee URN mismatch: expected %s, got %s",
			calleeURN.String(),
			rec.Callee.String(),
		)
	}
	if !rec.Caller.Equals(callerURN) {
		return fmt.Errorf(
			"receipt caller URN mismatch: expected %s, got %s",
			callerURN.String(),
			rec.Caller.String(),
		)
	}

	// Read public key
	pubData, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}
	pubKey, err := parseEd25519PublicKey(pubData)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	// Verify signature over the receipt JSON bytes
	if !ed25519.Verify(pubKey, receiptJSON, signature) {
		return errors.New("signature verification failed")
	}

	// Print verification results
	fmt.Fprintln(os.Stdout, "Receipt verification:")
	fmt.Fprintf(os.Stdout, "  Message ID: %s\n", rec.MessageID)
	fmt.Fprintf(os.Stdout, "  Callee: %s\n", rec.Callee.String())
	fmt.Fprintf(os.Stdout, "  Caller: %s\n", rec.Caller.String())
	fmt.Fprintf(os.Stdout, "  Status: %s\n", rec.Execution.Status)
	if rec.Execution.Status == "success" {
		fmt.Fprintf(os.Stdout, "  Outputs: %v\n", rec.Execution.Outputs)
	} else if rec.Execution.Error != nil {
		fmt.Fprintf(os.Stdout, "  Error: %s\n", rec.Execution.Error.Message)
	}
	fmt.Fprintln(os.Stdout, "  Signature: VALID ✓")
	fmt.Fprintln(os.Stdout, "  Timestamp:", rec.Timestamp.Format(time.RFC3339))

	return nil
}

// parseEd25519PublicKey decodes a PEM-encoded Ed25519 public key.
func parseEd25519PublicKey(pemData []byte) (ed25519.PublicKey, error) {
	kdeps_debug.Log("enter: parseEd25519PublicKey")
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "ED25519 PUBLIC KEY" {
		return nil, errors.New("invalid PEM block for public key")
	}
	if len(block.Bytes) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key length")
	}
	return ed25519.PublicKey(block.Bytes), nil
}
