# Memory Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps registry install memory`
>
> Usage: `run: { component: { name: memory, with: { action: "store", key: "...", value: "..." } } }`

The Memory component provides **persistent key-value storage** across invocations, backed by a local SQLite database.

> **Note**: The component supports `store` and `retrieve` operations. For semantic/vector memory (cosine similarity recall), use the [Embedding component](embedding) with a SQL resource for storage and retrieval.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `action` | string | no | `store` | Operation: `store` or `retrieve` |
| `key` | string | yes | — | Key identifier for the memory entry |
| `value` | string | no | — | Value to store (required when `action: store`) |
| `dbPath` | string | no | `~/.kdeps/memory.db` | Path to the SQLite database file |

## Using the Memory Component

**Store a fact:**

```yaml
run:
  component:
    name: memory
    with:
      action: store
      key: "user-preference-theme"
      value: "dark"
```

**Retrieve a fact:**

```yaml
run:
  component:
    name: memory
    with:
      action: retrieve
      key: "user-preference-theme"
```

Access the result via `output('<callerActionId>')`.

---

## Result Map

**Store:**

| Key | Type | Description |
|-----|------|-------------|
| `success` | bool | `true` when the value was stored. |
| `key` | string | The key that was stored. |

**Retrieve:**

| Key | Type | Description |
|-----|------|-------------|
| `success` | bool | `true` when the key was found. |
| `key` | string | The key that was retrieved. |
| `value` | string | The stored value, or empty if not found. |

---

## Expression Support

All fields support [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: memory
    with:
      action: store
      key: "session-{{ get('session_id') }}-preference"
      value: "{{ get('user_preference') }}"
      dbPath: /var/lib/kdeps/memory.db
```

</div>

---

## Full Example: Agent with Persistent Memory

A two-endpoint agent: `/learn` stores facts, `/ask` retrieves context and answers using an LLM.

```yaml
# workflow.yaml
settings:
  name: memory-agent
  targetActionId: respond
  apiServerMode: true
  apiServer:
    routes:
      - path: /learn
        methods: [POST]
      - path: /ask
        methods: [POST]
```

<div v-pre>

```yaml
# resources/learn.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: storeFact
  name: Store Fact

run:
  validations:
    routes: [/learn]
    methods: [POST]

  component:
    name: memory
    with:
      action: store
      key: "fact-{{ info('timestamp') }}"
      value: "{{ get('body').fact }}"
      dbPath: /var/lib/kdeps/memory/agent.db

  apiResponse:
    success: true
    response:
      stored: true
      key: "{{ output('storeFact').key }}"
```

```yaml
# resources/recall.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: recallFact
  name: Recall Fact

run:
  validations:
    routes: [/ask]
    methods: [POST]

  component:
    name: memory
    with:
      action: retrieve
      key: "{{ get('body').key }}"
      dbPath: /var/lib/kdeps/memory/agent.db
```

```yaml
# resources/respond.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: respond
  name: Answer Question
  requires: [recallFact]

run:
  validations:
    routes: [/ask]
    methods: [POST]

  chat:
    model: llama3
    prompt: |
      Answer using the context below. If context is empty, say you don't know.

      Context: {{ output('recallFact').value }}

      Question: {{ get('body').question }}

  apiResponse:
    success: true
    response:
      answer: "{{ output('respond') }}"
```

</div>

**Run:**

```bash
kdeps run workflow.yaml

# Store a fact
curl -X POST http://localhost:16394/learn \
  -H 'Content-Type: application/json' \
  -d '{"fact": "The capital of France is Paris."}'

# Ask a question
curl -X POST http://localhost:16394/ask \
  -H 'Content-Type: application/json' \
  -d '{"key": "fact-...", "question": "What is the capital of France?"}'
```
