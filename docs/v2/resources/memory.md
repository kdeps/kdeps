# Memory Resource

The `memory` resource gives agents **semantic, persistent experience storage** across invocations. Agents consolidate what they have learned, recall relevant past context by meaning (not exact match), and forget stale or incorrect entries — all backed by a local SQLite vector index.

It uses the same multi-backend embedding infrastructure as the [`embedding`](./embedding) resource, so any embedding provider that works for RAG also works for memory.

It can be used as a primary resource or as an [inline resource](../concepts/inline-resources) inside `before` / `after` blocks.

## Basic Usage

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: rememberFact
  name: Remember Fact

run:
  memory:
    operation: consolidate
    content: "The user prefers concise responses without examples."
    category: user-preferences
```

---

## Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `operation` | string | `consolidate` | Operation: `consolidate`, `recall`, or `forget`. |
| `content` | string | — | **Required.** Text to store (consolidate) or query (recall/forget). Supports [expressions](../concepts/expressions). |
| `category` | string | `memories` | Logical bucket for memories (one SQLite table per category). |
| `topK` | int | `5` | Max entries returned by `recall`. |
| `dbPath` | string | `/tmp/kdeps-memory/<category>.db` | Explicit path to the SQLite DB file. |
| `model` | string | `nomic-embed-text` | Embedding model name. |
| `backend` | string | `ollama` | Embedding provider: `ollama`, `openai`, `cohere`, `huggingface`. |
| `baseUrl` | string | backend default | Override the provider base URL. |
| `apiKey` | string | — | API key for cloud providers (OpenAI, Cohere, HuggingFace). |
| `metadata` | object | — | Key-value metadata stored alongside each memory entry. |
| `timeoutDuration` | string | `60s` | Embedding API call timeout (e.g. `30s`, `2m`). |

---

## Operations

### `consolidate`

Embeds `content` and stores the vector plus original text in the category. A `consolidated_at` timestamp is added automatically to the metadata.

```yaml
run:
  memory:
    operation: consolidate
    content: "Completed onboarding for user alice@example.com on 2026-04-01."
    category: agent-history
    metadata:
      user: alice
      event: onboarding
```

**Returns:**

```json
{
  "success": true,
  "operation": "consolidate",
  "id": 7,
  "category": "agent-history",
  "dimensions": 768
}
```

### `recall`

Embeds `content` (the query) and returns the `topK` most semantically similar memories sorted by cosine similarity (highest first).

```yaml
run:
  memory:
    operation: recall
    content: "What do I know about alice?"
    category: agent-history
    topK: 5
```

**Returns:**

```json
{
  "operation": "recall",
  "category": "agent-history",
  "count": 2,
  "memories": [
    {
      "id": 7,
      "content": "Completed onboarding for user alice@example.com on 2026-04-01.",
      "similarity": 0.94,
      "metadata": {
        "user": "alice",
        "event": "onboarding",
        "consolidated_at": "2026-04-01T10:00:00Z"
      }
    }
  ]
}
```

Access memories in downstream resources:

<div v-pre>

```yaml
metadata:
  requires: [recallContext]
run:
  chat:
    model: llama3
    prompt: |
      Past context:
      {{ output('recallContext').memories | map(.content) | join('\n') }}

      Current question: {{ output('body') }}
```

</div>

### `forget`

Removes memories from the category.

- If `content` is non-empty, deletes entries whose stored text exactly matches `content`.
- If `content` is empty, deletes **all** entries in the category.

```yaml
run:
  memory:
    operation: forget
    content: "Completed onboarding for user alice@example.com on 2026-04-01."
    category: agent-history
```

**Returns:**

```json
{
  "success": true,
  "operation": "forget",
  "category": "agent-history"
}
```

---

## Backends

Memory uses the same embedding backends as the `embedding` resource. All four providers are supported:

### Ollama (local, default)

```yaml
run:
  memory:
    operation: consolidate
    content: "{{ output('llmResponse') }}"
    category: facts
    model: nomic-embed-text
    backend: ollama
    baseUrl: http://localhost:11434   # optional
```

Recommended local embedding models: `nomic-embed-text` (768-dim), `mxbai-embed-large` (1024-dim), `all-minilm` (384-dim).

### OpenAI

<div v-pre>

```yaml
run:
  memory:
    operation: recall
    content: "{{ output('userQuery') }}"
    category: knowledge
    model: text-embedding-3-small
    backend: openai
    apiKey: "{{ env('OPENAI_API_KEY') }}"
    topK: 10
```

</div>

### Cohere

<div v-pre>

```yaml
run:
  memory:
    operation: consolidate
    content: "{{ output('summary') }}"
    category: summaries
    model: embed-english-v3.0
    backend: cohere
    apiKey: "{{ env('COHERE_API_KEY') }}"
```

</div>

### HuggingFace

<div v-pre>

```yaml
run:
  memory:
    operation: recall
    content: "recent user requests"
    category: requests
    model: sentence-transformers/all-MiniLM-L6-v2
    backend: huggingface
    apiKey: "{{ env('HF_API_KEY') }}"
```

</div>

---

## Storage

Memories are stored in SQLite (one file per category):

- **Default location:** `/tmp/kdeps-memory/<category>.db`
- **Custom location:** set `dbPath` to any writable path

For production use, set `dbPath` to a persistent location outside `/tmp`:

```yaml
run:
  memory:
    operation: consolidate
    content: "{{ output('body') }}"
    category: production-facts
    dbPath: /var/lib/kdeps/memory/production-facts.db
```

Each category table has this schema:

| Column | Type | Description |
|--------|------|-------------|
| `id` | INTEGER | Auto-increment primary key |
| `text` | TEXT | Original content text |
| `embedding` | TEXT | JSON-serialized float64 vector |
| `metadata` | TEXT | JSON-serialized metadata (always includes `consolidated_at`) |
| `created_at` | DATETIME | Row creation timestamp |

---

## Using Expressions in `content`

The `content` field supports [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  memory:
    operation: consolidate
    content: "{{ output('llmResponse') }}"
    category: agent-responses
    metadata:
      model: "{{ output('llmResponse').model }}"
      route: "{{ route() }}"
```

</div>

---

## Memory as an Inline Resource

Memory can run **before** or **after** the main resource action — useful for injecting recalled context without an extra resource file:

<div v-pre>

```yaml
run:
  # Recall relevant past context before the LLM call
  before:
    - memory:
        operation: recall
        content: "{{ output('userQuery') }}"
        category: knowledge
        topK: 3

  chat:
    model: llama3
    prompt: |
      Context from memory:
      {{ inlineOutput('memory').memories | map(.content) | join('\n') }}

      Question: {{ output('userQuery') }}

  # Store the LLM response after the chat
  after:
    - memory:
        operation: consolidate
        content: "{{ output('chatResponse') }}"
        category: responses
```

</div>

---

## Full Example: Agent with Persistent Memory

A two-endpoint agent: `/learn` stores new facts, `/ask` recalls context and answers using an LLM.

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

  memory:
    operation: consolidate
    content: "{{ output('body').fact }}"
    category: agent-knowledge
    model: nomic-embed-text
    dbPath: /var/lib/kdeps/memory/agent.db

  apiResponse:
    success: true
    response:
      stored: true
      id: "{{ output('storeFact').id }}"
```

```yaml
# resources/recall.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: recallFacts
  name: Recall Facts

run:
  validations:
    routes: [/ask]
    methods: [POST]

  memory:
    operation: recall
    content: "{{ output('body').question }}"
    category: agent-knowledge
    topK: 5
    model: nomic-embed-text
    dbPath: /var/lib/kdeps/memory/agent.db
```

```yaml
# resources/respond.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: respond
  name: Answer Question
  requires: [recallFacts]

run:
  validations:
    routes: [/ask]
    methods: [POST]

  chat:
    model: llama3
    prompt: |
      Answer using only the context below. If the context is empty, say you don't know.

      Context:
      {{ output('recallFacts').memories | map(.content) | join('\n---\n') }}

      Question: {{ output('body').question }}

  apiResponse:
    success: true
    response:
      answer: "{{ output('respond') }}"
      context_used: "{{ output('recallFacts').count }}"
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
  -d '{"question": "What is the capital of France?"}'
```

---

## Memory vs Embedding

Both resources use the same SQLite + cosine-similarity infrastructure. The distinction is conceptual:

| | `embedding` | `memory` |
|-|-------------|---------|
| Primary use | RAG — index/search documents | Agent state — store/recall experiences |
| Default category field | `collection` | `category` |
| Default DB path | `/tmp/kdeps-embedding/` | `/tmp/kdeps-memory/` |
| Metadata | user-supplied | user-supplied + automatic `consolidated_at` |
| Operations | `index`, `search`, `delete` | `consolidate`, `recall`, `forget` |

Use `embedding` for static document corpora. Use `memory` for dynamic agent experience that accumulates and evolves at runtime.
