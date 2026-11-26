# Glossary

This page defines the terms of the kdeps docs, in alphabetical order.

| Term | Definition | More |
| :--- | :--- | :--- |
| <a id="actionid"></a>`actionId` | A unique string that identifies a resource within a workflow. Used as the target of `requires:` dependencies. | [Resources overview](/workflow/resources) |
| <a id="agency"></a>agency | Multiple agents composed into one system. One agent delegates a task to another via the `agent:` action type; the callee runs its full pipeline and returns its output. | [Agencies](/agencies/) |
| agent | A workflow whose `chat` has a `loop:` block: the model calls tools over many rounds and keeps one conversation per `loop.session` id. There is no `kind: Agent` - `kind:` is `Workflow`, `Component`, or `Agency`. | [Agent loop](/workflow/agent-loop/) |
| `apiResponse` | A resource action type that returns a structured JSON response to the client. Usually the terminal node of a workflow. | [API response](/workflow/resources/api-response) |
| `before` / `after` | Expression blocks that run before or after a resource's main action. Used for data preparation, normalization, and validation. Statements execute in order. | [Expression blocks](/reference/expr-blocks) |
| `chat` | A resource action type for LLM chat completion. Sends a prompt and returns the response. The most common action type. | [LLM resource](/workflow/resources/llm) |
| <a id="check"></a>`check` | A list of boolean expressions in `validations.check`. All must be true or the resource fails and the workflow stops. | [Validation and control flow](/workflow/validation) |
| <a id="component"></a>component | A reusable, packaged resource bundle. Installed from the registry or built in a `components/` directory. Declared as `kind: Component`. | [Components](/agencies/components) |
| <a id="deterministic"></a>deterministic | Same input, same output, every time. In kdeps this describes the workflow-mode pipeline - route matching, `requires:` ordering, `validations`, and response shaping all resolve the same way for a given request. It does not describe the LLM's text output. Contrast [probabilistic](#probabilistic). | [Deterministic by design](/reference/start/why-kdeps#deterministic-by-design) |
| `embedding` | The shipped component around the native embedding executor: a local SQLite keyword store (index, search, upsert, delete), an embedding backend (vectorize, embed_query), and Cohere rerank. Keyword matching is SQL `LIKE`, not vector similarity. | [Embedding](/workflow/resources/embedding) |
| <a id="eval"></a>eval | One test case in the `evals:` list of `workflow.yaml`. `kdeps test` sends its request to the workflow in the process and checks the assertions. | [Evals: test a workflow](/workflow/testing) |
| `exec` | The kdeps component that runs a shell command through the kdeps API and returns the structured result (`success`, `exitCode`, `stdout`, `stderr`, `command`, `result`, `timedOut`). | [exec](/workflow/resources/exec) |
| expr | Short for expression: a statement evaluated by the expr-lang engine. Used in `before:` / `after:` blocks, `validations`, and `{{ }}` string interpolation. | [Expressions](/workflow/expressions) |
| `file` | A kdeps component (`component: file`) for filesystem operations: read, write, patch, list, delete, mkdir, copy, move, append. | [File](/workflow/resources/file) |
| `file()` | An expression function that reads uploaded or local files. Takes a glob pattern and an optional selector (`first`, `last`, `count`, `all`, `mime:<type>`). | [Data access](/workflow/data-access) |
| `get()` | The primary data access function. Reads from query params, headers, body, session, memory, environment, or resource outputs, with auto-detection. | [Data access](/workflow/data-access) |
| `git` | A kdeps component (`component: git`) for version control: status, diff, log, commit, branch, push, pull. | [Git](/workflow/resources/git) |
| <a id="grader"></a>grader | A check attached to a judge with `graders:`. Kinds: `exact`, `contains`, `not_contains`, `regex`, `json`, `expr`. It runs on the answer in process. A failure rejects that judge before it calls the model. | [Judge panel](/workflow/agent-loop/judges) |
| `harness.output.json` | A boolean field on `chat` resources. When true, forces the LLM to return valid JSON with no markdown wrapping. | [LLM resource](/workflow/resources/llm) |
| `http_request` | A kdeps component (`component: http_request`) for HTTP requests. Supports all methods, headers, query params, body, retry, caching, and named connections. | [HTTP client](/workflow/resources/http-client) |
| `info()` | An expression function that returns request metadata: `ID`, `timestamp`, `path`, `method`, `IP`, `sessionId`, `filecount`, `files`, `filetypes`. | [Data access](/workflow/data-access) |
| <a id="input-source"></a>input source | How requests reach a workflow: `api` (HTTP), `bot`, `file`, or `interactive` (a terminal prompt where each line is one request). Set under `deploy.input.sources`. | [Input sources](/workflow/input-sources) |
| `items` | An array a resource iterates over. The resource runs once per item. Access the current item via `item.current()` or `get('current')`. | [Items iteration](/workflow/items) |
| <a id="judge"></a>judge | One reviewer in the judge panel. A chat turns the panel on with `judges:` or `judges: auto` under `guardrails`. It reviews the answer before the chat returns it. | [Judge panel](/workflow/agent-loop/judges) |
| <a id="kdeps-api"></a>kdeps API | The local control listener of `kdeps run`. It binds `127.0.0.1`, needs `KDEPS_API_TOKEN`, and serves the tool calls of components and the agent loop. It is not the workflow `apiServer`. | [kdeps API](/reference/api) |
| <a id="loop"></a>`loop` | A while-loop on a resource. The body runs while the `while` expression is true, up to `maxIterations`. `every:` turns it into a scheduled ticker. | [While-loop iteration](/workflow/loop) |
| memory (expression) | Request-scoped key/value storage. Set with `set('key', value, 'memory')`; read with `get('key', 'memory')`. Cleared when the request completes. | [Data access](/workflow/data-access) |
| memory (persistent) | Project-scoped storage for the agent loop, kept as markdown notes in the knowledge vault, that survives across sessions. Built-in tools: `memory_save`, `memory_search`, `memory_delete`, `memory_query`. The graph is injected into the prompt. Workflow resources can still call `memory_list()`. | [Persistent memory](/workflow/agent-loop/memory) |
| memory graph | A directed graph of memory entries linked by their `References` fields and auto-linked by type. Inlined into the `<memory>` block in causal order. | [Memory internals](/reference/agent-loop/memory-internals#memory-graph) |
| `output()` | An expression function that reads the output of a completed resource by its `actionId`. | [Data access](/workflow/data-access) |
| <a id="probabilistic"></a>probabilistic | The same prompt can produce different output on each call. Every language model - Claude, GPT, Gemini, Groq, Ollama, local llamafile / GGUF - is probabilistic. A kdeps `chat:` resource inherits this; the pipeline around it stays [deterministic](#deterministic). | [LLM provider reference](/reference/llm-server/providers) |
| `python` | The kdeps component that runs a Python script through the kdeps API and returns stdout parsed as JSON when possible. Supports inline scripts, file paths, packages, and virtualenvs. | [python](/workflow/resources/python) |
| <a id="requires"></a>`requires` | A list of `actionId`s that must complete before this resource runs. Defines the DAG edges. Transitive dependencies resolve automatically. | [Execution flow](/workflow/execution-flow) |
| resource | The fundamental building block of a workflow: an `actionId`, a primary action type, optional `requires`, and optional validations, loop, and error handling. | [Resources overview](/workflow/resources) |
| `scenario` | A list of fixed messages (`role`, `prompt`) under a chat's `context:`, sent before the prompt. Use it for a system message or few-shot examples. | [LLM resource](/workflow/resources/llm) |
| `search_local` | A kdeps component (`component: search_local`) that searches local files by keyword and glob, optionally with a TF-IDF index and graph boost. | [search_local](/workflow/resources/search-local) |
| session | HTTP session storage across requests from the same client. Set with `set('key', value, 'session')`. Not the agent loop conversation (`loop.session`). | [Session configuration](/workflow/sessions) |
| `set()` | Stores a value in memory (this request) or session (across requests). | [Data access](/workflow/data-access) |
| <a id="shelf"></a>shelf | The engineering word an asset is filed under: `model`, `context`, `harness`, `tools`, `memory`, `rag`, `agent`, `guardrails`, `evals`, `workflow`, `deploy`. The path stays the id (`harness/safety`). `kdeps list <shelf>` shows one. Install profiles are a separate axis. | [CLI reference](/reference/cli) |
| <a id="skip"></a>`skip` | A list of boolean expressions in `validations.skip`. If any is true, the resource is skipped silently and the workflow continues. | [Validation and control flow](/workflow/validation) |
| `sql` | A kdeps component (`component: sql`) for SQL against PostgreSQL, MySQL, SQLite, SQL Server or Oracle. Parameterized queries prevent injection. | [SQL](/workflow/resources/sql) |
| <a id="streaming"></a>`streaming` | A boolean field on `chat` resources. When true, the LLM response streams token by token. | [LLM backends](/workflow/resources/llm-backends) |
| <a id="targetactionid"></a>`targetActionId` | The entry-point resource of a workflow, set as `targetActionId:` under `metadata:`. Execution resolves the dependency graph backward from here. | [workflow.yaml](/workflow/configuration) |
| tools | Functions registered with the LLM. On a chat resource, tools are the functions in `tools:`. | [Built-in tools](/workflow/agent-loop/tools). [Function calling](/workflow/tools) |
| `tools.components` | Tools provided by a component that are exposed to the calling agent or workflow for the LLM to invoke. | [Components](/agencies/components) |
| <a id="validations"></a>`validations` | A resource-level block with `skip`, `check`, `routes`, `methods`, and `error`. Controls whether a resource runs and how it fails. | [Validation and control flow](/workflow/validation) |
| workflow | The top-level unit of execution: a YAML file defining resources, dependencies, and configuration. Declared as `kind: Workflow`. | [workflow.yaml](/workflow/configuration) |

## See also

- [Execution flow](/workflow/execution-flow) - how the DAG resolves and runs
- [Expression functions reference](/reference/expression-functions) - every function available in expressions
- [Expression operators](/reference/expression-operators) - comparison and logical operators
