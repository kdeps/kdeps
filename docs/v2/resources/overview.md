# Resources overview

A resource is a single step in a workflow. It has an ID, optional dependencies, optional validation, and exactly one action. kdeps builds a dependency graph from all resources and runs them in order.

## Where it runs

All resource types work in both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-loop-mode). In workflow mode, resources execute as DAG steps ordered by `requires:`. In agent mode, whole workflows are registered as callable tools - the LLM invokes a workflow as a unit, and all resource dependencies inside it resolve correctly.

## Resource structure

```yaml
# resources/my-resource.yaml
actionId: myResource        # required: unique ID -- used by requires: and get()
name: My Resource           # required: human-readable label
description: What it does   # optional
category: api               # optional grouping label

requires:                   # like imports -- these run first and must produce output
  - otherResource           # myResource will not run until otherResource is done

items:                      # optional: run this resource once per item in the list
  - item1
  - item2

# Restrict which requests trigger this resource (optional)
validations:
  methods: [POST]                # only run on POST requests
  routes: [/api/v1/endpoint]    # only run on this route
  headers: [Authorization]      # only run when this header is present
  params: [q, limit]            # only run when these params are present
  skip:
    - get('skip') == true       # skip if this expression is true
  check:
    - get('q') != ''            # fail with error below if false
  error:
    code: 400
    message: Query required

# Expressions that run before/after the action
before:                 # runs before the action; use to prepare values
  - set('pre', 'value')
after:                  # runs after the action; use to process output
  - set('post', 'value')

# Exactly one primary action per resource (apiResponse: may accompany it
# on the same resource to format the HTTP response):
chat: { ... }        # send a prompt to an LLM; reply text at .message.content
httpClient: { ... }  # make an HTTP request; output is the parsed response body
sql: { ... }         # run a SQL query; output is the row set
python: { ... }      # run a Python script; output is its stdout (parsed as JSON)
exec: { ... }        # run a shell command; output is its stdout
email: { ... }       # send SMTP email or read/search/modify IMAP messages
telephony: { ... }   # in-call action (say, ask, menu, ...); output is TwiML
botReply: { ... }    # reply to the bot platform that delivered the message
file: { ... }       # filesystem operations: read, write, patch, list, delete
git: { ... }        # version control: status, diff, log, commit, push, pull
codeIntelligence: { ... }  # code navigation: search, definitions, diagnostics
agent: { ... }       # run another agent's full workflow; output is its apiResponse
apiResponse: { ... } # build the HTTP response returned to the caller
component:           # call an installable registry component
  name: botreply
  with:
    platform: telegram
    message: "Hello!"
```

Detailed reference for each action:
[`agent`](/resources/delegation/agent) ·
[`apiResponse`](/resources/api-response) ·
[`botReply`](/resources/messaging/bot-reply) ·
[`browser`](/resources/web/browser) ·
[`chat`](/resources/llm/) ·
[`codeIntelligence`](/resources/code-intelligence/navigation) ·
[`component`](/resources/delegation/component) ·
[`email`](/resources/messaging/email) ·
[`embedding`](/resources/rag/embedding) ·
[`exec`](/resources/scripting/exec) ·
[`file`](/resources/files/file) ·
[`git`](/resources/files/git) ·
[`httpClient`](/resources/web/http-client) ·
[`loader`](/resources/rag/loader) ·
[`ocr`](/resources/media/ocr) ·
[`python`](/resources/scripting/python) ·
[`scraper`](/resources/web/scraper) ·
[`searchLocal`](/resources/search/searchlocal) ·
[`searchWeb`](/resources/search/searchweb) ·
[`sql`](/resources/sql) ·
[`telephony`](/resources/messaging/telephony) ·
[`transcribe`](/resources/media/transcribe) ·
[`vectorStore`](/resources/rag/vector-store)

## Resource types

All executors are compiled into the `kdeps` binary and require no installation.
They are grouped here by function; each links to its own reference page.

### AI & language

| YAML key | Description | Page |
|---|---|---|
| `chat` | LLM interaction - responses, generation, tools, vision | [LLM](/resources/llm/) |
| `chat` (routing) | Delegate model choice to config / auto-fit | [LLM routing](/resources/llm/routing) |
| - | Model backends, providers, API keys | [LLM backends](/resources/llm/backends) |
| `loader` | Load PDF, HTML, CSV, text, or a directory into text chunks | [Loader](/resources/rag/loader) |
| `embedding` | Local SQLite keyword store: index / search / upsert / delete | [Embedding](/resources/rag/embedding) |
| `vectorStore` | External vector DB: Qdrant, Chroma, Pinecone, pgvector, ... | [Vector store](/resources/rag/vector-store) |
| `transcribe` | Speech to text via Whisper (OpenAI, Groq, local, offline) | [Transcribe](/resources/media/transcribe) |
| `ocr` | Text from an image via tesseract - local, no API key | [OCR](/resources/media/ocr) |

### Web

| YAML key | Description | Page |
|---|---|---|
| `httpClient` | HTTP requests - APIs, webhooks, auth, retry, cache | [HTTP client](/resources/web/http-client) |
| `scraper` | Fetch a URL and extract text, optional CSS selector | [Scraper](/resources/web/scraper) |
| `browser` | Playwright browser - navigation, forms, JS, screenshots | [Browser](/resources/web/browser) |
| `searchLocal` | Glob + keyword search across local files | [searchLocal](/resources/search/searchlocal) |
| `searchWeb` | Web search: DuckDuckGo (default), Brave, Bing, Tavily | [searchWeb](/resources/search/searchweb) |

### Data & system

| YAML key | Description | Page |
|---|---|---|
| `sql` | Database queries and transactions | [SQL](/resources/sql) |
| `file` | Read, write, patch, list, delete, copy, move files | [File](/resources/files/file) |
| `git` | Status, diff, log, commit, branch, push, pull | [Git](/resources/files/git) |
| `python` | Run a Python script, stdout parsed as JSON | [Python](/resources/scripting/python) |
| `exec` | Run a shell command, stdout captured | [Exec](/resources/scripting/exec) |
| `codeIntelligence` | Symbol search, definitions, references, folder graph | [Code intelligence](/resources/code-intelligence/navigation) · [folder graph](/resources/code-intelligence/graph) |

### Messaging

| YAML key | Description | Page |
|---|---|---|
| `email` | SMTP send, IMAP read / search / modify | [Email](/resources/messaging/email) |
| `telephony` | Voice call handling (say, ask, menu, dial, record) | [Telephony](/resources/messaging/telephony) |
| `botReply` | Reply to the chat platform that delivered the message | [Bot reply](/resources/messaging/bot-reply) |

### Orchestration

| YAML key | Description | Page |
|---|---|---|
| `agent` | Call another agent in an [agency](/reference/glossary#agency) | [Agent](/resources/delegation/agent) |
| `component` | Call a reusable resource bundle | [Component](/resources/delegation/component) |
| `apiResponse` | Return data to the HTTP caller | [API response](/resources/api-response) |

### Registry components (installable via `kdeps registry install`)

| Install name | Description |
|-------------|-------------|
| `scraper` | Extended content extraction: PDFs, .docx, .xlsx, images (type auto-detected) |
| `browser` | Playwright browser with stealth mode, persistent sessions, and file upload |
| `botreply` | Chat bot reply (Discord, Slack, Telegram, WhatsApp) |
| `embedding` | Vector embeddings via OpenAI Embeddings API |
| `search` | Web search via Tavily API |

See the [Components guide](/concepts/components) for installation and usage details.

## actionId and requires

[`actionId`](/reference/glossary#actionid) is the resource's unique name. It has two purposes: it controls which resource [`targetActionId`](/reference/glossary#targetactionid) points to, and it is the key you pass to `get()` to read a resource's output.

```yaml
# resources/llm.yaml
actionId: llm
name: LLM Chat
chat:
  prompt: "{{ get('q') }}"
```

```yaml
# resources/response.yaml
actionId: response
name: API Response
requires: [llm]          # response will not run until llm is done
apiResponse:
  response:
    answer: get('llm').message.content   # reply text from the llm resource
```

`requires:` lists direct dependencies only. kdeps resolves transitive dependencies automatically - you do not need to list the entire chain.

## Validation

[`validations`](/reference/glossary#validations) gates whether a resource runs at all. It fires before the action - failing fast means no LLM call, no HTTP call, no wasted work.

```yaml
# resources/example.yaml
validations:
  methods: [POST]          # skip unless the request method matches
  routes: [/api/v1/chat]  # skip unless the route matches
  headers: [Authorization] # skip unless this header is present
  params: [q]              # skip unless this query/body param is present

  skip:
    - get('mode') == 'fast'  # skip entirely when true (no error, just no-op)

  check:
    - get('q') != ''         # must be true or the request is rejected
    - get('limit') <= 100
  error:
    code: 400
    message: "q is required and limit must be <= 100"
```

[`skip`](/reference/glossary#skip) silently no-ops the resource. [`check`](/reference/glossary#check) returns an error to the caller. Both take a list - any one true condition is enough to trigger the behavior.

## Before and after expressions

`before:` runs before the action; use it to compute values the action reads.
`after:` runs after the action; use it to process output for downstream resources.

<div v-pre>

```yaml
# resources/example.yaml
before:
  - set('full_name', get('first') + ' ' + get('last'))
chat:
  prompt: "Hello {{ get('full_name') }}"   # reads the value set above
after:
  - set('summary', get('myResourceId'))    # store output under a new key
  - set('ts', info('timestamp'))
```

</div>

See [Expressions](/concepts/expressions) for detailed documentation.

## Items iteration

Process multiple items in sequence:

<div v-pre>

```yaml
# resources/example.yaml
items:
  - "First item"
  - "Second item"
  - "Third item"

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

## Loop iteration

Repeat a resource body while a condition is true (Turing-complete while-loop). Add `every:` to pause between iterations for a ticker pattern, or `at:` to fire at specific dates/times:

<div v-pre>

```yaml
# resources/example.yaml
loop:
  while: "loop.index() < 5"
  maxIterations: 1000   # safety cap (default: 1000)
  every: "1s"           # optional: wait 1 second between iterations
after:
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

## Resource output

Each resource produces output that can be accessed by dependent resources:

<div v-pre>

```yaml
# LLM resource output
actionId: llmResource
chat:
  prompt: "Answer: {{ get('q') }}"

# Access in another resource
requires: [llmResource]
apiResponse:
  response:
    answer: get('llmResource').message.content  # the reply text
```

</div>

## Execution flow

```d2
direction: down

A: Request {shape: oval}
B: Route Matching
C: Build Dep Graph

loop: "For each resource (in order)" {
  D1: Check Route
  D1S: skip {shape: oval}
  D2: Check Skip
  D2S: skip silently {shape: oval}
  D3: Preflight Check
  D3E: error {shape: oval}
  D4: "execute before:"
  D5: Execute Action
  D6: "execute after:"
  D7: Store Output

  D1 -> D1S: not matching
  D1 -> D2
  D2 -> D2S: condition true
  D2 -> D3
  D3 -> D3E: validation fails
  D3 -> D4 -> D5 -> D6 -> D7
}

E: Return Target
F: Response {shape: oval}

A -> B -> C -> loop -> E -> F
```

## Best practices

### 1. Use descriptive actionIds
```yaml
# Good
actionId: fetchUserProfile
actionId: validatePayment

# Avoid
actionId: resource1
actionId: r2
```

### 2. Single responsibility
Each resource should do one thing well. Split complex logic into multiple resources.

### 3. Validate early
Use `validations.check` to validate inputs before expensive operations.

### 4. Handle dependencies
Only list direct dependencies in [`requires`](/reference/glossary#requires). kdeps handles transitive dependencies.

### 5. Use appropriate timeouts
Set realistic `timeout` values based on expected execution time.

## See also

- [LLM resource](/resources/llm/) - AI model integration
- [HTTP client](/resources/web/http-client) - external API calls
- [SQL resource](/resources/sql) - database operations
- [Python resource](/resources/scripting/python) - script execution
- [Exec resource](/resources/scripting/exec) - shell commands
- [Email resource](/resources/messaging/email) - SMTP send, IMAP read/search/modify
- [API response](api-response) - response formatting
- [Agency & multi-agent](/concepts/agency) - multi-agent orchestration
- [Components](/concepts/components) - installable capability extensions
