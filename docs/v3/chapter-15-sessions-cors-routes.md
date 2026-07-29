# Chapter 15: Sessions, CORS, and Route Restrictions

Three configuration features handle stateful behavior, browser security, and traffic filtering: the session store, CORS settings, and route/method restrictions on resources.

## Sessions

By default, kdeps resources are stateless. Each request is processed independently. The `session:` configuration adds a cross-request key-value store, so values set during one request can be read in subsequent requests from the same caller.

### When to Use Sessions

- Multi-turn chatbots where context from previous turns matters
- Workflows where step 1 gathers data and step 2 (a separate request) uses it
- Rate limiting or per-user state tracking
- Shopping cart, wizard, or other multi-step UX flows

### Configuration

```yaml
# workflow.yaml
settings:
  session:
    type: sqlite                    # "sqlite" or "memory"
    path: "/data/sessions.db"       # SQLite file path (sqlite only)
    ttl: "30m"                      # session expires after 30 min of inactivity
    cleanupInterval: "5m"           # how often to purge expired sessions
```

Two storage backends:

| Backend | Persistence | Use case |
|---|---|---|
| `sqlite` | File-based; survives restarts | Production, multi-container (shared volume) |
| `memory` | In-process; lost on restart | Development, single-instance tests |

### Using Sessions in Resources

Use the third argument to `set()` to write to session scope:

```yaml
# Write to session scope
before:
  - set('history', get('history') + [get('message')], 'session')
  - set('turn_count', int(get('turn_count') or '0') + 1, 'session')
```

```yaml
# Read from session scope — same get() call, sessions are in the same store
before:
  - set('previous_history', get('history'))   # reads from session if set there
  - set('turn', get('turn_count'))
```

Session values are partitioned by session ID. Two different callers with different session IDs cannot read each other's session data.

### Session ID

The session ID comes from the `X-Session-ID` request header or is auto-generated on first request and returned in the `X-Session-ID` response header.

Client-side flow:
```
POST /api/v1/chat {"message": "Hello"}
→ Response headers: X-Session-ID: abc123

POST /api/v1/chat {"message": "What did I just say?"}
  Headers: X-Session-ID: abc123
→ Response uses session state from previous request
```

### Multi-Turn Chatbot Example

```yaml
# resources/chat.yaml
actionId: chat

before:
  # Load history from session (empty array if no history yet)
  - set('history', get('history') or [])
  # Append new user message
  - set('history', get('history') + [{"role": "user", "content": get('message')}], 'session')

chat:
  model: llama3.2:1b
  messages: "&#123;&#123; get('history') &#125;&#125;"

after:
  # Append assistant response to history
  - set('history', get('history') + [{"role": "assistant", "content": get('chat')}], 'session')
```

Each request reads the accumulated conversation history, adds the new user message, calls the model with the full history, and stores the model's response back.

### Session TTL and Cleanup

Sessions expire after the configured `ttl` of inactivity. `cleanupInterval` controls how often expired sessions are removed from storage.

For a 30-minute TTL with 5-minute cleanup, expired sessions are removed within 5 minutes of expiry. This prevents unbounded growth of the session database.

For production, use a TTL that matches your UX contract. A support chatbot session might last 24 hours; a short-lived form wizard might need only 15 minutes.

## Persistent Memory

While sessions are scoped to a single caller and expire after inactivity, persistent memory is a cross-request, cross-session key-value store that survives restarts. Values written to persistent memory are available to every request, from every caller, until explicitly deleted.

### When to Use Persistent Memory

- Storing facts the agent learns about the project or user over time
- Caching expensive computations or API results across requests
- Maintaining configuration or state that should outlive individual sessions
- Building agents that accumulate knowledge across conversations

### How It Works

Persistent memory is backed by a JSONL file on disk. Each entry has a key, a value, and a timestamp. The agent loop automatically extracts facts from each LLM turn and saves them to memory — no explicit `set()` call required.

### Writing to Memory

Use `set()` with the `'memory'` scope:

```yaml
before:
  - set('project_language', 'Go', 'memory')
  - set('api_endpoint', 'https://api.example.com/v2', 'memory')
```

### Reading from Memory

`get()` checks persistent memory in its priority chain (after request-scoped memory, before session storage). No special syntax needed:

```yaml
before:
  - set('lang', get('project_language'))    # reads from persistent memory if set there
```

You can also read directly from the `'memory'` scope:

```yaml
before:
  - set('lang', get('project_language', 'memory'))
```

### Searching Memory

Use `memory_search` to find entries by content (case-insensitive substring match):

```yaml
before:
  - set('results', memory_search('project'))
```

Returns an array of matching entries: `[{"key": "project_language", "value": "Go", "timestamp": "..."}]`.

### Listing and Deleting

```yaml
before:
  - set('all', memory_list())              # list all keys
  - memory_delete('stale_key')             # delete a single entry
```

### Auto-Extraction

In agent mode, the agent loop automatically extracts facts from every turn and tool call without any explicit `memory_save` calls. Four extraction mechanisms run:

1. **Explicit markers** — `[MEMORY: key] value` on its own line in any response
2. **Action sentences** — first action sentence ("Added X to Y", "Fixed Z") captured as `last_action`
3. **File references** — edited/modified/created file paths captured as `last_files`
4. **Tool results** — every write/exec/search tool call output saved as `tool:<name>:<key>` entry

Read-only tools (`read_file`, `list_files`, `search_local`) are filtered out. Each tool type is capped at 20 entries — oldest auto-deleted. Auto-extracted low-signal types (`note`/`fact`) are globally capped at 50 combined, pruning oldest-first on write; structural entries (prompt/progress/result/status/decision/context/tool_result/...) are never pruned, so the workflow chain and resume point always survive. A value longer than its store limit is cut and marked with a trailing `...` so a model reads it as a fragment, not the complete fact; the cut backs off to the nearest character boundary so a multibyte character is never split.

Memory entries are auto-classified into types (`prompt`, `purpose`, `progress`, `tool_result`, `result`, `status`, `decision`, `preference`, `context`, `file`, `action`, `error`, `note`) based on their key pattern.

### Memory Graph

Entries are automatically linked into a type-based directed graph showing how facts relate:

```
prompt -> purpose -> progress -> tool_result -> result -> status
                                    |
                              action, error
                              file, decision
                              fact, note
```

The graph is inlined into the `<memory>` block on every LLM call — entries in causal order (parents before children), each on exactly one line (multiline values have newlines collapsed to ` / `), each showing its `<- parent` edge, with the current unfinished task flagged `<== RESUME` and its relative age in the orientation map (e.g. `resume: result:build (2m ago)`, coarsely `just now`/`Nm`/`Nh`/`Nd ago`) so a cold model can judge whether the resume point is fresh or stale. This lets the agent trace how facts relate and where to continue — seeing that a `tool_result` came from a specific `progress` entry, or that a `decision` was informed by a `result`. Under a token budget the block keeps the active task chain, entries relevant to the current prompt, and the newest unresolved `error` first, so a large memory never drops where you are, what you just asked about, or a failure you should not repeat; the orientation map names that error (e.g. `| error: error:migration`), and an error whose value reads as handled (`resolved`/`fixed`/`wontfix`/`cancelled`/...) is not surfaced, though one that reads as re-opened (`reopened`/`not fixed`/`still failing`) is surfaced even when the word `fixed` also appears. When two entries carry the same fact (case/whitespace-insensitive), the later copy is flagged `(same as <first-key>)` rather than repeated as independent evidence — nothing is dropped, so edges stay intact, and short common values like `done` are never flagged.

### Compaction Integration

When the agent's conversation is compacted (summarized and truncated), the structured summary is auto-captured into memory. The `## Key Decisions` and `## Critical Context` sections are extracted as individual entries. A `checkpoint:summary` entry preserves the full condensed Goal/Progress snapshot.

### Session Persistence

The agent's full LLM config (model, backend, base URL) is saved to `session:config` on startup and after every `/model` switch. On the next run, the config is restored automatically. Working directory is saved on start (`session:started`) and resume (`session:resumed`).

### Configuration

Persistent memory is configured under `settings:` in `workflow.yaml`:

```yaml
settings:
  memory:
    path: "/data/memory.jsonl"            # persistent memory file path
    maxEntries: 10000                     # max entries before compaction triggers
    compactionThreshold: 5000              # compact when entries exceed this count
    compactionTarget: 2000                 # target entry count after compaction
```

| Field | Default | Description |
|---|---|---|
| `path` | `memory.jsonl` in the working directory | File path for the persistent memory store |
| `maxEntries` | `10000` | Maximum entries before compaction triggers |
| `compactionThreshold` | `5000` | Entry count that triggers compaction |
| `compactionTarget` | `2000` | Target entry count after compaction completes |

If `path` is not set, memory is in-memory only and lost on restart. For production, always set a file path.

### Memory vs. Sessions

| Feature | Session | Persistent Memory |
|---|---|---|
| Scope | Single caller (by session ID) | All callers, all sessions |
| Persistence | TTL-based, auto-expires | Survives restarts, manual deletion |
| Storage | SQLite or in-memory | JSONL file |
| Auto-extraction | No | Yes (agent mode) |
| Search | By key only | By key or content |
| Graph | No | Yes (dependency tracking) |
| Use case | Multi-turn conversations | Long-term knowledge accumulation |

## CORS

Cross-Origin Resource Sharing controls which browser origins can call your API. This only affects browser-based callers — server-to-server calls are unaffected by CORS settings.

### Default Behavior

Without a `cors:` block, kdeps allows all origins with credentials enabled:

```
Access-Control-Allow-Origin: <echoes requesting origin>
Access-Control-Allow-Credentials: true
```

This is permissive by default — appropriate for development, but you should restrict it for production.

### Configuration

```yaml
settings:
  apiServer:
    cors:
      allowOrigins:
        - "https://app.example.com"
        - "https://admin.example.com"
      allowMethods:
        - GET
        - POST
        - OPTIONS
      allowHeaders:
        - Content-Type
        - Authorization
        - X-Session-ID
      exposeHeaders:
        - X-Request-ID
        - X-Session-ID
      allowCredentials: true
      maxAge: "24h"             # how long browsers cache the preflight response
```

### Configuration Fields

| Field | Default | Description |
|---|---|---|
| `allowOrigins` | `["*"]` | Allowed origins. Use `["*"]` for all (only for public APIs) |
| `allowMethods` | All common methods | HTTP methods allowed in cross-origin requests |
| `allowHeaders` | Common headers | Request headers the browser can send |
| `exposeHeaders` | none | Response headers the browser can read |
| `allowCredentials` | `true` | Allow cookies and auth headers in cross-origin requests |
| `maxAge` | `"12h"` | Duration to cache preflight response |

### CORS for Open-Origin APIs

If your API accepts requests from any origin (workflow routes still require `KDEPS_API_AUTH_TOKEN` when `apiServer` is configured):

```yaml
cors:
  allowOrigins: ["*"]
  allowCredentials: false    # must be false when allowOrigins is *
  allowMethods: [GET, POST]
  maxAge: "24h"
```

Note: `allowCredentials: true` with `allowOrigins: ["*"]` is not allowed by the CORS spec. kdeps handles the `"*"` case by echoing the request origin to support credentials — but explicitly setting `allowCredentials: false` is clearer for truly public APIs.

### CORS for Multi-Domain Internal Apps

```yaml
cors:
  allowOrigins:
    - "https://app.example.com"
    - "https://admin.example.com"
    - "http://localhost:3000"      # development
    - "http://localhost:5173"      # Vite dev server
  allowMethods: [GET, POST, PUT, DELETE, OPTIONS]
  allowHeaders:
    - Content-Type
    - Authorization
    - X-Session-ID
  exposeHeaders:
    - X-Request-ID
  allowCredentials: true
  maxAge: "1h"
```

## Route and Method Restrictions

`validations.routes` and `validations.methods` on individual resources act as execution filters. A resource with these set only activates when the incoming request matches.

### Method Restrictions

```yaml
# resources/create.yaml
actionId: createItem
validations:
  methods: [POST]
sql:
  query: "INSERT INTO items ..."

# resources/list.yaml
actionId: listItems
validations:
  methods: [GET]
sql:
  query: "SELECT * FROM items"
```

`createItem` only runs on POST requests. `listItems` only runs on GET requests. A GET to the same route skips `createItem` silently.

### Route Restrictions

```yaml
# resources/user-lookup.yaml
actionId: userLookup
validations:
  routes: [/api/v1/users]
sql:
  query: "SELECT * FROM users WHERE id = $1"
  params: [get('id')]

# resources/product-lookup.yaml
actionId: productLookup
validations:
  routes: [/api/v1/products]
sql:
  query: "SELECT * FROM products WHERE id = $1"
  params: [get('id')]
```

### Route Wildcard Patterns

Routes support `*` wildcards to match path segments:

| Pattern | Matches | Does Not Match |
|---|---|---|
| `/users` | `/users` | `/users/123` |
| `/users/*` | `/users/123`, `/users/abc` | `/users` |
| `/api/*` | `/api/v1`, `/api/users/123` | `/api` |
| `/api/v1/*` | `/api/v1/users`, `/api/v1/chat` | `/api/v2/users` |

Use wildcards when a resource should respond to a family of paths:

```yaml
# Handles GET /api/v1/users, /api/v1/users/123, /api/v1/users/active, etc.
validations:
  methods: [GET]
  routes: [/api/v1/users/*]
```

Exact paths (`/api/v1/users`) match only that literal path. Add `/*` to also cover sub-paths.

### Combined Route + Method

```yaml
validations:
  routes: [/api/v1/documents]
  methods: [POST]
```

Only activates on `POST /api/v1/documents`. Any other route or method skips this resource.

### Multi-Route Workflows

When a single workflow serves multiple routes, the `targetActionId` resource must return a response for every valid route combination. One pattern:

```yaml
# workflow.yaml
settings:
  apiServer:
    routes:
      - path: /api/v1/users
        methods: [GET, POST]
```

```yaml
# resources/handle-get.yaml
actionId: handleGet
validations:
  methods: [GET]
sql:
  query: "SELECT * FROM users LIMIT 50"

# resources/handle-post.yaml
actionId: handlePost
validations:
  methods: [POST]
sql:
  query: "INSERT INTO users (name) VALUES ($1)"
  params: [get('name')]

# resources/respond.yaml
actionId: respond
requires: [handleGet, handlePost]
apiResponse:
  success: true
  response:
    result: get('handleGet') or get('handlePost')
```

`respond` requires both branches. For a given request, one branch runs and the other is skipped. `get('handleGet') or get('handlePost')` reads from whichever one produced output.

This pattern keeps all routing logic in the resource layer. The `workflow.yaml` declares which paths are open; the resources declare which ones they participate in.

## Putting It Together: A Stateful API

A chat API with session-based history, CORS for a React frontend, and route filtering:

```yaml
# workflow.yaml
settings:
  apiServer:
    hostIp: "0.0.0.0"
    portNum: 8080
    routes:
      - path: /api/v1/chat
        methods: [POST]
      - path: /api/v1/history
        methods: [GET, DELETE]
    cors:
      allowOrigins:
        - "https://chat.example.com"
        - "http://localhost:3000"
      allowHeaders: [Content-Type, Authorization, X-Session-ID]
      exposeHeaders: [X-Session-ID]
      allowCredentials: true
  session:
    type: sqlite
    path: "/data/sessions.db"
    ttl: "2h"
    cleanupInterval: "15m"
```

Resources use `validations.routes` and `validations.methods` to handle each endpoint. Session storage persists conversation history. CORS allows the React frontend to make requests from the browser.

X> ## Exercise
X>
X> Build a multi-turn chatbot that remembers the last three messages in a conversation.
X>
X> 1. Configure a session store in `workflow.yaml` (SQLite backend, `ttl: 1h`).
X> 2. In a `before:` expression, read the session key `history` (default to an empty array if absent).
X> 3. Append the current user message to the history array. Keep only the last 3 entries.
X> 4. Pass the history array as context in the LLM prompt: `"Conversation so far:\n&#123;&#123; get('history') &#125;&#125;\n\nUser: &#123;&#123; get('q') &#125;&#125;"`.
X> 5. After the LLM responds, store the updated history (including the new message) back to the session.
X>
X> Test the memory across multiple requests using the same session cookie or a `session_id` header. Verify that after 4 requests, only the most recent 3 messages appear in the history.
X>
X> Configure CORS to allow requests from `http://localhost:3000` so a local frontend could call this endpoint.
X>
X> **Stretch goal:** Add a `/clear` route that a `DELETE /api/v1/chat` request can hit to wipe the session history, then verify a fresh conversation starts with no memory of the previous one.
