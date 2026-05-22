# Execution Flow

How kdeps resolves, orders, and runs resources in a workflow.

## Overview

When you run `kdeps run workflow.yaml` or call `POST /api/v1/run`, the engine:

1. Parses the workflow and builds a dependency graph from `requires` fields
2. Detects cycles (fails fast if any exist)
3. Finds the `targetActionId` resource and walks its transitive dependencies
4. Topologically sorts resources so dependencies run before dependents
5. Executes resources in order -- resources without shared dependencies can run concurrently

## Execution Order

```
Request enters at targetActionId
              │
              v
┌──────────────────────────────┐
│  1. Build dependency graph   │  <- each resource is a node; requires = edges
│     Detect cycles            │  <- fails fast with ErrCodeDependencyCycle
└──────────────┬───────────────┘
               │
               v
┌──────────────────────────────┐
│  2. Walk transitive deps     │  <- from targetActionId backward through requires
│     Collect relevant nodes   │  <- only resources needed to reach target
└──────────────┬───────────────┘
               │
               v
┌──────────────────────────────┐
│  3. Topological sort         │  <- dependencies always run before dependents
│     (resources without       │  <- resources that share no transitive deps
│      shared deps can run     │     are independent and can run concurrently)
│      concurrently)           │
└──────────────┬───────────────┘
               │
               v
┌──────────────────────────────┐
│  4. Execute each resource    │
│     a. Run before: block     │  <- prepare data, normalize inputs
│     b. Evaluate skip         │  <- any true? skip silently, continue
│     c. Evaluate check        │  <- all true? proceed, else fail
│     d. Run main action       │  <- chat, httpClient, sql, etc.
│     e. Run after: block      │  <- process output, store in memory/session
│     f. Handle onError        │  <- if main action fails, run error handler
└──────────────┬───────────────┘
               │
               v
┌──────────────────────────────┐
│  5. Terminal resource runs   │  <- typically apiResponse: formats output
│     Response returned        │  <- JSON body with success/data/error
└──────────────────────────────┘
```

## Dependency Graph

### How requires works

<div v-pre>

```yaml
resources:
  - actionId: fetchData
    httpClient:
      url: https://api.example.com/data

  - actionId: analyzeData
    requires: [fetchData]            # runs only after fetchData succeeds
    chat:
      prompt: "Analyze: {{ output('fetchData') }}"

  - actionId: respond
    requires: [analyzeData]          # runs only after analyzeData succeeds
    apiResponse:
      response: "{{ output('analyzeData') }}"

metadata:
  targetActionId: respond            # engine walks backward from here
```

</div>

Execution order: `fetchData` -> `analyzeData` -> `respond`

### Transitive dependencies

The engine resolves the full transitive closure. If A requires B, and B requires C, then C runs before both:

<div v-pre>

```yaml
- actionId: C
  exec:
    command: echo "first"

- actionId: B
  requires: [C]
  chat:
    prompt: "Build on: {{ output('C') }}"

- actionId: A
  requires: [B]
  apiResponse:
    response: "{{ output('B') }}"
```

</div>

Execution order: `C` -> `B` -> `A`

### Independent resources

Resources that don't depend on each other (neither directly nor transitively) can run concurrently:

<div v-pre>

```yaml
- actionId: fetchUsers
  httpClient:
    url: https://api.example.com/users

- actionId: fetchProducts                    # no requires -- independent of fetchUsers
  httpClient:
    url: https://api.example.com/products

- actionId: merge
  requires: [fetchUsers, fetchProducts]      # waits for both
  chat:
    prompt: "Users: {{ output('fetchUsers') }}, Products: {{ output('fetchProducts') }}"
```

</div>

`fetchUsers` and `fetchProducts` run concurrently. `merge` waits for both.

## Cycle Detection

The engine detects cycles during graph construction and fails fast with `ErrCodeDependencyCycle`:

```yaml
# This creates a cycle and will fail:
- actionId: A
  requires: [B]

- actionId: B
  requires: [A]    # cycle: A -> B -> A
```

## Skip vs Check

Both run before the main action, but they behave differently:

| | skip | check |
|---|---|---|
| Logic | ANY expression true triggers skip | ALL expressions must be true |
| On failure | Resource skipped silently, workflow continues | Workflow stops, error returned |
| Error code | N/A | Configurable via `validations.error.code` |
| Use case | Conditional execution, optional resources | Input validation, preconditions |

### Skip example

```yaml
validations:
  skip:
    - get('q') == ''          # if query is empty, skip this resource
```

### Check example

```yaml
validations:
  check:
    - get('apiKey') != nil    # must have API key
    - len(get('q')) <= 500    # query must be under 500 chars
  error:
    code: 400
    message: Missing API key or query too long
```

## Loop Execution

When a resource has a `loop` config, the engine runs the full resource cycle (before -> check -> action -> after) repeatedly:

```
┌─────────────────────────────────┐
│  while loop.condition is true:  │
│    ┌──────────────────────┐     │
│    │  before: block       │     │
│    │  check (every iter)  │     │
│    │  main action         │     │
│    │  after: block        │     │
│    └──────────────────────┘     │
│    if loop.every is set:        │
│      sleep(every)               │
│    if iteration >= maxIterations:│
│      break (safety cap)         │
└─────────────────────────────────┘
```

See [Loop](/concepts/loop) for full details.

## Agent Mode Execution

In agent mode (`kdeps serve`), the execution model differs:

1. All resources are registered as tools (tool name = `actionId`)
2. The LLM receives the user prompt and decides which tool(s) to call
3. Each tool call dispatches a single-resource execution (before -> check -> action -> after)
4. Results return to the LLM, which may call more tools or produce a final answer
5. The loop continues until the LLM stops calling tools

Agent mode does not use `targetActionId` or the DAG -- the LLM drives execution dynamically.

## See Also

- [Workflow Mode](/modes/workflow-mode) -- deterministic DAG pipelines
- [Agent Mode](/modes/agent-mode) -- LLM-driven tool calling
- [Validation & Control Flow](/concepts/validation-and-control) -- skip, check, and error handling
- [Loop](/concepts/loop) -- while-loop iteration
