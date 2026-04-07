# Autopilot Resource

> **Note**: This capability is now provided as an installable component. See the [Components guide](../concepts/components) for how to install and use it.
>
> Install: `kdeps component install autopilot`
>
> Usage: `run: { component: { name: autopilot, with: { task: "...", context: "...", model: "gpt-4o" } } }`

The Autopilot component is kdeps' goal-directed task execution engine. Describe **what you want to achieve** in plain language; autopilot uses an LLM to plan and execute the task.

## Component Inputs

| Input | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `task` | string | yes | — | Plain-language description of the goal |
| `context` | string | no | — | Additional context or constraints for the task |
| `model` | string | no | `gpt-4o` | LLM model to use for planning and execution |

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
Use `autopilot` when the exact sequence of steps is not known ahead of time. For well-understood, repeatable pipelines, declarative resources are faster and more predictable.
:::

---

## Result Map

| Field | Type | Description |
|-------|------|-------------|
| `success` | bool | `true` if the task completed successfully |
| `result` | any | Output of the executed task |
| `model` | string | Model used for planning |

---

## Expression Support

All fields support [KDeps expressions](../concepts/expressions):

<div v-pre>

```yaml
run:
  component:
    name: autopilot
    with:
      task: "{{ get('user_task') }}"
      context: "{{ get('user_context') }}"
      model: gpt-4o
```

</div>

---

## Examples

### Research and Summarize

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: research
  name: Autopilot Researcher

run:
  component:
    name: autopilot
    with:
      task: "Search the web for '{{ get('q') }}' and return a 3-paragraph summary."
      model: "gpt-4o"
```

</div>

### Data Analysis

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: analyze
  name: Autopilot Data Analysis

run:
  component:
    name: autopilot
    with:
      task: |
        Analyze the following data and return key insights as JSON:
        {{ get('data') }}
      context: "Focus on trends and anomalies."
      model: gpt-4o
```

</div>

---

## Differences from `agent:`

| | `agent:` | `autopilot:` |
|-|----------|-------------|
| Workflow | Pre-written, static | LLM-directed at runtime |
| Steps | Known ahead of time | Determined by model |
| Predictability | High | Variable |
| Use case | Production pipelines | Exploratory / dynamic tasks |

## See Also

- [Agency and Multi-Agent](../concepts/agency.md) - Static agent delegation
- [LLM Resource](./llm.md) - Direct LLM interaction
- [Error Handling](../concepts/error-handling.md) - `onError:` for deterministic fallbacks
