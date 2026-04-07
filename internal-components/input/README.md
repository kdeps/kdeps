# input component

Reads workflow inputs and returns them as a structured JSON object.

## Purpose

The `input` component is designed for use with workflows that declare
`input.sources: [component]`. It collects all non-empty input slots passed
by the calling workflow and returns them as a single JSON object — useful for
logging, validation, or forwarding inputs to downstream resources.

## Usage

```yaml
run:
  component:
    name: input
    with:
      query: "{{ input('userQuery') }}"
      text: "Some text to forward"
```

The component returns a JSON object containing only the slots that received
non-empty values:

```json
{
  "query": "...",
  "text": "..."
}
```

## Input Slots

| Slot     | Purpose                      |
|----------|------------------------------|
| `query`  | Convenience slot for queries |
| `prompt` | Convenience slot for prompts |
| `text`   | Plain text input             |
| `data`   | Arbitrary data string        |
| `key`    | Key name slot                |
| `value`  | Value slot                   |
| `a`–`h`  | Generic slots A through H    |

All slots are optional strings. Empty or absent slots are omitted from the output.

## Component Input Source

Pair this component with the `component` input source to build workflows that
act as callable components:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-component-workflow
  targetActionId: process
settings:
  agentSettings:
    pythonVersion: "3.12"
input:
  sources: [component]
  component:
    description: "Accepts a query and returns a processed result"
```

When `sources: [component]` is declared:
- No HTTP server, bot listener, or file reader is started.
- The workflow is driven exclusively by `run.component` invocations from a
  parent workflow.
- Inputs arrive via the caller's `with:` block and are readable with
  `{{ input('key') }}` inside resources.

## Automatic Setup

No additional packages are required. The `input` component uses only the
Python standard library.
