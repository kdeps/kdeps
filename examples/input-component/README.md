# input-component

Demonstrates the **built-in `input` component** — a pre-installed component that collects named input slots and returns them as structured JSON.

No `kdeps component install` needed. The `input` component ships with kdeps.

## Structure

```
input-component/
├── workflow.yaml
└── resources/
    ├── 01-collect.yaml   # calls built-in input component
    ├── 02-answer.yaml    # passes collected inputs to LLM
    └── 03-response.yaml  # returns inputs + LLM answer
```

## Usage

```bash
kdeps run examples/input-component
```

## Requirements

- Ollama running locally with `llama3.2` (`ollama pull llama3.2`)

## The Built-in `input` Component

The `input` component accepts up to 14 named slots:

| Slot | Purpose |
|------|---------|
| `query` | A query string |
| `prompt` | An LLM prompt |
| `text` | Plain text |
| `data` | Arbitrary data |
| `key` | A key name |
| `value` | A value |
| `a` – `h` | Eight generic slots |

Only non-empty slots are included in the JSON output.

## How It Works

```yaml
# 1. Collect inputs
run:
  component:
    name: input
    with:
      query: "What is the capital of France?"
      text: "The Eiffel Tower is in Paris."
      key: "location"
```

The output is a JSON object with only the provided slots:
```json
{"query":"What is the capital of France?","text":"The Eiffel Tower is in Paris.","key":"location"}
```

Access individual fields from the output:
```yaml
# Using json_decode filter
"{{ output('collectInputs') | json_decode | get('query') }}"
```

## When to Use the `input` Component

The `input` component is most useful in workflows that declare `sources: [component]` — i.e., sub-workflows designed to be called by a parent. The parent passes inputs via `with:`, the sub-workflow uses the `input` component to collect and normalize them, then proceeds with its logic.

See the `component-input-source` example for a complete `sources: [component]` workflow.

## Validate

```bash
kdeps validate examples/input-component
```
