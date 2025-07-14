package agent

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestPklResourceReader(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Initialize agent reader with temporary database
	reader, err := InitializeAgent(fs, "/test/kdeps", "testAgent", "1.0.0", logger)
	if err != nil {
		t.Fatalf("failed to initialize agent reader: %v", err)
	}
	defer reader.Close()

	t.Run("Scheme", func(t *testing.T) {
		require.Equal(t, "agent", reader.Scheme())
	})

	t.Run("IsGlobbable", func(t *testing.T) {
		require.False(t, reader.IsGlobbable())
	})

	t.Run("HasHierarchicalUris", func(t *testing.T) {
		require.False(t, reader.HasHierarchicalUris())
	})

	t.Run("ListElements", func(t *testing.T) {
		uri, _ := url.Parse("agent:///test")
		elements, err := reader.ListElements(*uri)
		require.NoError(t, err)
		require.Nil(t, elements)
	})

	t.Run("ResolveAgentID_AlreadyQualified", func(t *testing.T) {
		uri, _ := url.Parse("agent:///@myAgent/action:1.0.0")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("@myAgent/action:1.0.0"), data)
	})

	t.Run("ResolveAgentID_LocalToQualified", func(t *testing.T) {
		uri, _ := url.Parse("agent:///action?agent=myAgent&version=1.0.0")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("@myAgent/action:1.0.0"), data)
	})

	t.Run("ResolveAgentID_MissingAgent", func(t *testing.T) {
		// Should use context from agent reader (testAgent:1.0.0)
		uri, _ := url.Parse("agent:///action?version=1.0.0")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("@testAgent/action:1.0.0"), data)
	})

	t.Run("ResolveAgentID_MissingVersion", func(t *testing.T) {
		// Should use context from agent reader (testAgent:1.0.0)
		uri, _ := url.Parse("agent:///action?agent=myAgent")
		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Equal(t, []byte("@myAgent/action:1.0.0"), data)
	})

	t.Run("ResolveAgentID_EmptyID", func(t *testing.T) {
		uri, _ := url.Parse("agent:///?agent=myAgent&version=1.0.0")
		_, err := reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no action ID provided")
	})

	t.Run("ResolveAgentID_NoContextOrParams", func(t *testing.T) {
		// Create reader with no context
		emptyReader, err := InitializeAgent(fs, "/test/kdeps", "", "", logger)
		require.NoError(t, err)
		defer emptyReader.Close()

		// Should fail when no agent/version in context or params
		uri, _ := url.Parse("agent:///action")
		_, err = emptyReader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "agent name and version required")
	})

	t.Run("ListInstalledAgents", func(t *testing.T) {
		// Create test agent directory structure
		agentsDir := filepath.Join("/test/kdeps", "agents")
		agent1Dir := filepath.Join(agentsDir, "agent1", "1.0.0")
		agent2Dir := filepath.Join(agentsDir, "agent2", "2.0.0")

		fs.MkdirAll(agent1Dir, 0o755)
		fs.MkdirAll(agent2Dir, 0o755)

		// Create workflow.pkl files
		afero.WriteFile(fs, filepath.Join(agent1Dir, "workflow.pkl"), []byte("test"), 0o644)
		afero.WriteFile(fs, filepath.Join(agent2Dir, "workflow.pkl"), []byte("test"), 0o644)

		uri, _ := url.Parse("agent:///?op=list")
		data, err := reader.Read(*uri)
		require.NoError(t, err)

		var agents []AgentInfo
		err = json.Unmarshal(data, &agents)
		require.NoError(t, err)
		require.Len(t, agents, 2)

		// Check agent1
		found := false
		for _, agent := range agents {
			if agent.Name == "agent1" && agent.Version == "1.0.0" {
				found = true
				break
			}
		}
		require.True(t, found, "agent1 should be found")

		// Check agent2
		found = false
		for _, agent := range agents {
			if agent.Name == "agent2" && agent.Version == "2.0.0" {
				found = true
				break
			}
		}
		require.True(t, found, "agent2 should be found")
	})

	t.Run("RegisterAgent", func(t *testing.T) {
		agentID := "@testAgent/action:1.0.0"
		agentPath := "/test/path"
		uri, _ := url.Parse(fmt.Sprintf("agent:///%s?op=register&path=%s", agentID, agentPath))

		data, err := reader.Read(*uri)
		require.NoError(t, err)
		require.Contains(t, string(data), "Registered agent")

		// Verify it was stored in the database
		var storedData string
		err = reader.DB.QueryRow("SELECT data FROM agents WHERE id = ?", agentID).Scan(&storedData)
		require.NoError(t, err)

		var agentInfo AgentInfo
		err = json.Unmarshal([]byte(storedData), &agentInfo)
		require.NoError(t, err)
		require.Equal(t, "testAgent", agentInfo.Name)
		require.Equal(t, "1.0.0", agentInfo.Version)
		require.Equal(t, agentPath, agentInfo.Path)
	})

	t.Run("RegisterAgent_InvalidID", func(t *testing.T) {
		uri, _ := url.Parse("agent:///invalid?op=register&path=/test")
		_, err := reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid agent ID format")
	})

	t.Run("RegisterAgent_MissingPath", func(t *testing.T) {
		uri, _ := url.Parse("agent:///@testAgent/action:1.0.0?op=register")
		_, err := reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "agent path required")
	})

	t.Run("UnregisterAgent", func(t *testing.T) {
		agentID := "@testAgent/action:1.0.0"

		// First register the agent
		registerURI, _ := url.Parse(fmt.Sprintf("agent:///%s?op=register&path=/test", agentID))
		_, err := reader.Read(*registerURI)
		require.NoError(t, err)

		// Then unregister it
		unregisterURI, _ := url.Parse(fmt.Sprintf("agent:///%s?op=unregister", agentID))
		data, err := reader.Read(*unregisterURI)
		require.NoError(t, err)
		require.Contains(t, string(data), "Unregistered agent")

		// Verify it was removed from the database
		var count int
		err = reader.DB.QueryRow("SELECT COUNT(*) FROM agents WHERE id = ?", agentID).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count)
	})

	t.Run("UnregisterAgent_EmptyID", func(t *testing.T) {
		uri, _ := url.Parse("agent:///?op=unregister")
		_, err := reader.Read(*uri)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no agent ID provided")
	})
}

func TestInitializeDatabase(t *testing.T) {
	t.Run("SuccessfulInitialization", func(t *testing.T) {
		db, err := InitializeDatabase(":memory:")
		require.NoError(t, err)
		require.NotNil(t, db)
		defer db.Close()

		var name string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='agents'").Scan(&name)
		require.NoError(t, err)
		require.Equal(t, "agents", name)
	})
}

func TestInitializeAgent(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	reader, err := InitializeAgent(fs, "/test/kdeps", "testAgent", "1.0.0", logger)
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.NotNil(t, reader.DB)
	require.NotEmpty(t, reader.DBPath)
	require.Equal(t, fs, reader.Fs)
	require.Equal(t, "/test/kdeps", reader.KdepsDir)
	require.Equal(t, "testAgent", reader.CurrentAgent)
	require.Equal(t, "1.0.0", reader.CurrentVersion)
	require.Equal(t, logger, reader.Logger)
	defer reader.Close()
}

func TestAgentInfo_JSON(t *testing.T) {
	agentInfo := AgentInfo{
		Name:    "testAgent",
		Version: "1.0.0",
		Path:    "/test/path",
	}

	jsonData, err := json.Marshal(agentInfo)
	require.NoError(t, err)

	var unmarshaled AgentInfo
	err = json.Unmarshal(jsonData, &unmarshaled)
	require.NoError(t, err)
	require.Equal(t, agentInfo, unmarshaled)
}

func TestTemporaryFileCleanup(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create agent reader
	reader, err := InitializeAgent(fs, "/test/kdeps", "testAgent", "1.0.0", logger)
	require.NoError(t, err)
	require.NotNil(t, reader)

	// Verify it's an in-memory database
	require.Equal(t, ":memory:", reader.DBPath)

	// Perform some operations to ensure the database is working
	uri, _ := url.Parse("agent:///@testAgent/action:1.0.0?op=register&path=/test/path")
	_, err = reader.Read(*uri)
	require.NoError(t, err)

	// Close and verify cleanup (should not fail for in-memory databases)
	err = reader.Close()
	require.NoError(t, err)

	// For in-memory databases, there's no file to clean up
	// The database is automatically cleaned up when closed
}

func TestRegisterAllAgentsAndActions(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create test agent directory structure with workflow.pkl files
	agentsDir := filepath.Join("/test/kdeps", "agents")
	agent1Dir := filepath.Join(agentsDir, "agent1", "1.0.0")
	agent2Dir := filepath.Join(agentsDir, "agent2", "2.0.0")

	fs.MkdirAll(agent1Dir, 0o755)
	fs.MkdirAll(agent2Dir, 0o755)

	// Create workflow.pkl files with actionIDs
	workflow1Content := `amends "package://schema.kdeps.com/core@0.4.2#/Workflow.pkl"

AgentID = "agent1"
Version = "1.0.0"
TargetActionID = "action1"

ActionID = "action1"
ActionID = "action2"
ActionID = "action3"
`
	workflow2Content := `amends "package://schema.kdeps.com/core@0.4.2#/Workflow.pkl"

AgentID = "agent2"
Version = "2.0.0"
TargetActionID = "action4"

ActionID = "action4"
ActionID = "action5"
`

	afero.WriteFile(fs, filepath.Join(agent1Dir, "workflow.pkl"), []byte(workflow1Content), 0o644)
	afero.WriteFile(fs, filepath.Join(agent2Dir, "workflow.pkl"), []byte(workflow2Content), 0o644)

	// Initialize agent reader
	reader, err := InitializeAgent(fs, "/test/kdeps", "agent1", "1.0.0", logger)
	require.NoError(t, err)
	defer reader.Close()

	// Verify that agents and actions were registered
	// Check agent1:version
	var count int
	err = reader.DB.QueryRow("SELECT COUNT(*) FROM agents WHERE id = ?", "@agent1:1.0.0").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Check agent2:version
	err = reader.DB.QueryRow("SELECT COUNT(*) FROM agents WHERE id = ?", "@agent2:2.0.0").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Check action registrations
	err = reader.DB.QueryRow("SELECT COUNT(*) FROM agents WHERE id = ?", "@agent1/action1:1.0.0").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = reader.DB.QueryRow("SELECT COUNT(*) FROM agents WHERE id = ?", "@agent1/action2:1.0.0").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	err = reader.DB.QueryRow("SELECT COUNT(*) FROM agents WHERE id = ?", "@agent2/action4:2.0.0").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Test version resolution
	uri, _ := url.Parse("agent:///@agent1")
	data, err := reader.Read(*uri)
	require.NoError(t, err)
	require.Equal(t, []byte("@agent1:1.0.0"), data)

	uri, _ = url.Parse("agent:///@agent1/action2")
	data, err = reader.Read(*uri)
	require.NoError(t, err)
	require.Equal(t, []byte("@agent1/action2:1.0.0"), data)
}

func TestLatestVersionResolution(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create test agent directory structure with multiple versions
	agentsDir := filepath.Join("/test/kdeps", "agents")
	agent1v1Dir := filepath.Join(agentsDir, "agent1", "1.0.0")
	agent1v2Dir := filepath.Join(agentsDir, "agent1", "2.0.0")
	agent1v10Dir := filepath.Join(agentsDir, "agent1", "10.0.0")

	fs.MkdirAll(agent1v1Dir, 0o755)
	fs.MkdirAll(agent1v2Dir, 0o755)
	fs.MkdirAll(agent1v10Dir, 0o755)

	// Create workflow.pkl files
	workflowContent := `amends "package://schema.kdeps.com/core@0.4.2#/Workflow.pkl"

AgentID = "agent1"
TargetActionID = "action1"

ActionID = "action1"
ActionID = "action2"
`

	afero.WriteFile(fs, filepath.Join(agent1v1Dir, "workflow.pkl"), []byte(workflowContent), 0o644)
	afero.WriteFile(fs, filepath.Join(agent1v2Dir, "workflow.pkl"), []byte(workflowContent), 0o644)
	afero.WriteFile(fs, filepath.Join(agent1v10Dir, "workflow.pkl"), []byte(workflowContent), 0o644)

	// Initialize agent reader
	reader, err := InitializeAgent(fs, "/test/kdeps", "agent1", "2.0.0", logger)
	require.NoError(t, err)
	defer reader.Close()

	// Test that @agent1 resolves to the latest version (10.0.0)
	uri, _ := url.Parse("agent:///@agent1")
	data, err := reader.Read(*uri)
	require.NoError(t, err)
	require.Equal(t, []byte("@agent1:10.0.0"), data)

	// Test that @agent1/action1 resolves to the latest version
	uri, _ = url.Parse("agent:///@agent1/action1")
	data, err = reader.Read(*uri)
	require.NoError(t, err)
	require.Equal(t, []byte("@agent1/action1:10.0.0"), data)

	// Test that specific versions still work
	uri, _ = url.Parse("agent:///@agent1:1.0.0")
	data, err = reader.Read(*uri)
	require.NoError(t, err)
	require.Equal(t, []byte("@agent1:1.0.0"), data)

	uri, _ = url.Parse("agent:///@agent1/action2:2.0.0")
	data, err = reader.Read(*uri)
	require.NoError(t, err)
	require.Equal(t, []byte("@agent1/action2:2.0.0"), data)
}
