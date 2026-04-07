# Resources Overview

Resources are the fundamental building blocks of KDeps workflows. Each resource performs a specific action and can depend on other resources.

## Resource Structure

Every resource follows this structure:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: myResource        # Unique identifier
  name: My Resource           # Human-readable name
  description: What it does   # Optional description
  category: api               # Optional: for organization
  requires:                   # Dependencies
    - otherResource

items:                        # Optional: for iteration
  - item1
  - item2

run:
  # Request restrictions and validation
  validations:
    methods: [POST]
    routes: [/api/v1/endpoint]
    headers: [Authorization]
    params: [q, limit]
    skip:
    - get('skip') == true
    check:
      - get('q') != ''
    error:
      code: 400
      message: Query required

  # Processing expressions
  exprBefore:                 # Runs BEFORE the main action
    - set('pre', 'value')
  expr:                       # Runs AFTER the main action (default)
    - set('post', 'value')
  exprAfter:                  # Alias for expr
    - set('also_post', 'value')

  # Action (only one per resource)
  chat: { ... }        # LLM chat (core)
  httpClient: { ... }  # HTTP request (core)
  sql: { ... }         # Database query (core)
  python: { ... }      # Python script (core)
  exec: { ... }        # Shell command (core)
  agent: { ... }       # Call another agent — agency mode (core)
  apiResponse: { ... } # API response (core)
  component:           # Installable component (e.g. scraper, tts, pdf…)
    name: scraper
    with:
      url: "https://example.com"
      selector: ".article"
```

## Resource Types

### Core executors (built into the binary)

| Type | Description | Use Case |
|------|-------------|----------|
| `chat` | LLM interaction | AI responses, text generation |
| `httpClient` | HTTP requests | External APIs, webhooks |
| `sql` | Database queries | Data retrieval, updates |
| `python` | Python scripts | Data processing, ML |
| `exec` | Shell commands | System operations |
| `agent` | Inter-agent delegation | Multi-agent agencies |
| `apiResponse` | Response formatting | Final output |

### Components (installable via `kdeps component install`)

| Type | Install name | Description |
|------|-------------|----------|
| `component: { name: scraper }` | `scraper` | Content extraction from web pages, PDFs, documents, images |
| `component: { name: tts }` | `tts` | Text-to-Speech synthesis |
| `component: { name: pdf }` | `pdf` | PDF generation from HTML or Markdown |
| `component: { name: calendar }` | `calendar` | ICS calendar read/write |
| `component: { name: search }` | `search` | Web or local filesystem search |
| `component: { name: botreply }` | `botreply` | Chat bot reply (Discord, Slack, Telegram, WhatsApp) |
| `component: { name: embedding }` | `embedding` | Vector embeddings & semantic search |
| `component: { name: browser }` | `browser` | Browser automation (Playwright) |
| `component: { name: remoteagent }` | `remoteagent` | Federated agent invocation (UAF) |
| `component: { name: memory }` | `memory` | Persistent semantic memory store |
| `component: { name: email }` | `email` | Email send/read/search (SMTP/IMAP) |
| `component: { name: autopilot }` | `autopilot` | Goal-directed workflow synthesis |

See the [Components guide](../concepts/components) for installation and usage details.

## Metadata

### actionId (Required)
Unique identifier for the resource. Used to reference output from other resources.

```yaml
metadata:
  actionId: llmResource

# Access output in another resource:
data: get('llmResource')
```

### description (Optional)
Human-readable description of what the resource does.

```yaml
metadata:
  actionId: llmResource
  name: LLM Chat
  description: Processes user queries using language models
```

### category (Optional)
Organize resources by category for better management.

```yaml
metadata:
  actionId: userAuth
  name: User Authentication
  category: auth

metadata:
  actionId: dataProcessor
  name: Data Processor
  category: processing
```

Common categories: `api`, `auth`, `processing`, `storage`, `ai`, `utils`.

### requires (Dependencies)
List of resources that must execute before this one.

```yaml
metadata:
  actionId: responseResource
  requires:
    - llmResource
    - httpResource
```

KDeps automatically builds a dependency graph and executes resources in the correct order.

## Request Restrictions

### validations.methods
Limit which HTTP methods trigger this resource:

```yaml
run:
  validations:
    methods: [GET, POST]
```

### validations.routes
Limit which routes trigger this resource:

```yaml
run:
  validations:
    routes: [/api/v1/chat, /api/v1/query]
```

### validations.headers / validations.params
Whitelist specific headers or parameters:

```yaml
run:
  validations:
    headers:
      - Authorization
      - X-API-Key
    params:
      - q
      - limit
      - offset
```

## Validation

### validations.skip
Skip resource execution based on conditions:

```yaml
run:
  validations:
    skip:
      - get('skip') == true
      - get('mode') == 'fast'
```

If any condition evaluates to `true`, the resource is skipped.

### validations.check
Validate inputs before execution:

```yaml
run:
  validations:
    check:
      - get('q') != ''
      - get('limit') <= 100
    error:
      code: 400
      message: Invalid request parameters
```

If validation fails, the error response is returned immediately.

## Processing with Expressions

Execute logic before or after the main action:

### exprBefore
Runs **before** the main action. Use this to prepare data used in the resource's own configuration (like prompts or URLs).

<div v-pre>

```yaml
run:
  exprBefore:
    - set('full_name', get('first') + ' ' + get('last'))
  chat:
    prompt: "Hello {{ get('full_name') }}"
```

</div>

### expr (or exprAfter)
Runs **after** the main action. Use this to process results or update state for subsequent resources.

<div v-pre>

```yaml
run:
  chat:
    prompt: "Summary of {{ get('q') }}"
  expr:
    - set('summary', get('myResourceId'))
    - set('processed_at', info('timestamp'))
```

</div>

See [Expressions](../concepts/expressions.md) for detailed documentation.

## Items Iteration

Process multiple items in sequence:

<div v-pre>

```yaml
items:
  - "First item"
  - "Second item"
  - "Third item"

run:
  chat:
    prompt: "Process: {{ get('current') }}"
```

</div>

Access iteration context:
- `get('current')` - Current item
- `get('prev')` - Previous item
- `get('next')` - Next item
- `get('index')` - Current index (0-based)
- `get('count')` - Total item count

## Loop Iteration

Repeat a resource body while a condition is true (Turing-complete while-loop). Add `every:` to pause between iterations for a ticker pattern, or `at:` to fire at specific dates/times:

<div v-pre>

```yaml
run:
  loop:
    while: "loop.index() < 5"
    maxIterations: 1000   # safety cap (default: 1000)
    every: "1s"           # optional: wait 1 second between iterations
  expr:
    - "{{ set('result', loop.count()) }}"
  apiResponse:
    success: true
    response:
      count: "{{ get('result') }}"
```

</div>

Access loop context:
- `loop.index()` - Current index (0-based)
- `loop.count()` - Current count (1-based)
- `loop.results()` - Results from all prior iterations

Loop fields:
- `while` - Boolean expression; loop runs while truthy
- `maxIterations` - Safety cap (default: 1000)
- `every` - Optional inter-iteration delay (`"500ms"`, `"1s"`, `"2m"`, `"1h"`). Mutually exclusive with `at`
- `at` - Optional array of specific dates/times (RFC3339, `"HH:MM"`, or `"YYYY-MM-DD"`). Mutually exclusive with `every`

When `apiResponse` is present, each iteration produces one streaming response map.

## Resource Output

Each resource produces output that can be accessed by dependent resources:

<div v-pre>

```yaml
# LLM resource output
metadata:
  actionId: llmResource
run:
  chat:
    model: llama3.2:1b
    prompt: "Answer: {{ get('q') }}"

# Access in another resource
metadata:
  requires: [llmResource]
run:
  apiResponse:
    response:
      answer: get('llmResource')  # Get the LLM response
```

</div>

## Execution Flow

```
Request
    ↓
┌─────────────────┐
│ Route Matching  │
└────────┬────────┘
         ↓
┌─────────────────┐
│ Build Dep Graph │
└────────┬────────┘
         ↓
For each resource (in order):
    ┌─────────────────┐
    │ Check Route     │ → Skip if not matching
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Check Skip      │ → Skip if condition true
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Preflight Check │ → Error if validation fails
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Execute exprBefore │
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Execute Action  │
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Execute expr    │
    └────────┬────────┘
             ↓
    ┌─────────────────┐
    │ Store Output    │
    └─────────────────┘
         ↓
┌─────────────────┐
│ Return Target   │
└─────────────────┘
```

## Best Practices

### 1. Use Descriptive actionIds
```yaml
# Good
actionId: fetchUserProfile
actionId: validatePayment

# Avoid
actionId: resource1
actionId: r2
```

### 2. Single Responsibility
Each resource should do one thing well. Split complex logic into multiple resources.

### 3. Validate Early
Use `validations.check` to validate inputs before expensive operations.

### 4. Handle Dependencies
Only list direct dependencies in `requires`. KDeps handles transitive dependencies.

### 5. Use Appropriate Timeouts
Set realistic `timeoutDuration` values based on expected execution time.

## Next Steps

- [LLM Resource](llm) - AI model integration
- [HTTP Client](http-client) - External API calls
- [SQL Resource](sql) - Database operations
- [Python Resource](python) - Script execution
- [Exec Resource](exec) - Shell commands
- [API Response](api-response) - Response formatting
- [Agency & Multi-Agent](../concepts/agency) - Multi-agent orchestration
- [Components](../concepts/components) - Installable capability extensions (scraper, tts, pdf, email, and more)
