# Autopilot Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps component install autopilot`
>
> Usage: `run: { component: { name: autopilot, with: { task: "...", context: "...", model: "gpt-4o" } } }`

The Autopilot component is kdeps' goal-directed workflow synthesis engine. Describe **what you want to achieve** in plain language; autopilot synthesizes a kdeps workflow, executes it, evaluates the result, and retries with reflection if needed.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `task` | string | yes | — | Plain-language description of the goal |
| `context` | string | no | — | Additional context or constraints for the task |
| `model` | string | no | `gpt-4o` | LLM model to use for synthesis and evaluation |

## Using the Autopilot Component

```yaml
run:
  component:
    name: autopilot
    with:
      task: "Research the top 5 open-source LLM frameworks and summarize their strengths"
      context: "Focus on frameworks that support local inference"
      model: "gpt-4o"
```

Access the result via `output('<callerActionId>')`.

::: info When to use Autopilot
Use `autopilot` when the exact sequence of steps is not known ahead of time and needs to be determined dynamically. For well-understood, repeatable pipelines, declarative resources are faster and more predictable.
:::

---

## Reference: Full Autopilot Configuration

The following sections document the full configuration surface available in the underlying autopilot implementation.



## How It Works

```
Goal  ──►  Synthesize YAML  ──►  Validate  ──►  Execute  ──►  Evaluate
                ▲                                                  │
                │                  retry with reflection           │
                └──────────────────────────────────────────────────┘
                                 (up to maxIterations)
```

Each iteration:

1. **Synthesize** — An LLM generates a complete kdeps workflow YAML from the goal and any previous iteration context (reflection).
2. **Validate** — The synthesized YAML is validated structurally (`apiVersion`, `kind`, `metadata.name`) before execution.
3. **Execute** — The validated workflow is run through the kdeps engine against the current execution context.
4. **Evaluate** — An LLM determines whether the result satisfies the goal, returning a boolean verdict and a natural-language explanation.

If the goal is not met, the evaluation context is fed back into the next synthesis prompt so the model can reflect on what went wrong.

## Basic Usage

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: pilot
  name: Autopilot Task

run:
  autopilot:
    goal: "Find the capital of France and return it as plain text."
    maxIterations: 3
```

</div>

## Configuration Reference

```yaml
run:
  autopilot:
    # Required: plain-language description of what to accomplish
    goal: "Summarize the top 5 Hacker News stories in bullet points."

    # Optional: maximum synthesis+execute+evaluate cycles (default: 3)
    maxIterations: 5

    # Optional: LLM model for synthesis and evaluation
    # If omitted, uses the system-configured default model
    model: "gpt-4o"

    # Optional: resource types the LLM may use in synthesized workflows
    # Helps scope what the model can reach for
    availableTools:
      - chat
      - httpClient
      - search
      - sql

    # Optional: plain-language or expression hint for when "done" means done
    # Passed verbatim to the LLM evaluator as additional context
    successCriteria: "The result must contain at least 5 bullet points."

    # Optional: store the AutopilotResult JSON under this key
    # Access it later with get('myKey')
    storeAs: "autopilotOutput"
```

## Output Structure

The autopilot returns an `AutopilotResult` object. When `storeAs` is set, it is available via `get()`:

<div v-pre>

```yaml
run:
  autopilot:
    goal: "Classify the customer complaint in get('complaint')."
    storeAs: "classification"

  apiResponse:
    success: true
    response:
      result: "{{ get('classification').finalResult }}"
      iterations: "{{ get('classification').totalRuns }}"
      succeeded: "{{ get('classification').succeeded }}"
```

</div>

### AutopilotResult Fields

| Field | Type | Description |
|-------|------|-------------|
| `goal` | `string` | The original goal string |
| `succeeded` | `bool` | `true` if any iteration met the success criteria |
| `totalRuns` | `int` | Number of iterations actually executed |
| `finalResult` | `any` | Output of the last executed iteration |
| `iterations` | `[]AutopilotIteration` | Per-iteration audit trail |

### AutopilotIteration Fields

| Field | Type | Description |
|-------|------|-------------|
| `index` | `int` | Zero-based iteration number |
| `synthesizedYaml` | `string` | The YAML generated for this iteration |
| `result` | `any` | Execution output of this iteration |
| `evaluation` | `string` | LLM's explanation of success/failure |
| `succeeded` | `bool` | Whether this iteration met the goal |
| `error` | `string` | Set if synthesis, validation, or execution failed |

## Examples

### Research & Summarize

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: researcher
  version: "1.0.0"
  targetActionId: research

resources:
  - metadata:
      actionId: research
      name: Autopilot Researcher
    run:
      autopilot:
        goal: "Search the web for '{{ get('q') }}' and return a 3-paragraph summary."
        maxIterations: 4
        model: "gpt-4o"
        availableTools: [search, chat]
        successCriteria: "Response contains at least 3 paragraphs."
        storeAs: "researchResult"

      apiResponse:
        success: true
        response:
          summary: "{{ get('researchResult').finalResult }}"
          attempts: "{{ get('researchResult').totalRuns }}"
```

</div>

### Data Pipeline with Fallback

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: dataPipeline
  name: Autopilot Data Pipeline
run:
  autopilot:
    goal: |
      Query the database for users who signed up in the last 7 days,
      compute their average session duration, and return a JSON object
      with keys: count, avgSessionSeconds.
    maxIterations: 5
    availableTools: [sql, python, chat]
    successCriteria: "Result is a JSON object with 'count' and 'avgSessionSeconds' keys."
    storeAs: "pipelineResult"
```

</div>

### Inspecting the Iteration Trail

<div v-pre>

```yaml
run:
  autopilot:
    goal: "Generate a UUID and return it."
    maxIterations: 2
    storeAs: "ap"

  apiResponse:
    success: true
    response:
      result: "{{ get('ap').finalResult }}"
      trail: "{{ get('ap').iterations }}"
```

</div>

## Retry and Reflection

When an iteration fails (synthesis error, validation error, execution error, or evaluation says "not done"), the next synthesis call receives the full history of previous iterations — including their errors and evaluations — so the LLM can reason about what went wrong and adjust its approach.

This means:
- A validation failure ("missing `metadata.name`") is reflected back so the model learns to include it.
- An execution failure ("tool X not found") is reflected back so the model chooses a different tool.
- An evaluation failure ("only 2 paragraphs, need 3") prompts the model to extend the output.

## Scoping with `availableTools`

`availableTools` constrains which resource types the LLM is told it may use. This is a **hint to the synthesizer prompt**, not an enforcement mechanism. It helps the model stay focused and avoids hallucinating resource types that aren't configured.

```yaml
availableTools:
  - chat       # LLM interaction
  - httpClient # External APIs
  - sql        # Databases
  - exec       # Shell commands
  - python     # Python scripts
  - search     # Web/local search
  - scraper    # Content extraction
  - embedding  # Vector search
```

If omitted, the synthesizer is told all resource types are available.

## Performance Considerations

- Each iteration involves at least two LLM calls (synthesis + evaluation), plus the execution cost of the synthesized workflow.
- Keep `maxIterations` low (3–5) for latency-sensitive use cases.
- Use `successCriteria` to exit early as soon as the result is good enough.
- For deterministic, well-understood tasks, use standard resources — autopilot adds overhead by design.

## Differences from `agent:`

| | `agent:` | `autopilot:` |
|-|----------|-------------|
| Workflow | Pre-written, static | Synthesized at runtime |
| Steps | Known ahead of time | Determined by LLM |
| Retry | Manual (`onError:`) | Automatic with reflection |
| Predictability | High | Variable |
| Use case | Production pipelines | Exploratory / dynamic tasks |

## See Also

- [Agency & Multi-Agent](../concepts/agency.md) — Static agent delegation
- [LLM Resource](./llm.md) — Direct LLM interaction
- [Error Handling](../concepts/error-handling.md) — `onError:` for deterministic fallbacks
- [Expression Functions](../concepts/expression-functions-reference.md) — `get()`, `set()`, and more
