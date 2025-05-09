---
outline: deep
---

# Memory Operations

Memory operations store, retrieve, and clear key-value pairs in a persistent store, useful for state or data across executions. Session operations do the same but are temporary, lasting only for a single request (e.g., an API call or process).

Both provide `getItem`, `setItem`, `deleteItem`, and `clear` to manage string-based key-value pairs.

## Memory Operation Functions

These functions manage persistent data.

### `memory.getItem(id: String): String`

Gets a memory item by its ID.

- **Parameters**:
  - `id`: The item’s key.
- **Returns**: The item’s value or an empty string if not found.

#### Example

```apl
local context = "@(memory.getItem("task_123"))"
```

Gets `task_123` value, evaluated when accessed due to late binding.

### `memory.setItem(id: String, value: String): String`

Stores or updates a memory item.

- **Parameters**:
  - `id`: The item’s key.
  - `value`: The value to store.
- **Returns**: The stored value.

#### Example

```apl
expr {
  "@(memory.setItem("task_123", "done"))"
}
```

Stores `"done"` for `task_123`. Uses `expr` for side-effect.

### `memory.deleteItem(id: String): String`

Deletes a memory item.

- **Parameters**:
  - `id`: The item’s key.
- **Returns**: The deleted value.

### `memory.clear(): String`

Clears all memory items.

- **Returns**: Confirmation message.

#### Example

```apl
clear()
```

Resets memory store.

## Session Operation Functions

Session operations manage temporary data, scoped to a single request and cleared afterward. They mirror memory operations but are not persistent.

### `session.getItem(id: String): String`

Gets a session item by its ID.

- **Parameters**:
  - `id`: The item’s key.
- **Returns**: The item’s value or an empty string if not found.

#### Example

```apl
local temp = "@(session.getItem("req_789"))"
```

Gets `req_789` value, available only during the request.

### `session.setItem(id: String, value: String): String`

Stores a session item for the current request.

- **Parameters**:
  - `id`: The item’s key.
  - `value`: The value to store.
- **Returns**: The stored value.

#### Example

```apl
expr {
  "@(session.setItem("req_789", "temp_data"))"
}
```

Stores `"temp_data"` for `req_789`, discarded after the request.

### `session.deleteItem(id: String): String`

Deletes a session item.

- **Parameters**:
  - `id`: The item’s key.
- **Returns**: The deleted value.

### `session.clear(): String`

Clears all session items for the current request.

- **Returns**: Confirmation message.

#### Example

```apl
session.clear()
```

Clears session data for the request.

## Memory vs. Session

- **Persistence**:
  - **Memory**: Persistent across requests or sessions (e.g., task state).
  - **Session**: Temporary, cleared after the request (e.g., request-specific data).
- **Use Cases**:
  - **Memory**: Task results, cached data (e.g., `task_123`).
  - **Session**: Temporary flags, request context (e.g., `req_789`).
- **API**: Both use `getItem`, `setItem`, `deleteItem`, `clear` with identical signatures.

#### Example

```apl
expr {
  // Persistent task data
  "@(memory.setItem("task_123", "done"))"
  // Temporary request data
  "@(session.setItem("req_789", "temp"))"
}
local task = "@(memory.getItem("task_123"))" // "done"
local temp = "@(session.getItem("req_789"))" // "temp" (this request only)
```

## Notes

- Both operate on string key-value pairs.
- Use `memory` for data that lasts, `session` for request-only data.
- `clear` in either removes all items, so use carefully.
- Operations are synchronous, returning immediately.
