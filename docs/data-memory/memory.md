---
outline: deep
---

# Memory Operations

Memory operations store, retrieve, and clear key-value pairs in a persistent store, useful for state or data across executions. Session operations do the same but are temporary, lasting only for a single request (e.g., an API call or process).

Both provide `getRecord`, `setRecord`, `deleteRecord`, and `clear` to manage string-based key-value pairs.

## Memory Operation Functions

These functions manage persistent data.

### `memory.getRecord(id: String): String`

Gets a memory record by its ID.

- **Parameters**:
  - `id`: The record’s key.
- **Returns**: The record’s value or an empty string if not found.

#### Example

```apl
local context = "@(memory.getRecord("task_123"))"
```

Gets `task_123` value, evaluated when accessed due to late binding.

### `memory.setRecord(id: String, value: String): String`

Stores or updates a memory record.

- **Parameters**:
  - `id`: The record’s key.
  - `value`: The value to store.
- **Returns**: The stored value.

#### Example

```apl
expr {
  "@(memory.setRecord("task_123", "done"))"
}
```

Stores `"done"` for `task_123`. Uses `expr` for side-effect.

### `memory.deleteRecord(id: String): String`

Deletes a memory record.

- **Parameters**:
  - `id`: The record’s key.
- **Returns**: The deleted value.

### `memory.clear(): String`

Clears all memory records.

- **Returns**: Confirmation message.

#### Example

```apl
clear()
```

Resets memory store.

## Session Operation Functions

Session operations manage temporary data, scoped to a single request and cleared afterward. They mirror memory operations but are not persistent.

### `session.getRecord(id: String): String`

Gets a session record by its ID.

- **Parameters**:
  - `id`: The record’s key.
- **Returns**: The record’s value or an empty string if not found.

#### Example

```apl
local temp = "@(session.getRecord("req_789"))"
```

Gets `req_789` value, available only during the request.

### `session.setRecord(id: String, value: String): String`

Stores a session record for the current request.

- **Parameters**:
  - `id`: The record’s key.
  - `value`: The value to store.
- **Returns**: The stored value.

#### Example

```apl
expr {
  "@(session.setRecord("req_789", "temp_data"))"
}
```

Stores `"temp_data"` for `req_789`, discarded after the request.

### `session.deleteRecord(id: String): String`

Deletes a session record.

- **Parameters**:
  - `id`: The record’s key.
- **Returns**: The deleted value.

### `session.clear(): String`

Clears all session records for the current request.

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
- **API**: Both use `getRecord`, `setRecord`, `deleteRecord`, `clear` with identical signatures.

#### Example

```apl
expr {
  // Persistent task data
  "@(memory.setRecord("task_123", "done"))"
  // Temporary request data
  "@(session.setRecord("req_789", "temp"))"
}
local task = "@(memory.getRecord("task_123"))" // "done"
local temp = "@(session.getRecord("req_789"))" // "temp" (this request only)
```

## Notes

- Both operate on string key-value pairs.
- Use `memory` for data that lasts, `session` for request-only data.
- `clear` in either removes all records, so use carefully.
- Operations are synchronous, returning immediately.
