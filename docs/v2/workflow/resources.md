# Resources overview

A resource is a single step in a workflow. It has an ID, optional dependencies, optional validation, and exactly one action. kdeps builds a dependency graph from all resources and runs them in order.

## Where it runs

All resource types work in both [workflow mode](/workflow/) and [agent mode](/agent/). In workflow mode, resources execute as DAG steps ordered by `requires:`. In agent mode, whole workflows are registered as callable tools - the LLM invokes a workflow as a unit, and all resource dependencies inside it resolve correctly.

## The shape of a resource

```yaml
# resources/my-resource.yaml
actionId: myResource        # required: unique ID -- used by requires: and get()
name: My Resource           # required: human-readable label
description: What it does   # optional
category: api               # optional grouping label

requires:                   # like imports -- these run first and must produce output
  - otherResource           # myResource will not run until otherResource is done

items:                      # optional: run this resource once per item -- see /workflow/items
  - item1
  - item2

loop:                       # optional: repeat while a condition holds -- see /workflow/loop
  while: "loop.index() < 5"

# Gate whether the resource runs at all -- see /workflow/validation
validations:
  methods: [POST]                # only run on POST requests
  routes: [/api/v1/endpoint]    # only run on this route
  headers: [Authorization]      # only run when this header is present
  params: [q, limit]            # only run when these params are present
  skip:
    - get('skip') == true       # skip silently if true
  check:
    - get('q') != ''            # fail with the error below if false
  error:
    code: 400
    message: Query required

# Expressions that run around the action -- see /workflow/expressions
before:                 # prepare values the action reads
  - set('pre', 'value')
after:                  # process the output for downstream resources
  - set('post', 'value')
```

## The one action

```yaml
# resources/my-resource.yaml
actionId: myResource        # required: unique ID -- used by requires: and get()

# Exactly one primary action per resource (apiResponse: may accompany it
# on the same resource to format the HTTP response):
chat: { ... }        # send a prompt to an LLM; reply text at .message.content
httpClient: { ... }  # make an HTTP request; output is the parsed response body
python: { ... }      # run a Python script; output is its stdout (parsed as JSON)
exec: { ... }        # run a shell command; output is its stdout
codeIntelligence: { ... }  # code navigation: search, definitions, diagnostics
agent: { ... }       # run another agent's full workflow; output is its apiResponse
apiResponse: { ... } # build the HTTP response returned to the caller
component:           # call an installable registry component
  name: botreply
  with:
    platform: telegram
    message: "Hello!"
```

## actionId and requires

[`actionId`](/reference/glossary#actionid) is the resource's unique name. It is what [`targetActionId`](/reference/glossary#targetactionid) points to, and the key you pass to `get()` to read the resource's output.

```yaml
# resources/response.yaml
actionId: response
name: API Response
requires: [llm]          # response will not run until llm is done
apiResponse:
  response:
    answer: get('llm').message.content   # reply text from the llm resource
```

`requires:` lists direct dependencies only. kdeps resolves transitive dependencies automatically - you do not list the whole chain.

## Resource types

All executors are compiled into the `kdeps` binary and require no installation. They are grouped here by function; each links to its own reference page.

## AI & language

| YAML key | Description | Page |
|---|---|---|
| `chat` | LLM interaction - responses, generation, tools, vision | [LLM](/workflow/resources/llm) |
| `chat` (routing) | Delegate model choice to config / auto-fit | [LLM routing](/workflow/resources/llm-routing) |
| - | Model backends, providers, API keys | [LLM backends](/workflow/resources/llm-backends) |
| `loader` | Load PDF, HTML, CSV, text, or a directory into text chunks | [Loader](/workflow/resources/loader) |
| `embedding` | Local SQLite keyword store: index / search / upsert / delete | [Embedding](/workflow/resources/embedding) |
| `vectorStore` | External vector DB: Qdrant, Chroma, Pinecone, pgvector, ... | [Vector store](/workflow/resources/vector-store) |
| `component: transcribe` | Speech to text via Whisper (OpenAI, Groq, local, offline) - a kdeps component, YAML only | [Transcribe](/workflow/resources/transcribe) |
| `component: ocr` | Text from an image via tesseract - local, no API key (a kdeps component, YAML only) | [OCR](/workflow/resources/ocr) |

## Web

| YAML key | Description | Page |
|---|---|---|
| `httpClient` | HTTP requests - APIs, webhooks, auth, retry, cache | [HTTP client](/workflow/resources/http-client) |
| `component: scraper` | Fetch a URL and extract text, optional CSS selector - a kdeps component, YAML only | [Scraper](/workflow/resources/scraper) |
| `component: browser` | Playwright browser - navigation, forms, JS, screenshots - a kdeps component, YAML only | [Browser](/workflow/resources/browser) |
| `searchLocal` | Glob + keyword search across local files | [searchLocal](/workflow/resources/search-local) |
| `component: searchWeb` | Web search: DuckDuckGo (default), Brave, Bing, Tavily - a kdeps component, YAML only | [searchWeb](/workflow/resources/search-web) |

## Data & system

| YAML key | Description | Page |
|---|---|---|
| `component: sql` | Database queries and transactions - a kdeps component, YAML only | [SQL](/workflow/resources/sql) |
| `component: file` | Read, write, patch, list, delete, copy, move files - a kdeps component, YAML only | [File](/workflow/resources/file) |
| `component: git` | Status, diff, log, commit, branch, push, pull - a kdeps component, YAML only | [Git](/workflow/resources/git) |
| `python` | Run a Python script, stdout parsed as JSON | [Python](/workflow/resources/python) |
| `exec` | Run a shell command, stdout captured | [Exec](/workflow/resources/exec) |
| `codeIntelligence` | Symbol search, definitions, references, folder graph | [Code intelligence](/workflow/resources/code-navigation) · [folder graph](/workflow/resources/code-graph) |

## Messaging

| YAML key | Description | Page |
|---|---|---|
| `component: email` | SMTP send, IMAP read / search / modify - a kdeps component, YAML only | [Email](/workflow/resources/email) |
| `component: twiml` | Voice call TwiML (say, gather, dial, record) - a kdeps component, YAML only | [Telephony](/workflow/resources/telephony) |
| `apiResponse` (bot mode) | The workflow result is sent to the chat platform that delivered the message | [Bot replies](/workflow/resources/bot-reply) |

## Orchestration

| YAML key | Description | Page |
|---|---|---|
| `agent` | Call another agent in an [agency](/reference/glossary#agency) | [Agent](/agencies/agent-resource) |
| `component` | Call a reusable resource bundle | [Component](/agencies/component-resource) |
| `apiResponse` | Return data to the HTTP caller | [API response](/workflow/resources/api-response) |

## Registry components (installable via `kdeps install`)

Some install names (`embedding`) also exist as native YAML keys; the registry component is a separate package. An installed `browser` registry component takes the place of the shipped `browser` component.

| Install name | Description |
|-------------|-------------|
| `browser` | Playwright browser with stealth mode, persistent sessions, and file upload |
| `botreply` | Chat bot reply (Discord, Slack, Telegram, WhatsApp) |
| `embedding` | Vector embeddings via OpenAI Embeddings API |
| `search` | Web search via Tavily API |

See the [Components guide](/agencies/components) for installation and usage details.

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

## See also

- [Validation and control flow](/workflow/validation) - the `validations:` block
- [Expressions](/workflow/expressions) - `before:`/`after:`, `get()`, `set()`
- [Items iteration](/workflow/items) and [While-loop](/workflow/loop)
- [Error handling (onError)](/workflow/error-handling) - retry and fallback
- [Agencies](/agencies/) - multi-agent orchestration
- [Components](/agencies/components) - installable capability extensions
