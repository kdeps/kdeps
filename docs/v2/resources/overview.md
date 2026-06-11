# Resources Overview

A resource is a single step in a workflow. It has an ID, optional dependencies, optional validation, and exactly one action. kdeps builds a dependency graph from all resources and runs them in order.

## Where it runs

All resource types work in both [workflow mode](/modes/workflow-mode) and [agent mode](/modes/agent-mode). In workflow mode, resources execute as DAG steps ordered by `requires:`. In agent mode, whole workflows are registered as callable tools -- the LLM invokes a workflow as a unit, and all resource dependencies inside it resolve correctly.

## Resource Structure

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
agent: { ... }       # run another agent's full workflow; output is its apiResponse
apiResponse: { ... } # build the HTTP response returned to the caller
component:           # call an installable registry component
  name: botreply
  with:
    platform: telegram
    message: "Hello!"
```

## Resource Types

### Native executors (always available)

These executors are compiled into the `kdeps` binary and require no installation.

| YAML key | Description | Use Case |
|----------|-------------|----------|
| `chat` | LLM interaction | AI responses, text generation |
| `httpClient` | HTTP requests | External APIs, webhooks |
| `sql` | Database queries | Data retrieval, updates |
| `python` | Python scripts | Data processing, ML |
| `exec` | Shell commands | System operations |
| `scraper` | Web scraping | Fetch URL, optional CSS selector |
| `embedding` | Keyword store | SQLite index/search/upsert/delete |
| `searchLocal` | File search | Glob + keyword search across local files |
| `searchWeb` | Web search | DuckDuckGo (default), Brave, Bing, Tavily |
| `browser` | Browser automation | Playwright-based navigation, screenshots, JS eval |
| `email` | Email send/receive | SMTP send, IMAP read/search/modify |
| `telephony` | Voice call handling | TwiML actions (say, ask, menu, dial, record) for Twilio-compatible providers |
| `botReply` | Bot platform reply | Send a text reply to the platform that delivered the current message |
| `agent` | Inter-agent delegation | Call another agent in an [agency](/reference/glossary#agency) |
| `apiResponse` | API response | Return data to the HTTP caller |

### Registry components (installable via `kdeps registry install`)

| Install name | Description |
|-------------|-------------|
| `scraper` | Extended content extraction: PDFs, .docx, .xlsx, images (type auto-detected) |
| `browser` | Playwright browser with stealth mode, persistent sessions, and file upload |
| `botreply` | Chat bot reply (Discord, Slack, Telegram, WhatsApp) |
| `embedding` | Vector embeddings via OpenAI Embeddings API |
| `search` | Web search via Tavily API |

See the [Components guide](../concepts/components) for installation and usage details.

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

`requires:` lists direct dependencies only. kdeps resolves transitive dependencies automatically -- you do not need to list the entire chain.

## Validation

[`validations`](/reference/glossary#validations) gates whether a resource runs at all. It fires before the action -- failing fast means no LLM call, no HTTP call, no wasted work.

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

[`skip`](/reference/glossary#skip) silently no-ops the resource. [`check`](/reference/glossary#check) returns an error to the caller. Both take a list -- any one true condition is enough to trigger the behavior.

## before and after expressions

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

## Items Iteration

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

## Loop Iteration

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

## Resource Output

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

## Execution Flow

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
Only list direct dependencies in [`requires`](/reference/glossary#requires). KDeps handles transitive dependencies.

### 5. Use Appropriate Timeouts
Set realistic `timeout` values based on expected execution time.

## See Also

- [LLM Resource](llm) -- AI model integration
- [HTTP Client](http-client) -- external API calls
- [SQL Resource](sql) -- database operations
- [Python Resource](python) -- script execution
- [Exec Resource](exec) -- shell commands
- [Email Resource](email) -- SMTP send, IMAP read/search/modify
- [API Response](api-response) -- response formatting
- [Agency & Multi-Agent](../concepts/agency) -- multi-agent orchestration
- [Components](../concepts/components) -- installable capability extensions
