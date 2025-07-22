# Asynchronous PKL Resource System (Pklres)

## Overview

Pklres is a **generic key-value store** that can record anything from shallow to deep nested data without schema restrictions. It's scoped by `graphID` and uses `actionID` as collection keys.

## Key Features

### 1. Generic Storage
- **No schema restrictions**: Can store any data structure
- **Flexible data types**: From simple strings to complex nested objects
- **JSON-based**: All values are stored and retrieved as JSON

### 2. Simple Operations
- `get(collectionKey, key)` - Retrieve a value
- `set(collectionKey, key, value)` - Store a value  
- `list(collectionKey)` - List all keys in a collection

### 3. Scoping and Organization
- **Scope**: `graphID` (request ID) ensures data isolation between workflow executions
- **Collection**: `actionID` (canonicalized) organizes data by resource
- **Storage Structure**: `graphID -> actionID -> key -> value`

## Architecture

### Storage Structure
```
graphID (request ID) -> actionID (canonical) -> key -> value (JSON)
```

### Key Components

1. **PklResourceReader**: Implements the generic key-value store
2. **PklresHelper**: Provides a simple interface for Go code
3. **PKL Functions**: Core functions for PKL integration

### 3. Asynchronous Dependency Resolution

Pklres supports async dependency resolution with dependency tracking:

```go
// Pre-resolve dependencies based on execution order
err := pklres.PreResolveDependencies(executionOrder, dependencies)

// Update dependency status
err := pklres.UpdateDependencyStatus(actionID, "completed", resultData, nil)

// Wait for dependencies to be ready
err := pklres.WaitForDependencies(actionID, timeout)
```

### 4. Collection Key Handling

All collection keys are actionIDs that are automatically canonicalized using the agent resolver:

```go
// Always canonicalize the collection key using Agent reader
canonicalCollectionKey := resolveActionID(collectionKey)
```

This ensures that:
- Resource types like "llm", "exec", "python" are resolved to their canonical actionID format
- Action IDs like "@localproject/llmResource:1.0.0" are resolved to their fully qualified form
- All pklres operations use consistent, canonical actionIDs as collection keys

### 5. Scope and Graph Validation

Pklres operations use graphID as the scope and validate against the dependency graph:

**Scope**: All pklres operations are scoped by graphID (request ID) to ensure data isolation between different workflow executions.

**Graph Validation**: Pklres operations are only processed for resources that exist in the dependency graph:

```go
// Check if this collection key exists in the dependency graph
if !r.IsInDependencyGraph(canonicalCollectionKey) {
    // Return null for get operations, ignore set operations
    return []byte("null"), nil
}
```

**Storage Structure**: `graphID -> actionID -> key -> value`

## Example Workflow

### 1. Initialize Pklres
```go
reader, err := pklres.InitializePklResource("workflow-123", "myagent", "1.0.0", "/path/to/kdeps", fs)
```

### 2. Store Data
```go
// Store simple values
err := pklres.Set("@myagent/llm:1.0.0", "response", "Hello, world!")

// Store complex data
complexData := map[string]interface{}{
    "user": "john",
    "data": []string{"item1", "item2"},
    "nested": map[string]interface{}{
        "key": "value",
    },
}
jsonData, _ := json.Marshal(complexData)
err := pklres.Set("@myagent/data:1.0.0", "user_data", string(jsonData))
```

### 3. Retrieve Data
```go
// Get simple values
response, err := pklres.Get("@myagent/llm:1.0.0", "response")

// Get complex data
jsonData, err := pklres.Get("@myagent/data:1.0.0", "user_data")
var userData map[string]interface{}
json.Unmarshal([]byte(jsonData), &userData)
```

### 4. List Keys
```go
keys, err := pklres.List("@myagent/llm:1.0.0")
// Returns: ["response", "model", "prompt", ...]
```

## PKL Integration

### Core Functions
```pkl
// Generic key-value operations
function get(collectionKey: String?, key: String?): String
function set(collectionKey: String?, key: String?, value: String?): String  
function list(collectionKey: String?): Listing<String>

// Legacy functions for backward compatibility
function getPklValue(id: String?, typ: String?, key: String?): String
function setPklValue(id: String?, typ: String?, key: String?, value: String?): String
function getAllRecords(typ: String?): Listing<String>
```

### Usage Examples
```pkl
// Store data
set("@localproject/llm:1.0.0", "response", "AI response here")
set("@localproject/exec:1.0.0", "output", "Command output")

// Retrieve data  
response = get("@localproject/llm:1.0.0", "response")
output = get("@localproject/exec:1.0.0", "output")

// List available keys
keys = list("@localproject/llm:1.0.0")
```

## Pklres Operations

- `pklres.get("@localproject/llm:1.0.0", "key")` - Returns cached value from canonicalized actionID collection
- `pklres.get("@localproject/exec:1.0.0", "key")` - Returns cached value from canonicalized actionID collection
- `pklres.get("@localproject/llmResource:1.0.0", "key")` - Returns cached value from canonicalized action ID
- `pklres.get("unknownResource", "key")` - Returns null (not in dependency graph)

## Benefits

1. **Simplicity**: Pure key-value store without schema restrictions
2. **Flexibility**: Can store any data structure from simple to complex
3. **Consistency**: All collection keys are canonicalized actionIDs
4. **Isolation**: GraphID scoping ensures data isolation
5. **Performance**: In-memory storage with async dependency resolution
6. **Backward Compatibility**: Legacy functions still work

## Implementation Details

### Go Interface
```go
type PklresHelper struct {
    resolver *DependencyResolver
}

func (h *PklresHelper) Get(collectionKey, key string) (string, error)
func (h *PklresHelper) Set(collectionKey, key, value string) error  
func (h *PklresHelper) List(collectionKey string) ([]string, error)
```

### Storage Backend
```go
// In-memory storage: graphID -> actionID -> key -> value (JSON string)
store map[string]map[string]map[string]string
```

### URI Format
```
pklres:///graphID?collection=actionID&key=key&op=get|set|list&value=value
``` 