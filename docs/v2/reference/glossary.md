# Glossary

Every kdeps term defined in one place. The first mention of any term on other
docs pages links here. This is a reference page and applies to both workflow
mode and agent mode.

| Term | Definition | More |
| :--- | :--- | :--- |
| <a id="actionid"></a>`actionId` | A unique string that identifies a resource within a workflow. Used as the target of `requires:` dependencies. In agent mode, resources are not exposed as tools; the whole workflow is the tool, named after `metadata.name`. | [Resources overview](/resources/overview) |
| agent | An autonomous LLM-driven pipeline defined by `kind: Agent`. Has tools, memory, and multi-step reasoning. Run with `kdeps [path]`. | [Agent loop mode](/modes/agent-loop-mode) |
| <a id="agency"></a>agency | Multiple agents composed into one system. One agent delegates a task to another via the `agent:` action type; the callee runs its full pipeline and returns its output. | [AI agencies](/concepts/agency) |
| `apiResponse` | A resource action type that returns a structured JSON response to the client. Usually the terminal node of a workflow. | [API response](/resources/api-response) |
| `before` / `after` | Expression blocks that run before or after a resource's main action. Used for data preparation, normalization, and validation. Statements execute in order. | [Expression blocks](/reference/expr-blocks) |
| `browser` | A resource action type for browser automation. Navigates pages, clicks elements, fills forms, and extracts content. | [Browser](/resources/web/browser) |
| `chat` | A resource action type for LLM chat completion. Sends a prompt and returns the response. The most common action type. | [LLM (chat)](/resources/llm/) |
| <a id="check"></a>`check` | A list of boolean expressions in `validations.check`. All must be true or the resource fails and the workflow stops. | [Validation and control flow](/concepts/validation-and-control) |
| `codeIntelligence` | A resource action type for code navigation: symbol search, definition lookup, reference finding, hover info, and diagnostics. | [Code intelligence](/resources/code-intelligence/navigation) |
| <a id="component"></a>component | A reusable, packaged resource bundle. Installed from the registry or built in a `components/` directory. Declared as `kind: Component`. | [Components](/concepts/components) |
| `componentTools` | Tools provided by a component that are exposed to the calling agent or workflow for the LLM to invoke. | [Components](/concepts/components) |
| <a id="deterministic"></a>deterministic | Same input, same output, every time. In kdeps this describes the workflow-mode pipeline - route matching, `requires:` ordering, `validations`, and response shaping all resolve the same way for a given request. It does not describe the LLM's text output. Contrast [probabilistic](#probabilistic). | [Deterministic by design](/concepts/why-kdeps#deterministic-by-design) |
| `embedding` | A resource action type that generates vector embeddings from text, for semantic search and RAG. | [Embedding](/resources/rag/embedding) |
| `exec` | A resource action type that runs shell commands and captures stdout, stderr, and exit code. | [Exec](/resources/scripting/exec) |
| expr | Short for expression: a statement evaluated by the expr-lang engine. Used in `before:` / `after:` blocks, `validations`, and `{{ }}` string interpolation. | [Expressions](/concepts/expressions) |
| `file()` | An expression function that reads uploaded or local files. Takes a glob pattern and an optional selector (`first`, `last`, `count`, `all`, `mime:<type>`). | [Unified API](/concepts/unified-api) |
| `file` (resource) | A resource action type for filesystem operations: read, write, patch, list, delete, mkdir, copy, move, append. | [File](/resources/files/file) |
| `get()` | The primary data access function. Reads from query params, headers, body, session, memory, environment, or resource outputs, with auto-detection. | [Unified API](/concepts/unified-api) |
| `git` | A resource action type for version control: status, diff, log, commit, branch, push, pull. | [Git](/resources/files/git) |
| `httpClient` | A resource action type for HTTP requests. Supports all methods, headers, query params, body, retry, and caching. | [HTTP client](/resources/web/http-client) |
| `info()` | An expression function that returns request metadata: `ID`, `timestamp`, `path`, `method`, `IP`, `sessionId`, `filecount`, `files`, `filetypes`. | [Unified API](/concepts/unified-api) |
| `items` | An array a resource iterates over. The resource runs once per item. Access the current item via `item.current()` or `get('current')`. | [Items iteration](/concepts/items) |
| `jsonResponse` | A boolean field on `chat` resources. When true, forces the LLM to return valid JSON with no markdown wrapping. | [LLM (chat)](/resources/llm/) |
| <a id="loop"></a>`loop` | A while-loop on a resource. The body runs while the `while` expression is true, up to `maxIterations`. `every:` turns it into a scheduled ticker. | [While-loop iteration](/concepts/loop) |
| memory (expression) | Request-scoped key/value storage. Set with `set('key', value, 'memory')`; read with `get('key', 'memory')`. Cleared when the request completes. | [Session and memory](/configuration/session) |
| memory (persistent) | Project-scoped storage for the agent loop in a bbolt database that survives across sessions. Built-in tools: `memory_save`, `memory_search`, `memory_delete`, `memory_list`, `memory_query`. | [Persistent memory](/concepts/memory) |
| memory graph | A directed graph of memory entries linked by their `References` fields and auto-linked by type. Inlined into the `<memory>` block in causal order. | [Memory internals](/concepts/memory-internals#memory-graph) |
| `output()` | An expression function that reads the output of a completed resource by its `actionId`. | [Unified API](/concepts/unified-api) |
| <a id="probabilistic"></a>probabilistic | The same prompt can produce different output on each call. Every language model - Claude, GPT, Gemini, Groq, Ollama, local llamafile / GGUF - is probabilistic. A kdeps `chat:` resource inherits this; the pipeline around it stays [deterministic](#deterministic). | [LLM provider reference](/reference/llm-providers) |
| `python` | A resource action type that runs Python scripts. Supports inline scripts, file paths, packages, and virtual environments. | [Python](/resources/scripting/python) |
| <a id="requires"></a>`requires` | A list of `actionId`s that must complete before this resource runs. Defines the DAG edges. Transitive dependencies resolve automatically. | [Execution flow](/guides/execution-flow) |
| resource | The fundamental building block of a workflow: an `actionId`, a primary action type, optional `requires`, and optional validations, loop, and error handling. | [Resources overview](/resources/overview) |
| scenario | A test case for a resource or workflow: inputs and expected outputs, checked during `kdeps validate`. | [Validation examples](/reference/validation-examples) |
| `scraper` | A resource action type for web scraping. Extracts structured data from HTML using CSS selectors. | [Scraper](/resources/web/scraper) |
| `searchLocal` | A resource action type for local semantic search over indexed documents. | [searchLocal](/resources/search/searchlocal) |
| `searchWeb` | A resource action type for web search via configured search APIs. | [searchWeb](/resources/search/searchweb) |
| session | Session-scoped key/value storage that persists across requests from the same client. Set with `set('key', value, 'session')`. | [Session and memory](/configuration/session) |
| `set()` | Stores a value in memory (this request) or session (across requests). | [Unified API](/concepts/unified-api) |
| <a id="skip"></a>`skip` | A list of boolean expressions in `validations.skip`. If any is true, the resource is skipped silently and the workflow continues. | [Validation and control flow](/concepts/validation-and-control) |
| `sql` | A resource action type for SQL queries against PostgreSQL, MySQL, or SQLite. Parameterized queries prevent injection. | [SQL](/resources/sql) |
| <a id="stealth-mode"></a>stealth mode | A "Muted" agent-loop UI: the whole REPL renders in near-black dark grays and the model name is barely visible - for running kdeps in public. Enable with `--stealth`, `KDEPS_STEALTH=1`, or `/stealth`. Rendering only; prompts, responses, and logs are unchanged. | [Agent loop REPL features](/modes/agent-loop-repl#stealth-mode) |
| <a id="streaming"></a>`streaming` | A boolean field on `chat` resources. When true, the LLM response streams token by token. | [LLM backends](/resources/llm/backends) |
| <a id="targetactionid"></a>`targetActionId` | The entry-point resource of a workflow, set in `metadata.targetActionId`. Execution resolves the dependency graph backward from here. | [workflow.yaml](/configuration/workflow) |
| tools | Functions registered with the LLM. In agent mode, tools are whole workflows and components. In workflow mode, tools are functions defined in `chat.tools`. | [Tools](/concepts/tools) |
| <a id="validations"></a>`validations` | A resource-level block with `skip`, `check`, `routes`, `methods`, and `error`. Controls whether a resource runs and how it fails. | [Validation and control flow](/concepts/validation-and-control) |
| workflow | The top-level unit of execution: a YAML file defining resources, dependencies, and configuration. Declared as `kind: Workflow`. | [workflow.yaml](/configuration/workflow) |

## See also

- [Execution flow](/guides/execution-flow) - how the DAG resolves and runs
- [Expression functions reference](/reference/expression-functions-reference) - every function available in expressions
- [Expression operators](/reference/expression-operators) - comparison and logical operators
