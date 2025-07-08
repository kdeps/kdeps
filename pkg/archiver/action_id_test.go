package archiver

import (
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/logging"
	pklProject "github.com/kdeps/schema/gen/project"
	"github.com/spf13/afero"
)

func TestResolveActionIDWithAgentReader(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a test workflow
	testWf := &testWorkflow{
		agentID: "testagent",
		version: "1.0.0",
	}

	// Initialize agent reader using the global singleton
	agentReader, err := agent.GetGlobalAgentReader(fs, "", logger)
	if err != nil {
		t.Fatalf("failed to initialize agent reader: %v", err)
	}

	tests := []struct {
		name     string
		actionID string
		expected string
	}{
		{
			name:     "already qualified",
			actionID: "@other/action:2.1.0",
			expected: "@other/action:2.1.0",
		},
		{
			name:     "local action",
			actionID: "myAction",
			expected: "@testagent/myAction:1.0.0",
		},
		{
			name:     "action with version",
			actionID: "myAction:0.3.0",
			expected: "@testagent/myAction:1.0.0", // version from workflow takes precedence
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolvedID := resolveActionIDWithAgentReader(tt.actionID, testWf, agentReader)
			if resolvedID != tt.expected {
				t.Errorf("Test %s: expected %s, got %s", tt.name, tt.expected, resolvedID)
			}
		})
	}
}

func TestProcessRequiresBlockWithAgentReader(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a test workflow
	testWf := &testWorkflow{
		agentID: "testagent",
		version: "1.0.0",
	}

	// Initialize agent reader using the global singleton
	agentReader, err := agent.GetGlobalAgentReader(fs, "", logger)
	if err != nil {
		t.Fatalf("failed to initialize agent reader: %v", err)
	}

	input := strings.Join([]string{
		"",
		"    \"\"",                // quoted empty
		"    \"@otherAgent/foo\"", // @-prefixed without version
		"    \"localAction\"",     // plain quoted value
		"    \"config_value\"",    // config value should remain unchanged
	}, "\n")

	result, agentsToCopyAll := processRequiresBlockWithAgentReader(input, testWf, agentReader)
	lines := strings.Split(result, "\n")

	if lines[0] != "" {
		t.Errorf("blank line must stay blank, got: %q", lines[0])
	}
	if strings.TrimSpace(lines[1]) != "\"\"" {
		t.Errorf("quoted empty should remain quoted empty, got: %q", lines[1])
	}
	if !strings.Contains(lines[2], "@otherAgent/foo") {
		t.Errorf("external agent should remain unchanged, got: %q", lines[2])
	}
	if !strings.Contains(lines[3], "@testagent/localAction:1.0.0") {
		t.Errorf("local action should be resolved, got: %q", lines[3])
	}
	if strings.TrimSpace(lines[4]) != "\"config_value\"" {
		t.Errorf("quoted config_value should remain unchanged, got: %q", lines[4])
	}

	// Should not have any agents for copying all resources in this test
	if len(agentsToCopyAll) != 0 {
		t.Errorf("Expected no agents for copying all resources, got %v", agentsToCopyAll)
	}
}

func TestExtractNameVersionFromResolvedID(t *testing.T) {
	tests := []struct {
		name           string
		resolvedID     string
		defaultName    string
		defaultVersion string
		expectedName   string
		expectedVer    string
	}{
		{
			name:           "fully qualified",
			resolvedID:     "@agent/action:2.1.0",
			defaultName:    "default",
			defaultVersion: "1.0.0",
			expectedName:   "agent",
			expectedVer:    "2.1.0",
		},
		{
			name:           "not qualified",
			resolvedID:     "action",
			defaultName:    "default",
			defaultVersion: "1.0.0",
			expectedName:   "default",
			expectedVer:    "1.0.0",
		},
		{
			name:           "partial qualified",
			resolvedID:     "@agent/action",
			defaultName:    "default",
			defaultVersion: "1.0.0",
			expectedName:   "agent",
			expectedVer:    "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, ver := extractNameVersionFromResolvedID(tt.resolvedID, tt.defaultName, tt.defaultVersion)
			if name != tt.expectedName || ver != tt.expectedVer {
				t.Errorf("extractNameVersionFromResolvedID(%s, %s, %s) = (%s, %s), want (%s, %s)",
					tt.resolvedID, tt.defaultName, tt.defaultVersion, name, ver, tt.expectedName, tt.expectedVer)
			}
		})
	}
}

// Test workflow for testing
type testWorkflow struct {
	agentID string
	version string
}

func (m *testWorkflow) GetAgentID() string                { return m.agentID }
func (m *testWorkflow) GetVersion() string                { return m.version }
func (m *testWorkflow) GetDescription() *string           { desc := ""; return &desc }
func (m *testWorkflow) GetWebsite() *string               { return nil }
func (m *testWorkflow) GetAuthors() *[]string             { return nil }
func (m *testWorkflow) GetDocumentation() *string         { return nil }
func (m *testWorkflow) GetRepository() *string            { return nil }
func (m *testWorkflow) GetHeroImage() *string             { return nil }
func (m *testWorkflow) GetAgentIcon() *string             { return nil }
func (m *testWorkflow) GetTargetActionID() string         { return "" }
func (m *testWorkflow) GetWorkflows() []string            { return nil }
func (m *testWorkflow) GetSettings() *pklProject.Settings { return nil }

func TestRequiresBlockWithNoActions(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a test workflow
	testWf := &testWorkflow{
		agentID: "testagent",
		version: "1.0.0",
	}

	// Initialize agent reader using the global singleton
	agentReader, err := agent.GetGlobalAgentReader(fs, "", logger)
	if err != nil {
		t.Fatalf("failed to initialize agent reader: %v", err)
	}

	// Test requires block with agent name (no actions)
	requiresBlock := `requires = {
  myagent
  // This is a comment
  "config_value"
  "another_config"
}`

	processed, agentsToCopyAll := processRequiresBlockWithAgentReader(requiresBlock, testWf, agentReader)

	// Should contain the agent name for copying all resources
	if !strings.Contains(processed, "myagent") {
		t.Error("Expected agent name to be preserved")
	}

	// Should have the agent in the list for copying all resources
	if len(agentsToCopyAll) != 1 || agentsToCopyAll[0] != "myagent" {
		t.Errorf("Expected agentsToCopyAll to contain 'myagent', got %v", agentsToCopyAll)
	}

	// Should preserve the original config values
	if !strings.Contains(processed, `"config_value"`) {
		t.Error("Expected config_value to be preserved")
	}
	if !strings.Contains(processed, `"another_config"`) {
		t.Error("Expected another_config to be preserved")
	}
}

func TestRequiresBlockWithActions(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a test workflow
	testWf := &testWorkflow{
		agentID: "testagent",
		version: "1.0.0",
	}

	// Initialize agent reader using the global singleton
	agentReader, err := agent.GetGlobalAgentReader(fs, "", logger)
	if err != nil {
		t.Fatalf("failed to initialize agent reader: %v", err)
	}

	// Test requires block with actions
	requiresBlock := `requires = {
  "myAction"
  "anotherAction"
  "config_value"
}`

	processed, agentsToCopyAll := processRequiresBlockWithAgentReader(requiresBlock, testWf, agentReader)

	// Should NOT have any agents for copying all resources
	if len(agentsToCopyAll) != 0 {
		t.Errorf("Expected no agents for copying all resources, got %v", agentsToCopyAll)
	}

	// Should resolve the actions
	if !strings.Contains(processed, `"@testagent/myAction:1.0.0"`) {
		t.Error("Expected myAction to be resolved")
	}
	if !strings.Contains(processed, `"@testagent/anotherAction:1.0.0"`) {
		t.Error("Expected anotherAction to be resolved")
	}

	// Should preserve the config value
	if !strings.Contains(processed, `"config_value"`) {
		t.Error("Expected config_value to be preserved")
	}
}

func TestRequiresBlockWithAgentAndActions(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a test workflow
	testWf := &testWorkflow{
		agentID: "testagent",
		version: "1.0.0",
	}

	// Initialize agent reader using the global singleton
	agentReader, err := agent.GetGlobalAgentReader(fs, "", logger)
	if err != nil {
		t.Fatalf("failed to initialize agent reader: %v", err)
	}

	// Test requires block with both agent name and actions
	requiresBlock := `requires = {
  myagent
  "myAction"
  "anotherAction"
}`

	processed, agentsToCopyAll := processRequiresBlockWithAgentReader(requiresBlock, testWf, agentReader)

	// Should have the agent in the list for copying all resources
	if len(agentsToCopyAll) != 1 || agentsToCopyAll[0] != "myagent" {
		t.Errorf("Expected agentsToCopyAll to contain 'myagent', got %v", agentsToCopyAll)
	}

	// Should contain the agent name
	if !strings.Contains(processed, "myagent") {
		t.Error("Expected agent name to be preserved")
	}

	// Should resolve the actions
	if !strings.Contains(processed, `"@testagent/myAction:1.0.0"`) {
		t.Error("Expected myAction to be resolved")
	}
	if !strings.Contains(processed, `"@testagent/anotherAction:1.0.0"`) {
		t.Error("Expected anotherAction to be resolved")
	}
}
