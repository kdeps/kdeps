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
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package cmd_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/cmd"
)

func TestPackageWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (string, string)
		wantErr     bool
		errContains string
		verify      func(t *testing.T, outputDir string)
	}{
		{
			name: "successful packaging",
			setup: func(t *testing.T) (string, string) {
				sourceDir := t.TempDir()
				outputDir := t.TempDir()

				// Create workflow.yaml
				workflowContent := `
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-workflow
  version: "1.0.0"
  targetActionId: test-action
settings:
  agentSettings:
    pythonVersion: "3.12"
  apiServer:
    portNum: 16395
`
				require.NoError(
					t,
					os.WriteFile(
						filepath.Join(sourceDir, "workflow.yaml"),
						[]byte(workflowContent),
						0600,
					),
				)

				// Create resources directory and file
				resourcesDir := filepath.Join(sourceDir, "resources")
				require.NoError(t, os.Mkdir(resourcesDir, 0750))

				resourceContent := `
actionId: test-action
name: Test Action
apiResponse:
  success: true
  response:
    message: "test"
`
				require.NoError(
					t,
					os.WriteFile(
						filepath.Join(resourcesDir, "test-action.yaml"),
						[]byte(resourceContent),
						0600,
					),
				)

				return sourceDir, outputDir
			},
			wantErr: false,
			verify: func(t *testing.T, outputDir string) {
				// Check if package file was created
				matches, err := filepath.Glob(filepath.Join(outputDir, "*.kdeps"))
				require.NoError(t, err)
				require.Len(t, matches, 1)

				// Check if docker-compose.yml was created
				composePath := filepath.Join(outputDir, "docker-compose.yml")
				require.FileExists(t, composePath)
			},
		},
		{
			name: "invalid workflow directory - missing workflow.yaml",
			setup: func(t *testing.T) (string, string) {
				sourceDir := t.TempDir()
				outputDir := t.TempDir()

				// Create resources directory but no workflow.yaml
				resourcesDir := filepath.Join(sourceDir, "resources")
				require.NoError(t, os.Mkdir(resourcesDir, 0750))

				return sourceDir, outputDir
			},
			wantErr:     true,
			errContains: "invalid workflow directory",
		},
		{
			name: "invalid workflow directory - missing resources",
			setup: func(t *testing.T) (string, string) {
				sourceDir := t.TempDir()
				outputDir := t.TempDir()

				// Create workflow.yaml but no resources directory
				require.NoError(
					t,
					os.WriteFile(filepath.Join(sourceDir, "workflow.yaml"), []byte("test"), 0600),
				)

				return sourceDir, outputDir
			},
			wantErr:     true,
			errContains: "invalid workflow directory",
		},
		{
			name: "invalid workflow yaml",
			setup: func(t *testing.T) (string, string) {
				sourceDir := t.TempDir()
				outputDir := t.TempDir()

				// Create invalid workflow.yaml
				require.NoError(
					t,
					os.WriteFile(
						filepath.Join(sourceDir, "workflow.yaml"),
						[]byte("invalid: yaml: content: ["),
						0600,
					),
				)

				// Create resources directory
				resourcesDir := filepath.Join(sourceDir, "resources")
				require.NoError(t, os.Mkdir(resourcesDir, 0750))

				return sourceDir, outputDir
			},
			wantErr:     true,
			errContains: "failed to parse workflow",
		},
		{
			name: "custom package name",
			setup: func(t *testing.T) (string, string) {
				sourceDir := t.TempDir()
				outputDir := t.TempDir()

				// Create valid workflow.yaml
				workflowContent := `
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-workflow
  version: "1.0.0"
  targetActionId: test-action
settings:
  agentSettings:
    pythonVersion: "3.12"
  apiServer:
    portNum: 16395
`
				require.NoError(
					t,
					os.WriteFile(
						filepath.Join(sourceDir, "workflow.yaml"),
						[]byte(workflowContent),
						0600,
					),
				)

				// Create resources directory
				resourcesDir := filepath.Join(sourceDir, "resources")
				require.NoError(t, os.Mkdir(resourcesDir, 0750))

				return sourceDir, outputDir
			},
			wantErr: false,
			verify: func(t *testing.T, outputDir string) {
				// Check if package file was created with custom name
				customPkgPath := filepath.Join(outputDir, "custom-package.kdeps")
				require.FileExists(t, customPkgPath)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceDir, outputDir := tt.setup(t)

			flags := &cmd.PackageFlags{Output: outputDir, Name: "custom-package"}
			args := []string{sourceDir}
			err := cmd.PackageWorkflowWithFlags(&cobra.Command{}, args, flags)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				require.NoError(t, err)
				if tt.verify != nil {
					tt.verify(t, outputDir)
				}
			}
		})
	}
}
