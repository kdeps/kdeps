# Agent ID Resolution System

## Overview

The Agent ID Resolution System replaces the regex-based ID parsing with a robust PKL-native approach that provides better reliability, maintainability, and extensibility. This system operates at the PKL level and integrates seamlessly with the existing kdeps architecture.

**Key Improvement**: The system now uses temporary files for agent databases, providing better isolation and automatic cleanup.

## Architecture

### Core Components

1. **Agent Package (`pkg/agent/`)**: Implements the PKL ResourceReader interface for agent operations
2. **Agent Schema**: PKL schema definition for agent configuration and metadata
3. **Agent Template**: Template for generating agent.pkl files
4. **Resolver Integration**: Integration with the DependencyResolver for seamless operation

### Key Features

- **PKL-Native Resolution**: All ID resolution happens at the PKL level
- **Agent Registry**: Tracks installed agents and their metadata
- **Namespace Management**: Provides namespace isolation and resolution
- **Version Management**: Handles version constraints and resolution
- **Dependency Tracking**: Automatically tracks agent dependencies
- **Package Management**: Integrates with the packaging system
- **Temporary File Storage**: Uses temporary files for better isolation and automatic cleanup

## Usage

### Basic ID Resolution

```pkl
// Resolve a local action ID to a fully qualified agent ID
local resolvedID = "@(agent.resolve("actionName", agent="myAgent", version="1.0.0"))"
// Result: "@myAgent/actionName:1.0.0"

// Use the resolved ID in resource definitions
resource {
    [resolvedID] {
        // Resource configuration
    }
}
```

### Agent Management

```pkl
// List all installed agents
local agents = "@(agent.list())"

// Register a new agent
local result = "@(agent.register("@myAgent/action:1.0.0", path="/path/to/agent"))"

// Unregister an agent
local result = "@(agent.unregister("@myAgent/action:1.0.0"))"
```

### Agent Configuration

```pkl
// Agent configuration in workflow.pkl
agent {
    name = "myAgent"
    version = "1.0.0"
    
    settings {
        autoResolve = true
        defaultNamespace = "myAgent"
        versionPolicy = "strict"
    }
    
    capabilities {
        idResolution = true
        namespaceManagement = true
        dependencyTracking = true
    }
}
```

## Implementation Details

### Agent ResourceReader

The `PklResourceReader` in the agent package implements the following operations:

1. **resolve**: Resolves local action IDs to fully qualified agent IDs
2. **list**: Lists all installed agents
3. **register**: Registers a new agent in the database
4. **unregister**: Removes an agent from the database

### Temporary File Storage

The agent system uses temporary files for database storage, providing several benefits:

1. **Isolation**: Each agent operation uses a separate temporary database
2. **Automatic Cleanup**: Temporary files are automatically removed when the agent reader is closed
3. **No File Conflicts**: No risk of file conflicts between concurrent operations
4. **Security**: Temporary files are created with secure permissions

```go
// Temporary file creation
tmpFile, err := os.CreateTemp("", "agent_*.db")
if err != nil {
    return nil, fmt.Errorf("failed to create temporary file: %w", err)
}

// Automatic cleanup in Close() method
func (r *PklResourceReader) Close() error {
    // Close database
    if r.DB != nil {
        r.DB.Close()
    }
    
    // Remove temporary file
    if r.tmpFile != nil {
        os.Remove(r.tmpFile.Name())
    }
    
    return nil
}
```

### Database Schema

The agent database uses SQLite with the following schema:

```sql
CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    data TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

The `data` field contains JSON-encoded `AgentInfo` structures.

### Integration Points

1. **Resolver Integration**: The agent reader is integrated into the DependencyResolver
2. **Resource Loading**: Agent operations are available during PKL resource loading
3. **Import System**: Agent imports are handled by the import system
4. **Response Generation**: Agent data is included in API responses
5. **Automatic Cleanup**: Temporary files are cleaned up when the resolver is closed

## Migration from Regex-Based System

### Before (Regex-Based)

```go
// Old regex-based ID resolution
idPattern := regexp.MustCompile(`@([^/]+)/([^:]+):([^@]+)`)
matches := idPattern.FindStringSubmatch(content)
if len(matches) == 4 {
    agentName := matches[1]
    actionName := matches[2]
    version := matches[3]
    // Process ID components
}
```

### After (PKL-Native)

```pkl
// New PKL-native ID resolution
local resolvedID = "@(agent.resolve("actionName", agent="myAgent", version="1.0.0"))"
```

### Benefits of Migration

1. **Type Safety**: PKL provides compile-time type checking
2. **Error Handling**: Better error messages and validation
3. **Extensibility**: Easy to add new agent operations
4. **Maintainability**: Centralized agent logic
5. **Performance**: No regex compilation overhead
6. **Debugging**: Better debugging and logging capabilities
7. **Isolation**: Temporary files provide better isolation
8. **Cleanup**: Automatic cleanup of temporary resources

## Configuration

### Agent Settings

```pkl
agent {
    settings {
        // Enable automatic ID resolution
        autoResolve = true
        
        // Default namespace for local action IDs
        defaultNamespace = "myAgent"
        
        // Version management policy
        versionPolicy = "strict" // Options: "strict", "flexible", "latest"
        
        // Dependency resolution
        resolveDependencies = true
        
        // Package management
        autoPackage = true
    }
}
```

### Resolution Policies

```pkl
resolution {
    policies {
        // How to handle conflicts
        conflictResolution = "error" // Options: "error", "warn", "override"
        
        // Version resolution strategy
        versionStrategy = "exact" // Options: "exact", "semver", "latest"
        
        // Namespace fallback behavior
        namespaceFallback = "strict" // Options: "strict", "flexible", "auto"
    }
}
```

## Advanced Features

### Namespace Management

```pkl
// Get current namespace
local currentNamespace = "@(agent.namespace.current)"

// Set namespace
local result = "@(agent.namespace.set("newNamespace"))"

// Validate namespace
local isValid = "@(agent.namespace.validate("namespace"))"
```

### Version Management

```pkl
// Get current version
local currentVersion = "@(agent.version.current)"

// Compare versions
local comparison = "@(agent.version.compare("1.0.0", "1.1.0"))"

// Validate version
local isValid = "@(agent.version.validate("1.0.0"))"
```

### Dependency Tracking

```pkl
dependencies {
    resolution {
        // Automatic dependency resolution
        autoResolve = true
        
        // Dependency version constraints
        versionConstraints = true
        
        // Circular dependency detection
        circularDetection = true
    }
}
```

## Testing

### Unit Tests

The agent package includes comprehensive unit tests:

```bash
go test ./pkg/agent/...
```

### Integration Tests

Integration tests verify the agent system works with the full kdeps stack:

```bash
go test ./pkg/resolver/... -tags=integration
```

### Temporary File Testing

Tests verify that temporary files are properly created and cleaned up:

```go
func TestTemporaryFileCleanup(t *testing.T) {
    reader, err := InitializeAgent(fs, "/test/kdeps", logger)
    require.NoError(t, err)
    
    // Verify temporary file exists
    require.FileExists(t, reader.DBPath)
    
    // Close and verify cleanup
    err = reader.Close()
    require.NoError(t, err)
    
    // Verify temporary file is removed
    require.NoFileExists(t, reader.DBPath)
}
```

## Performance Considerations

1. **Temporary File Operations**: Agent operations use temporary SQLite databases
2. **Caching**: Resolution results are cached for performance
3. **Lazy Loading**: Agent metadata is loaded on-demand
4. **Connection Pooling**: Database connections are reused within the same operation
5. **Automatic Cleanup**: Temporary files are automatically removed, preventing disk space issues

## Security Considerations

1. **Input Validation**: All agent inputs are validated
2. **SQL Injection**: Parameterized queries prevent SQL injection
3. **Path Validation**: Agent paths are validated for security
4. **Access Control**: Agent operations respect access controls
5. **Temporary File Security**: Temporary files are created with secure permissions
6. **Automatic Cleanup**: Temporary files are automatically removed, preventing data leakage

## Future Enhancements

1. **Remote Registry**: Support for remote agent registries
2. **Agent Signing**: Digital signature verification for agents
3. **Agent Updates**: Automatic agent update mechanisms
4. **Multi-Tenancy**: Support for multi-tenant agent environments
5. **Agent Metrics**: Performance and usage metrics collection
6. **Persistent Storage**: Optional persistent storage for agent metadata

## Troubleshooting

### Common Issues

1. **Agent Not Found**: Check if the agent is properly registered
2. **Version Mismatch**: Verify version constraints and policies
3. **Namespace Issues**: Check namespace configuration and fallback settings
4. **Database Errors**: Verify database connectivity and permissions
5. **Temporary File Issues**: Check temporary directory permissions and disk space

### Debugging

Enable debug logging to troubleshoot agent operations:

```bash
export KDEPS_LOG_LEVEL=debug
```

### Logs

Agent operations are logged with structured logging:

```json
{
  "level": "info",
  "msg": "Resolved action ID",
  "actionID": "actionName",
  "resolvedID": "@myAgent/actionName:1.0.0",
  "agent": "myAgent",
  "version": "1.0.0",
  "tempFile": "/tmp/agent_123456.db"
}
```

## API Reference

### Agent Functions

- `agent.resolve(actionID, agent, version)`: Resolve local ID to qualified ID
- `agent.list()`: List all installed agents
- `agent.register(agentID, path)`: Register a new agent
- `agent.unregister(agentID)`: Unregister an agent

### Agent Properties

- `agent.name`: Current agent name
- `agent.version`: Current agent version
- `agent.settings`: Agent configuration settings
- `agent.capabilities`: Agent capabilities
- `agent.metadata`: Agent metadata

### Resolution Properties

- `resolution.policies`: Resolution policies
- `resolution.cache`: Cache settings
- `resolution.validation`: Validation rules

## Conclusion

The Agent ID Resolution System provides a robust, maintainable, and extensible solution for agent ID management in kdeps. By moving from regex-based parsing to PKL-native resolution and using temporary files for storage, the system gains type safety, better error handling, improved performance, and enhanced security while maintaining backward compatibility. 