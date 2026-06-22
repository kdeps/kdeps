# Glossary

Every kdeps term defined in one place. First mention of any term in other docs pages links here.

## A

### actionId
A unique string identifier for a [resource](#resource) within a workflow. Used as the target for `requires` dependencies. Must be unique across all resources in the workflow. In [agent mode](#agent-mode), resources are not exposed as tools directly -- the whole workflow is the tool, named after `metadata.name`.

### agent
An autonomous LLM-driven pipeline defined by `kind: Agent` in a workflow. Agents have tools, memory, and multi-step reasoning. Run with `kdeps [path]`. See [Agent Mode](/modes/agent-loop-mode).

### agency
A pattern where one agent calls another agent as a sub-agent. The caller delegates a task, the callee runs autonomously, and the result is returned. Defined via the `agent:` action type. See [Agencies](/concepts/agency).

### apiResponse
A resource action type that returns a structured JSON response to the client. The terminal node in most workflows -- the final resource that formats and sends the output. See [API Response](/resources/api-response).

## B

### before
An expression block that runs before a resource's main action. Use for data preparation, normalization, and input validation. Each expression executes sequentially. See [Expression Blocks](/reference/expr-blocks).

### browser
A resource action type for browser automation (Playwright-based). Navigates pages, clicks elements, fills forms, and extracts content. See [Browser](/resources/browser).

## C

### chat
A resource action type for LLM chat completion. Sends a prompt to a language model and returns the response. The most commonly used action type. See [LLM (Chat)](/resources/llm).

### check
A list of boolean expressions in `validations.check`. ALL must be true for the resource to execute. If any is false, the resource fails with a configurable error code and message. Unlike [skip](#skip), check failure stops the workflow. See [Validation & Control Flow](/concepts/validation-and-control).

### component
A reusable, packaged workflow published to the kdeps registry. Components encapsulate resources, logic, and configuration that can be imported into other workflows. Defined by `kind: Component`. See [Components](/concepts/components).

### componentTools
Tools provided by a [component](#component) that are exposed to the calling agent or workflow. When an agent calls a component, the component's tools become available for the LLM to invoke.

### codeIntelligence
A resource action type for code navigation and intelligence. Supports symbol search, definition lookup, reference finding, document symbols, hover info, and diagnostics via ripgrep and Go vet. See [Code Intelligence](/resources/codeintelligence).

## E

### embedding
A resource action type that generates vector embeddings from text. Used for semantic search and RAG pipelines. See [Embedding](/resources/embedding).

### exec
A resource action type that runs shell commands. Captures stdout, stderr, and exit code. See [Exec](/resources/exec).

### expr
<div v-pre>

Short for "expression." A statement evaluated by the expr-lang engine. Used in `before`/`after` blocks, `validations.check`/`validations.skip`, and `{{ }}` string interpolation.

</div>

## F

### file
Accesses uploaded files or local files. The `file()` function takes a glob pattern and optional selector (`first`, `last`, `count`, `all`, `mime:<type>`).

### file (resource)
A resource action type for filesystem operations. Supports read, write, patch, list, delete, exists, mkdir, copy, move, and append operations on files and directories. See [File](/resources/file).

## G

### get
The primary data access function. Retrieves values from query params, headers, request body, session, memory, environment variables, or resource outputs. Uses auto-detection when no source hint is given. See [Expression Functions](/reference/expression-functions-reference).

### git
A resource action type for version control operations. Supports status, diff, log, commit, branch, push, pull, and other git operations. See [Git](/resources/git).

## H

### httpClient
A resource action type for making HTTP requests. Supports GET, POST, PUT, DELETE, headers, query params, request body, retry, and caching. See [HTTP Client](/resources/http-client).

## I

### info
Returns request metadata: `ID`, `timestamp`, `path`, `method`, `IP`, `sessionId`, `filecount`, `files`, `filetypes`.

### items
An array of data that a resource iterates over. When set, the resource runs once per item. Access the current item via `item.current()` or `get('current')`. See [Items & Loop](/concepts/items).

## J

### jsonResponse
A boolean field on `chat` resources. When `true`, forces the LLM to return valid JSON (no markdown wrapping, no explanatory text).

## L

### loop
A while-loop configuration on a resource. The resource body executes repeatedly as long as the `while` expression is true, up to `maxIterations`. Supports `every` for ticker-style scheduled execution. See [Loop](/concepts/loop).

## M

### memory
Request-scoped key/value storage. Values set with `set('key', value, 'memory')` persist for the duration of a single request and are accessible via `get('key', 'memory')`. Cleared when the request completes.

## O

### output
Accesses the output of a completed resource by its actionId. Use `output('actionId')` to read results from upstream resources.

## P

### python
A resource action type that runs Python scripts. Supports inline scripts, file paths, packages, and virtual environments. See [Python](/resources/python).

## R

### requires
A list of actionIds that must complete successfully before this resource can run. Defines DAG edges. The engine resolves transitive dependencies automatically.

### resource
The fundamental building block of a kdeps workflow. Each resource has an `actionId`, a primary action type (chat, httpClient, sql, etc.), optional `requires` dependencies, and optional validations/loop/error handling.

## S

### scenario
A test scenario defined for a resource or workflow. Scenarios specify inputs and expected outputs, used for validation during `kdeps validate`.

### scraper
A resource action type for web scraping. Extracts structured data from HTML pages using CSS selectors. See [Scraper](/resources/scraper).

### searchLocal
A resource action type for local semantic search over indexed documents. See [Search](/resources/search).

### searchWeb
A resource action type for web search via configured search APIs.

### session
Session-scoped key/value storage. Values set with `set('key', value, 'session')` persist across requests from the same client. See [Session & Persistence](/configuration/session).

### set
Stores a value in memory or session. Usage: `set('key', value)` (memory) or `set('key', value, 'session')` (persistent across requests).

### skip
A list of boolean expressions in `validations.skip`. If ANY expression is true, the resource is skipped. Unlike [check](#check), skipping is silent -- the workflow continues to the next resource. See [Validation & Control Flow](/concepts/validation-and-control).

### sql
A resource action type for SQL database queries. Supports PostgreSQL, MySQL, SQLite. Parameterized queries prevent injection. See [SQL](/resources/sql).

### streaming
A boolean field on `chat` resources. When `true`, the LLM response is streamed token-by-token instead of waiting for the full response.

## T

### targetActionId
The entry point resource for a workflow. Execution starts by resolving the dependency graph from this resource backward. Set in `metadata.targetActionId`.

### tools
Functions registered with the LLM. In [agent mode](#agent-mode), tools are whole workflows (named after `metadata.name`) and components -- not individual resources. The LLM decides which tool to call based on the user prompt. In workflow mode, tools are custom functions defined in `chat.tools` on a `chat:` resource. See [Tools](/concepts/tools).

## V

### validations
A resource-level configuration block containing `skip`, `check`, `routes`, `methods`, and `error`. Controls whether a resource runs and how it fails. See [Validation & Control Flow](/concepts/validation-and-control).

## W

### workflow
The top-level unit of execution in kdeps. A YAML file defining resources, their dependencies, and configuration. Declared as `kind: Workflow`. See [workflow.yaml Reference](/configuration/workflow).

## See Also

- [Execution Flow](/guides/execution-flow) -- how the DAG resolves and runs
- [Expression Functions Reference](/reference/expression-functions-reference) -- all functions available in expressions
- [Expression Operators](/reference/expression-operators) -- comparison and logical operators
