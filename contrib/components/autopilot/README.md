# autopilot

> Autonomous task execution via LLM planning.

The autopilot component takes a natural-language task description, sends it to an LLM with a structured planning prompt, and returns the model's response. Use it to add AI-driven automation to any workflow step.

## Inputs

| Name | Type | Required | Default | Description |
|------|------|----------|---------|-------------|
| `task` | string | yes | — | Task description for the LLM to plan and execute |
| `context` | string | no | `""` | Additional context to help the LLM understand the task |
| `model` | string | no | `gpt-4o` | LLM model to use for planning |

## Outputs

Accessed via `output('<your-action-id>')`:

| Field | Type | Description |
|-------|------|-------------|
| (response body) | string | The LLM's plan and execution response text |

The output is the raw chat response from the configured model.

## Usage

```yaml
run:
  component:
    name: autopilot
    with:
      task: "Summarize the quarterly sales report and highlight key trends"
      context: "We are a B2B SaaS company focused on SMBs."
      model: gpt-4o
  apiResponse:
    success: true
    response:
      plan: "{{ output('myAction') }}"
```

Minimal (required inputs only):

```yaml
run:
  component:
    name: autopilot
    with:
      task: "Write a Python function that parses ISO 8601 dates"
```

## Requirements

- An LLM backend must be configured in the workflow's `agentSettings` (Ollama, OpenAI, etc.).
- The `model` input must match a model available on the configured backend.

## Notes

- The component wraps a single `run.chat:` resource. Any model supported by the workflow's LLM backend can be used.
- Set `context` to give the model relevant background when the task alone is ambiguous.
