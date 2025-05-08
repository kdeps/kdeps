---
outline: deep
---

# Memory Operations

Memory operations provide a way to store, retrieve, and clear key-value pairs in a persistent memory store. These
operations are useful for managing state or caching data across different executions or sessions.

The memory operations include `getItem`, `setItem`, and `clear`, which allow you to interact with the memory store
efficiently.

## Memory Operation Functions

Below are the available memory operation functions, their purposes, and how to use them.

### `getItem(id: String): String`

Retrieves the textual content of a memory item by its identifier.

- **Parameters**:
  - `id`: The identifier of the memory item.
- **Returns**: The textual content of the memory entry, or an empty string if not found.

#### Example: Retrieving a Stored Value

```apl
local taskContext = "@(memory.getItem("task_123_context"))"
```

In this example, the `getItem` function retrieves the `task_123_context` record item.
> **Note:** Because Apple PKL uses late binding, the taskContext expression won’t be evaluated until it is actually
> accessed—for example, when included in a response output or passed into an LLM prompt.

### `setItem(id: String, value: String): String`

Sets or updates a memory item with a new value.

- **Parameters**:
  - `id`: The identifier of the memory item.
  - `value`: The value to store.
- **Returns**: The set value as confirmation.

#### Example: Storing a Value

```apl
expr {
  taskId = "task_123"
  result = "completed_successfully"
  "@(memory.setItem(taskId, result))"
}
```

In this example, `memory.setItem` stores the value `"completed_successfully"` under the key `"task_123"` in memory.

We use the `exp`r block because `setItem` is a side-effecting function—it performs an action but doesn't return a
meaningful value. That is why it's placed inside an `expr` block: to ensure the expression is evaluated for its effect
rather than for a result that would otherwise be ignored.

### `clear(): String`

Clears all memory items in the store.

- **Returns**: A confirmation message.

#### Example: Resetting All Stored Data

```apl
clear()
```

This example clears all memory items, resetting the memory store to an empty state. A confirmation message is returned.

## Notes

- The `getItem` and `setItem` functions operate on string-based key-value pairs.
- The `clear` function removes all stored items, so use it cautiously to avoid unintended data loss.
- Memory operations are synchronous and return immediately with the result or confirmation.
