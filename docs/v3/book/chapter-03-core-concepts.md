# Chapter 3: Core Concepts

Before going deep on any individual feature, it helps to have a mental model of how kdeps works as a whole. This chapter covers the five ideas that everything else builds on: resources, the DAG, the data store, the two operating modes, and backends.

## Resources: The Unit of Work

A **resource** is a single step in a workflow. Each resource lives in its own YAML file under `resources/`, has a unique `actionId`, and does exactly one thing: makes an LLM call, runs a SQL query, executes a shell command, fetches a URL, runs a Python script, or builds an HTTP response.

Resources are declarative. You describe what the resource should do; kdeps handles the execution. You never write code that calls the next step explicitly — dependencies are declared, not called.

The anatomy of a resource:

```yaml
# resources/example.yaml
actionId: myResource         # unique ID — this is how other resources reference this one
name: My Resource            # human-readable label for logs and tooling
description: What it does    # optional, surfaced in agent mode tool descriptions

requires:                    # resources that must complete before this one runs
  - otherResource

validations:                 # gate conditions — resource is skipped or errors here
  methods: [POST]
  routes: [/api/v1/chat]
  check:
    - get('input') != ''
  error:
    code: 400
    message: "input is required"

before:                      # expressions that run before the action
  - set('normalized', lower(trim(get('input'))))

chat:                        # the action — exactly one per resource
  model: llama3.2:1b
  prompt: "&#123;&#123; get('normalized') &#125;&#125;"

after:                       # expressions that run after the action
  - set('uppercased', upper(get('myResource')))
```

Every resource follows this structure. The action field (here, `chat:`) varies — it can be `sql:`, `httpClient:`, `python:`, `exec:`, `email:`, `telephony:`, `botReply:`, `browser:`, `scraper:`, `searchWeb:`, `searchLocal:`, `embedding:`, `file:`, `git:`, `codeIntelligence:`, `loader:`, `vectorStore:`, `transcribe:`, `agent:`, `component:`, or `apiResponse:`. Everything else — `requires:`, `validations:`, `before:`, `after:` — is common to all resource types.

## The DAG: Dependency-Ordered Execution

When kdeps loads a workflow, it reads all resource files in `resources/` and builds a directed acyclic graph (DAG) from the `requires:` declarations.

```
resources/
├── fetch.yaml    (actionId: fetch)
├── parse.yaml    (actionId: parse,   requires: [fetch])
├── enrich.yaml   (actionId: enrich,  requires: [fetch])
├── combine.yaml  (actionId: combine, requires: [parse, enrich])
└── respond.yaml  (actionId: respond, requires: [combine])
```

kdeps resolves this into an execution plan:

```
fetch
 ├── parse ─┐
 └── enrich─┤
            ▼
          combine
            │
            ▼
          respond
```

`parse` and `enrich` both depend on `fetch`, so they can run concurrently after `fetch` completes. `combine` waits for both. `respond` waits for `combine`.

You never write scheduling code. You declare dependencies. kdeps handles the rest.

**targetActionId** in `workflow.yaml` tells kdeps which resource is the terminal node — the one whose output becomes the HTTP response. Only the resources reachable from `targetActionId` through `requires:` are executed. Resources with no path to `targetActionId` are not run. This is how you include unused resources in a directory without causing side effects.

## The Data Store: get() and set()

Resources communicate through a per-request key-value store. After a resource executes, its output is automatically stored under a key equal to its `actionId`. You read it with `get('actionId')`.

```yaml
# Resource with actionId: llm produces output
chat:
  prompt: "Summarize this."

# A later resource reads that output
apiResponse:
  response:
    summary: get('llm')   # reads output of actionId: llm
```

You can also write to the store explicitly using `set()` in `before:` or `after:` blocks:

```yaml
before:
  - set('query', lower(trim(get('q'))))  # normalize before the action

after:
  - set('word_count', len(split(get('myResource'), ' ')))  # compute after
```

The store is scoped to a single request. Nothing leaks between requests unless you explicitly use the session store (covered in Chapter 15). This makes the system stateless by default, which is what you want for most HTTP APIs.

## Workflow Mode vs. Agent Mode

kdeps has two runtime modes. The key insight is that these are execution environments, not different workflow types. The same `workflow.yaml` runs in both.

**Workflow mode** (`kdeps run`) is deterministic. Incoming HTTP requests trigger the DAG. Resources execute in dependency order. Validations fire. Outputs are predictable. This is the mode you use when you want a reliable, auditable, testable pipeline — which is most of the time.

When `apiServer` is configured, workflow routes require `KDEPS_API_AUTH_TOKEN` (Chapter 2). Pass `Authorization: Bearer $KDEPS_API_AUTH_TOKEN` on curl calls to `/api/*` routes. `/health` is exempt. `/_kdeps/*` management routes use `KDEPS_MANAGEMENT_TOKEN` instead.

**Agent mode** (`kdeps [path]`) puts the LLM in the driver's seat. Each workflow is registered as a callable tool. The LLM decides which tools to call, in what order, based on the user's prompt. The full workflow DAG still executes when a tool is invoked — the LLM sees tools, not individual resources.

The distinction matters for system design:

| Concern | Workflow mode | Agent mode |
|---|---|---|
| Control flow | Declared in `requires:` | LLM decides |
| Input | HTTP request body | User prompt |
| Reproducibility | Same input → same path | Non-deterministic |
| Use case | Production pipelines | Interactive assistants, autonomous agents |
| Testing | Straightforward | Harder — LLM behavior varies |

Most production systems use workflow mode for their core pipelines and agent mode as the orchestration layer above — the LLM decides which pipeline to call, but once called, the pipeline runs deterministically.

## Backends: Separating Model from Execution

kdeps separates two concerns that are often conflated: **which model to call** and **where to call it**.

**Which model** is set per resource in the `chat:` block:

```yaml
chat:
  model: llama3.2:1b
  prompt: "&#123;&#123; get('q') &#125;&#125;"
```

**Where to call it** is set in `~/.kdeps/config.yaml`. With no configuration
at all, the model runs as a local llamafile — a self-contained binary that
kdeps downloads once and serves itself. Switching to a cloud provider is one
line:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai          # default is "file" (local llamafile)
  openai_api_key: sk-...
```

Your workflow files do not change. The model name in the resource file is passed to whatever provider you configure.

Supported backends:
- **Llamafile** — local inference, the default: no API key, no server install, works offline
- **Ollama** — local inference via the Ollama server, opt-in
- **OpenAI** — GPT-4o, GPT-4, GPT-3.5
- **Anthropic** — Claude 3 family
- **Groq** — fast inference for Llama, Mixtral
- **Any OpenAI-compatible endpoint** — Together AI, Perplexity, local vLLM, etc.

Chapter 6 covers backend configuration in full, including per-resource model overrides and credential management.

## Expressions: The Glue

Expressions are how you pass data between resources, validate inputs, and compute values. They appear in two forms:

**String interpolation** — embed a kdeps expression in any string value using `&#123;&#123; &#125;&#125;`:

```yaml
prompt: "Translate the following to French: &#123;&#123; get('text') &#125;&#125;"
url: "https://api.example.com/users/&#123;&#123; get('userId') &#125;&#125;"
```

**Statement blocks** — bare expressions in `before:`, `after:`, `validations.check`, and `validations.skip`:

```yaml
before:
  - set('query', lower(trim(get('q'))))
  - set('is_valid', len(get('query')) > 0)

validations:
  check:
    - get('is_valid')
```

The expression engine is [expr-lang](https://expr-lang.org/) with kdeps-specific helpers on top. Chapter 11 covers the full expression language, but you can go a long way with just `get()`, `set()`, `lower()`, `trim()`, `len()`, and basic comparison operators.

## Putting It Together

Here is the same two-resource workflow from Chapter 2, but now annotated through the lens of these five concepts:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent              # backend-independent identity
  targetActionId: response    # terminal node of the DAG
```

```yaml
# resources/llm.yaml
actionId: llm                 # key in the data store after execution

validations:                  # guard: fail fast before calling the model
  check:
    - get('q') != ''
  error:
    code: 400
    message: "'q' is required"

chat:                         # action: LLM call on whatever backend is configured
  model: llama3.2:1b
  prompt: "&#123;&#123; get('q') &#125;&#125;"   # expression: interpolate from data store
```

```yaml
# resources/response.yaml
actionId: response            # matches targetActionId — this is the terminal node

requires: [llm]               # DAG edge: don't run until 'llm' has output

apiResponse:                  # terminal action: builds the HTTP response
  success: true
  response:
    answer: get('llm')        # expression: reads from data store
```

This small workflow encapsulates all five core concepts. Every workflow you build, no matter how complex, is an extension of this pattern.

In the next two chapters, we will go deep on each operating mode.

X> ## Exercise
X>
X> Given this partial workflow, answer the questions below without running the code:
X>
X> ```yaml
X> # resources/fetch.yaml
X> actionId: fetch
X> httpClient:
X>   url: "https://api.example.com/data"
X>   method: GET
X>
X> # resources/transform.yaml
X> actionId: transform
X> requires: [fetch]
X> python:
X>   script: "..."
X>
X> # resources/summarize.yaml
X> actionId: summarize
X> requires: [fetch]
X> chat:
X>   prompt: "Summarize: &#123;&#123; get('fetch') &#125;&#125;"
X>
X> # resources/respond.yaml
X> actionId: respond
X> requires: [transform, summarize]
X> apiResponse:
X>   success: true
X>   response:
X>     summary: get('summarize')
X>     processed: get('transform')
X> ```
X>
X> 1. Draw the DAG. Which resources can run in parallel?
X> 2. `transform` and `summarize` both require `fetch`. What does kdeps guarantee about the order of their execution relative to each other?
X> 3. If `transform` fails, does `summarize` still run? Does `respond` run?
X> 4. Where is the result of the `fetch` resource stored, and how do `transform` and `summarize` access it?
X> 5. What is the minimum number of sequential "rounds" needed to execute this DAG?
X>
X> **Stretch goal:** Add a fifth resource `cache` that requires `[summarize]` and writes the summary to a session key. Draw the updated DAG and identify whether any parallelism is lost.
