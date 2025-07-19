# Async PKLRes Dependency Resolution

The async PKLRes system provides automatic dependency resolution and caching for pklres operations during workflow execution. This system ensures that pklres key values are pre-resolved and available when needed, based on the graph resolver's execution order.

## Overview

When a run action is executed, the system:

1. **Pre-resolves all pklres dependencies** based on the graph resolver's execution order
2. **Caches dependency relationships** between resources
3. **Manages async execution** where resources wait for their dependencies to complete
4. **Ignores pklres calls** that don't exist in the pre and post graph execution

## How It Works

### 1. Dependency Pre-Resolution

During `HandleRunAction()`, the system calls `PreResolveDependencies()` with:
- The execution order from the graph resolver
- The resource dependencies map

```go
// Pre-resolve all pklres dependencies based on the execution order
if dr.PklresReader != nil {
    if err := dr.PklresReader.PreResolveDependencies(stack, dr.ResourceDependencies); err != nil {
        // Handle error
    }
}
```

### 2. Dependency Status Management

Each resource in the execution order gets a `DependencyData` structure:

```go
type DependencyData struct {
    ActionID      string    `json:"actionID"`
    Dependents    []string  `json:"dependents"`     // Resources that depend on this
    Dependencies  []string  `json:"dependencies"`   // Resources this depends on
    Status        string    `json:"status"`         // "pending", "processing", "completed", "error"
    ResultData    string    `json:"resultData"`     // Actual execution results
    Timestamp     int64     `json:"timestamp"`
    Error         string    `json:"error,omitempty"`
    CompletedAt   int64     `json:"completedAt,omitempty"`
}
```

### 3. Async Execution Flow

When processing a resource:

1. **Check dependencies**: The system checks if all dependencies are ready
2. **Wait if needed**: If dependencies aren't ready, the resource waits
3. **Update status**: Mark resource as "processing"
4. **Execute**: Run the resource handler
5. **Complete**: Mark resource as "completed" or "error"

```go
// Check if this resource's dependencies are ready
if dr.PklresReader != nil {
    depData, err := dr.PklresReader.GetDependencyData(canonicalResourceID)
    if err == nil && depData != nil {
        // Update status to processing
        dr.PklresReader.UpdateDependencyStatus(canonicalResourceID, "processing", "", nil)
        
        // Wait for all dependencies to be ready
        if len(depData.Dependencies) > 0 {
            dr.PklresReader.WaitForDependencies(canonicalResourceID, waitTimeout)
        }
    }
}
```

### 4. Graph Validation

Pklres operations are only processed for resources that exist in the dependency graph:

```go
// Check if this collection key exists in the dependency graph
if !r.IsInDependencyGraph(canonicalCollectionKey) {
    // Return null for get operations, ignore set operations
    return []byte("null"), nil
}
```

## Example Workflow

Consider a workflow with three resources:

```apl
// Resource A: No dependencies
ActionID = "resourceA"
Requires {}

// Resource B: Depends on A
ActionID = "resourceB"
Requires {
    "resourceA"
}

// Resource C: Depends on B
ActionID = "resourceC"
Requires {
    "resourceB"
}
```

### Execution Flow

1. **Pre-resolution**: All three resources are marked as "pending"
2. **Resource A**: No dependencies, executes immediately, marked "completed"
3. **Resource B**: Waits for A to complete, then executes, marked "completed"
4. **Resource C**: Waits for B to complete, then executes, marked "completed"

### Pklres Operations

- `pklres.get("resourceA", "key")` - Returns cached value if A is completed
- `pklres.get("resourceB", "key")` - Returns cached value if B is completed
- `pklres.get("unknownResource", "key")` - Returns null (not in dependency graph)

## Benefits

1. **Performance**: Pklres values are pre-resolved and cached
2. **Reliability**: Dependencies are automatically managed
3. **Consistency**: Only resources in the execution graph are processed
4. **Monitoring**: Dependency status is tracked and logged
5. **Error Handling**: Failed dependencies are properly propagated

## Monitoring

The system provides several methods for monitoring dependency status:

```go
// Get status summary of all dependencies
statusSummary := reader.GetDependencyStatusSummary()

// Get list of pending dependencies
pendingDeps := reader.GetPendingDependencies()

// Check if specific dependency is ready
isReady := reader.IsDependencyReady("resourceID")

// Wait for dependencies with timeout
err := reader.WaitForDependencies("resourceID", 5*time.Minute)
```

## Configuration

The async pklres system is automatically enabled when:
- A `PklresReader` is available in the `DependencyResolver`
- The workflow has resources with dependencies
- The graph resolver provides an execution order

No additional configuration is required - the system works transparently with existing workflows. 