# memory

> Persistent key-value memory store backed by SQLite.

The memory component stores and retrieves named string values across workflow runs. Use it to remember conversation state, cache intermediate results, or share data between separate workflow executions.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `action` | string | no | `store` | Operation: `store`, `retrieve`, or `forget` |
| `key` | string | yes | — | Key to store, retrieve, or delete |
| `value` | string | no | — | Value to store (required when `action` is `store`) |
| `dbPath` | string | no | `~/.kdeps/memory.db` | Path to the SQLite database file |

## Actions

| Action | Description |
|--------|-------------|
| `store` | Write `value` under `key`. Overwrites any existing value. |
| `retrieve` | Read the value stored under `key`. Returns `""` if not found. |
| `forget` | Delete the entry for `key`. |

## Outputs

Accessed via `output('<your-action-id>')`:

```json
{ "result": "<value or status>", "action": "store", "key": "mykey" }
```

| Field | Type | Description |
|-------|------|-------------|
| `result` | string | The retrieved value (`retrieve`), `"stored"` (`store`), or `"forgotten"` (`forget`) |
| `action` | string | The action that was performed |
| `key` | string | The key that was used |

## Usage

Store a value:

```yaml
run:
  component:
    name: memory
    with:
      action: store
      key: lastQuery
      value: "{{ input('query') }}"
  apiResponse:
    success: true
    response:
      status: "{{ output('storeStep').result }}"
```

Retrieve a value:

```yaml
run:
  component:
    name: memory
    with:
      action: retrieve
      key: lastQuery
  apiResponse:
    success: true
    response:
      lastQuery: "{{ output('retrieveStep').result }}"
```

Forget a value:

```yaml
run:
  component:
    name: memory
    with:
      action: forget
      key: lastQuery
```

Chaining store and retrieve in a conversation bot:

```yaml
# Step 1: store the user's message
- actionId: rememberMessage
  run:
    component:
      name: memory
      with:
        action: store
        key: "user_{{ session('id') }}"
        value: "{{ input('message') }}"

# Step 2: retrieve it in the LLM context
- actionId: chatWithContext
  run:
    chat:
      prompt: |
        Previous message: {{ output('rememberMessage').result }}
        New message: {{ input('message') }}
```

## Requirements

- Python stdlib only (`sqlite3`, `os`, `json`) — no additional packages needed.
- Write access to the directory containing `dbPath`.

## Notes

- The default `dbPath` is `~/.kdeps/memory.db` — data persists across all workflow runs on the same machine.
- For isolated per-workflow storage, set `dbPath` to a workflow-specific location (e.g. `/tmp/mywf-memory.db`).
- All values are stored as strings. Serialize complex types with JSON before storing.
- The `forget` action is idempotent — it succeeds even if the key does not exist.
