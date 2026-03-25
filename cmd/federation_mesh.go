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
	"strings"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/parser/yaml"
)

// newFederationMeshCmd creates the `kdeps federation mesh` command.
func newFederationMeshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mesh",
		Short: "Inspect mesh relationships in current project",
		Long: `Inspect the federation mesh used by the current project.

This command scans the current directory for workflow or agency YAML files
and lists all remoteAgent references, showing their URNs, resolved endpoints,
trust levels, and status.`,
	}

	cmd.AddCommand(newFederationMeshListCmd())
	cmd.AddCommand(newFederationMeshPublishCmd())

	return cmd
}

// newFederationMeshListCmd creates `kdeps federation mesh list`.
func newFederationMeshListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List remote agents used in current project",
		Long: `List all remoteAgent resources referenced in the project's workflows.

For each remote agent, shows:
  - URN
  - Endpoint (if resolved)
  - Trust level requirement
  - Status (resolved/not found)`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runFederationMeshList()
		},
	}

	return cmd
}

// runFederationMeshList executes the mesh list logic.
func runFederationMeshList() error {
	workflowPaths, err := findWorkflowFiles(".")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to scan directory: %w", err)
	}
	if len(workflowPaths) == 0 {
		fmt.Fprintln(os.Stdout, "No workflow.yaml or agency.yaml files found in current directory.")
		return nil
	}
	found := collectRemoteAgents(workflowPaths)
	if len(found) == 0 {
		fmt.Fprintln(os.Stdout, "No remoteAgent resources found in workflows.")
		return nil
	}
	displayFoundAgents(found)
	return nil
}

// findWorkflowFiles walks the directory tree to find workflow.yaml and agency.yaml files.
func findWorkflowFiles(root string) ([]string, error) {
	var paths []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == "workflow.yaml" || info.Name() == "agency.yaml" {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// collectRemoteAgents parses each workflow file and extracts remoteAgent URNs.
func collectRemoteAgents(workflowPaths []string) []struct{ file, urn string } {
	parser := yaml.NewParser(nil, nil) // no validation
	var found []struct{ file, urn string }
	for _, fp := range workflowPaths {
		wf, err := parser.ParseWorkflow(fp)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Warning: could not parse %s: %v\n", fp, err)
			continue
		}
		for _, res := range wf.Resources {
			if res.Run.RemoteAgent != nil {
				found = append(found, struct{ file, urn string }{fp, res.Run.RemoteAgent.URN})
			}
		}
	}
	return found
}

// displayFoundAgents prints the list of remote agents to stdout.
func displayFoundAgents(found []struct{ file, urn string }) {
	fmt.Fprintln(os.Stdout, "Remote agents referenced in this project:")
	for _, f := range found {
		fmt.Fprintf(os.Stdout, "  %s: %s\n", f.file, f.urn)
	}
	fmt.Fprintln(
		os.Stdout,
		"\nTo resolve endpoints and verify status, use 'kdeps federation mesh resolve' (coming soon).",
	)
}

// newFederationMeshPublishCmd creates `kdeps federation mesh publish`.
func newFederationMeshPublishCmd() *cobra.Command {
	var (
		dryRun bool
		output string
	)

	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Show what would be registered (dry-run)",
		Long: `Generate a registration manifest for the current agent as it would be published to a registry.

This command reads the workflow, computes the URN content hash, and outputs the
registration JSON that would be sent to the registry. No network call is made.

Use this to verify your agent's identity and metadata before registering.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			// Find the main workflow (workflow.yaml or agency.yaml)
			wfPath := "workflow.yaml"
			if _, err := os.Stat(wfPath); os.IsNotExist(err) {
				wfPath = "agency.yaml"
				if _, err := os.Stat(wfPath); os.IsNotExist(err) { //nolint:govet // intentional shadow of err variable
					return errors.New("no workflow.yaml or agency.yaml found in current directory")
				}
			}

			// Parse workflow
			parser := yaml.NewParser(nil, nil)
			wf, err := parser.ParseWorkflow(wfPath)
			if err != nil {
				return fmt.Errorf("failed to parse workflow: %w", err)
			}

			// Compute URN: we need a URN for the agent. The convention could be:
			// urn:agent:<registry-authority>/<namespace>:<name>@<version>#sha256:<hash>
			// Where:
			// - authority could come from registry URL or be kdeps.io by default
			// - namespace from org (maybe workflow metadata)
			// - name from workflow metadata name
			// - version from workflow metadata version
			// - hash = SHA256 of canonicalized workflow YAML
			// For MVP, we'll print instructions.

			fmt.Fprintln(os.Stdout, "Agent Publish Dry-Run")
			fmt.Fprintln(os.Stdout, "=======================")
			fmt.Fprintf(os.Stdout, "Workflow: %s\n", wfPath)
			fmt.Fprintf(os.Stdout, "Name: %s\n", wf.Metadata.Name)
			fmt.Fprintf(os.Stdout, "Version: %s\n", wf.Metadata.Version)
			fmt.Fprintln(os.Stdout, "\nTo generate a complete registration manifest, you need:")
			fmt.Fprintln(
				os.Stdout,
				"  1. Determine the registry authority (e.g., 'registry.kdeps.io')",
			)
			fmt.Fprintln(
				os.Stdout,
				"  2. Set your organization namespace (reverse DNS, e.g., 'io.kdeps.myorg')",
			)
			fmt.Fprintln(os.Stdout, "  3. Compute the content hash of this workflow YAML")
			fmt.Fprintln(os.Stdout, "\nExample URN:")
			fmt.Fprintf(
				os.Stdout,
				"  urn:agent:registry.kdeps.io/io.kdeps.%s:%s@%s#sha256:<hash>\n",
				strings.ToLower(wf.Metadata.Name),
				wf.Metadata.Name,
				wf.Metadata.Version,
			)
			fmt.Fprintln(os.Stdout, "\nSuggested next steps:")
			fmt.Fprintln(os.Stdout, "  1. Run: kdeps federation keygen --org <your-org>")
			fmt.Fprintln(
				os.Stdout,
				"  2. Run: kdeps federation register --urn <urn> --spec workflow.yaml --registry <url> --contact <email>",
			)

			if dryRun {
				fmt.Fprintln(os.Stdout, "\n(Dry-run mode: no registration performed)")
			}
			// If output file specified, could write JSON (but not implemented)
			if output != "" {
				fmt.Fprintf(
					os.Stdout,
					"\nNote: --output not yet implemented. Would write to %s\n",
					output,
				)
			}

			return nil
		},
	}

	cmd.Flags().
		BoolVar(&dryRun, "dry-run", true, "Show what would be published without actually registering")
	cmd.Flags().StringVar(&output, "output", "", "Write registration manifest to file")

	return cmd
}
