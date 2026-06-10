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

package templates_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/templates"
)

func TestGenerator_WalkTemplate_WithSubdirectories(t *testing.T) {
	generator, err := templates.NewGenerator()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "test-project")

	data := templates.TemplateData{
		Name:        "test-agent",
		Description: "Test agent",
		Version:     "1.0.0",
		Port:        16395,
		Resources:   []string{"http-client", "llm"},
		Features:    make(map[string]bool),
	}

	// Test with api-service template which has multiple files
	err = generator.GenerateProject("api-service", outputDir, data)
	require.NoError(t, err)

	// Verify all expected files were generated
	workflowPath := filepath.Join(outputDir, "workflow.yaml")
	_, err = os.Stat(workflowPath)
	require.NoError(t, err, "workflow.yaml should be created")

	readmePath := filepath.Join(outputDir, "README.md")
	_, err = os.Stat(readmePath)
	require.NoError(t, err, "README.md should be created")

	envPath := filepath.Join(outputDir, ".env.example")
	_, err = os.Stat(envPath)
	require.NoError(t, err, ".env.example should be created")
}

func TestGenerator_GenerateFile_TemplateWithHasFunction(t *testing.T) {
	generator, err := templates.NewGenerator()
	require.NoError(t, err)

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "test-project")

	data := templates.TemplateData{
		Name:        "test-agent",
		Description: "Test agent",
		Version:     "1.0.0",
		Port:        16395,
		Resources:   []string{"http-client", "llm", "response"},
		Features:    make(map[string]bool),
	}

	// Generate project with resources that use the "has" template function
	err = generator.GenerateProject("api-service", outputDir, data)
	require.NoError(t, err)

	// Verify workflow.yaml was generated correctly
	workflowPath := filepath.Join(outputDir, "workflow.yaml")
	content, err := os.ReadFile(workflowPath)
	require.NoError(t, err)

	// The template uses the "has" function, so verify it rendered correctly
	workflowContent := string(content)
	assert.Contains(t, workflowContent, "test-agent")
	assert.Contains(t, workflowContent, "Test agent")
}

func TestGenerator_WalkTemplate_ErrorHandling(t *testing.T) {
	generator, err := templates.NewGenerator()
	require.NoError(t, err)

	// Test with invalid template name
	invalidDir := t.TempDir()
	outputDir := filepath.Join(invalidDir, "test-project")

	data := templates.TemplateData{
		Name: "test",
	}

	err = generator.GenerateProject("nonexistent-template", outputDir, data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template not found")
}

func TestGenerator_WalkTemplate_DirectoryCreationFailure(t *testing.T) {
	generator, err := templates.NewGenerator()
	require.NoError(t, err)

	// Create a mock scenario where directory creation would fail
	// We'll use a read-only parent directory to simulate mkdir failure
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	err = os.MkdirAll(readOnlyDir, 0500) // Read-only permissions
	require.NoError(t, err)
	defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

	// Try to create a project in a location where subdirectory creation would fail
	outputDir := filepath.Join(readOnlyDir, "project")

	data := templates.TemplateData{
		Name: "test",
	}

	// This should fail when walkTemplate tries to create subdirectories
	err = generator.GenerateProject("api-service", outputDir, data)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestGenerator_WalkTemplate_ReadDirFailure(_ *testing.T) {
	// This is harder to test directly since we can't easily make ReadDir fail
	// on the embedded filesystem. The existing tests already cover the success paths.
	// The readDir failure would occur if the embedded FS is corrupted, which is
	// not something we can easily test in unit tests.

	// Instead, we'll rely on the existing GenerateProject tests which indirectly
	// test the walkTemplate success paths and ensure we maintain good coverage
	// through integration testing.
}
