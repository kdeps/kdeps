# Execution flow

How kdeps resolves, orders, and runs resources in a workflow. This applies to both modes: it is the DAG that workflow mode runs directly, and the same DAG that executes when agent mode calls a workflow tool.

## Overview

When you run `kdeps run workflow.yaml` or call `POST /api/v1/run`, the engine:

1. Parses the workflow and builds a dependency graph from [`requires`](/reference/glossary#requires) fields
2. Detects cycles (fails fast if any exist)
3. Finds the [`targetActionId`](/reference/glossary#targetactionid) resource and walks its transitive dependencies
4. Topologically sorts resources so dependencies run before dependents. The order is the same on each run. kdeps uses the `requires` of a resource in the order that the resource lists them
5. Executes resources in that order, running independent resources in parallel (see [Independent resources](#independent-resources))

## Execution order

```d2
direction: down

A: Request at targetActionId {shape: oval}
B: "1. Build dependency graph\neach resource = node; requires = edges\nfails fast on cycles"
C: "2. Walk transitive deps\nbackward from targetActionId\ncollect only needed nodes"
D: "3. Topological sort\ndeps always run before dependents\nindependent resources run in parallel"
E: "4. Execute each resource\na. before: block\nb. skip - any true? skip silently\nc. check - all true? proceed, else fail\nd. main action (chat, sql, http...)\ne. after: block\nf. onError handler"
F: "5. Terminal resource\napiResponse: formats output"
G: JSON response returned {shape: oval}

A -> B -> C -> D -> E -> F -> G
```

## Dependency graph

### How requires works

<div v-pre>

```yaml
# workflow.yaml (conceptual)
metadata:
  targetActionId: respond            # engine walks backward from here

resources:
  - actionId: fetchData
    component:
      name: http_request
      with:
        url: https://api.example.com/data

  - actionId: analyzeData
    flow:
      requires: [fetchData]            # runs only after fetchData succeeds
    chat:
      context:
        prompt: "Analyze: {{ output('fetchData') }}"

  - actionId: respond
    flow:
      requires: [analyzeData]          # runs only after analyzeData succeeds
    apiResponse:
      response: "{{ output('analyzeData') }}"
```

</div>

Execution order: `fetchData` -> `analyzeData` -> `respond`

### Transitive dependencies

The engine resolves the full transitive closure. If A requires B, and B requires C, then C runs before both:

<div v-pre>

```yaml
# resources/example.yaml
- actionId: C
  component:
    name: exec
    with:
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

Resources that do not depend on each other run in parallel, a maximum of 4 at the same time. A resource starts only after its `requires` finish:

<div v-pre>

```yaml
# resources/example.yaml
- actionId: draft
  chat:
    prompt: "{{ get('q') }}"

- actionId: reviewScope                      # requires draft only: runs beside reviewRisks
  requires: [draft]
  chat:
    prompt: "Review the scope: {{ output('draft') }}"

- actionId: reviewRisks
  requires: [draft]
  chat:
    prompt: "Review the risks: {{ output('draft') }}"

- actionId: report
  requires: [reviewScope, reviewRisks]       # starts when both reviews finish
  chat:
    prompt: "Merge: {{ output('reviewScope') }} {{ output('reviewRisks') }}"
```

</div>

`reviewScope` and `reviewRisks` run at the same time. `report` waits for both.

The engine also orders resources that share a key without `requires`:

| In a resource's YAML | Effect |
|---|---|
| `get('k')` after another resource's `set('k')` | waits for the writer |
| `set('k')` after another resource's `get('k')` or `set('k')` | keeps their order |
| `output('id')` or `get('id')` of another resource | waits for that resource |

A `component:` call runs beside others when the component is read-only in the [permission policy](/agent/tools#permission-modes) (`web_search`, `read_file`, `web_scraper`, ...) and has no `teardown:`. Each call gets its own inputs and outputs. As a result, three `web_search` calls with different queries run at the same time. Other components (`write_file`, `bash_exec`, `http_request`, ...) run alone.

A resource whose effects the YAML can't show also runs alone, in its usual place: `exec:`, `python:`, `items:`, `loop:`, `apiResponse:`, `chat` with `tools`, header or param `validations`, a computed key (`get(name)`), `'item'`/`'loop'` storage, `session()` and `memory_*` calls.

One cap covers both modes: workflow resources, subagents, judges and agent tool calls. It is the `parallel` event:

```yaml
# ~/.kdeps/events/parallel.yaml
name: parallel
items: 4          # tasks that run at once, in each batch
run: cap
disabled: false   # true runs everything one at a time
```

## Cycle detection

The engine detects cycles during graph construction and fails fast with `ErrCodeDependencyCycle`:

```yaml
# This creates a cycle and will fail:
- actionId: A
  requires: [B]

- actionId: B
  requires: [A]    # cycle: A -> B -> A
```

## Skip vs check

Both run before the main action, but they behave differently:

| | skip | check |
|---|---|---|
| Logic | ANY expression true triggers skip | ALL expressions must be true |
| On failure | Resource skipped silently, workflow continues | Workflow stops, error returned |
| Error code | N/A | Configurable via `validations.error.code` |
| Use case | Conditional execution, optional resources | Input validation, preconditions |

### Skip example

```yaml
# resources/example.yaml
validations:
  skip:
    - get('q') == ''          # if query is empty, skip this resource
```

### Check example

```yaml
# resources/example.yaml
validations:
  check:
    - get('apiKey') != nil    # must have API key
    - len(get('q')) <= 500    # query must be under 500 chars
  error:
    code: 400
    message: Missing API key or query too long
```

## Loop execution

When a resource has a [`loop`](/reference/glossary#loop) config, the engine runs the full resource cycle (before -> check -> action -> after) repeatedly:

```d2
direction: down

A: loop starts {shape: oval}
B: "loop.condition is true?" {shape: diamond}
C: "before: block"
D: check (every iteration)
E: main action
F: "after: block"
H: "loop.every set?" {shape: diamond}
I: sleep(every)
J: "iteration >= maxIterations?" {shape: diamond}
G: loop ends {shape: oval}

A -> B
B -> C: yes
B -> G: no
C -> D -> E -> F -> H
H -> I: yes
H -> J: no
I -> J
J -> G: yes, safety cap
J -> B: no
```

See [Loop](/workflow/loop) for full details.

## Agent mode execution

In agent mode (`kdeps [path]`), the execution model differs:

1. kdeps registers each workflow as one tool (tool name = `name:` under `metadata:`)
2. kdeps registers each agency as one tool (tool name = `agency.metadata.name`). The LLM cannot call the internal agents directly
3. kdeps also registers the components from each loaded workflow as separate tools
4. The LLM receives the user prompt and selects the tool to call
5. A workflow tool call runs the full DAG. All `requires:` dependencies run in order. kdeps returns `apiResponse.response` to the LLM
6. Agency tool calls run the agency's entry-point pipeline. Internal `agent:` resources resolve against the agency's agent map
7. Component tool calls run the component in isolation with inputs mapped to its interface fields
8. The LLM can call more tools or give a final answer

Pointing at a single file registers one tool. Pointing at a folder registers one tool per workflow/agency found recursively (plus components). Resources are never registered as individual tools.

## See also

- [Workflow mode](/workflow/) - deterministic DAG pipelines
- [Agent mode](/agent/) - LLM-driven tool calling
- [Validation & control flow](/workflow/validation) - skip, check, and error handling
- [Loop](/workflow/loop) - a loop that repeats when a condition is true
