# Shared Workflow Example

Demonstrates the **shared workflow** pattern: one sub-agent (`shared-llm`) is called
multiple times by a parent agent (`faq-router`) with different parameters, all bundled
together in an agency.

## What it does

`POST /api/v1/faq` accepts a question and returns **two answers in one response** — a
brief (1–2 sentence) summary and a detailed (4–6 sentence) explanation — both generated
by the same underlying LLM resource.

```json
// POST /api/v1/faq   body: {"q": "What is quantum entanglement?"}
{
  "success": true,
  "data": {
    "question": "What is quantum entanglement?",
    "brief":    "Quantum entanglement is a ...",
    "detailed": "Quantum entanglement is a phenomenon where ..."
  }
}
```

## Directory layout

```
shared-workflow/
├── agency.yaml                          # bundles both agents
└── agents/
    ├── shared-llm/                      # re-usable LLM wrapper (no HTTP server)
    │   ├── workflow.yaml
    │   └── resources/
    │       ├── generate.yaml            # chat: resource with style param
    │       └── llm-result.yaml          # targetActionId — returns chat output
    └── faq-router/                      # entry-point HTTP agent
        ├── workflow.yaml
        └── resources/
            ├── brief-answer.yaml        # calls shared-llm with style=brief
            ├── detailed-answer.yaml     # calls shared-llm with style=detailed
            └── faq-response.yaml        # combines both into the final response
```

## How the shared workflow pattern works

The `agent:` resource type lets any workflow call another workflow within the same
agency by name.  The called workflow runs its full action graph and returns its
`targetActionId` result to the caller, which stores it under the calling resource's
`actionId`.

```
faq-router  ──briefAnswer──►  shared-llm (style=brief)   ──►  llmResult
            ──detailedAnswer► shared-llm (style=detailed) ──►  llmResult
```

`brief-answer.yaml` (excerpt):

```yaml
run:
  agent:
    name: shared-llm      # matches metadata.name in agents/shared-llm/workflow.yaml
    params:
      prompt: "{{ get('q') }}"
      style: "brief"
```

`faq-response.yaml` then reads both results:

```yaml
run:
  apiResponse:
    response:
      data:
        brief:    get('briefAnswer')
        detailed: get('detailedAnswer')
```

## Running

```bash
kdeps run examples/shared-workflow/agency.yaml
```

Then send a request:

```bash
curl -s -X POST http://localhost:16395/api/v1/faq \
  -H 'Content-Type: application/json' \
  -d '{"q": "What is quantum entanglement?"}' | jq .
```

## Requirements

- [Ollama](https://ollama.com) installed and running (or set `installOllama: true` in
  `workflow.yaml` to have kdeps install it automatically)
- `llama3.2:1b` model pulled: `ollama pull llama3.2:1b`
