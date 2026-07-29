# Resources

One step: id, optional deps, optional validation, **one** primary action.

Works in workflow mode as DAG nodes, and inside workflows the [coding agent](/agent) calls as tools.

## Shape

<div v-pre>

```yaml
actionId: fetch
name: Fetch URL
requires: [validate]
validations:
  methods: [POST]
  check:
    - get('url') != ''
  error:
    code: 400
    message: url required
before:
  - set('u', trim(get('url')))
httpClient:
  method: GET
  url: "{{ get('u') }}"
onError:
  action: retry
  maxRetries: 2
```

</div>

- `requires:` — order; read with `get('id')`  
- `items:` / `loop:` — [Iteration](/iteration)  
- `onError:` — [Errors](/errors)  
- `before` / `after` — [Expressions](/expressions)  

## Actions (built-in)

| Key | Role |
|-----|------|
| `chat` | LLM — [LLM](/llm) |
| `httpClient` | HTTP |
| `sql` | Database |
| `python` | Script (stdout / JSON) |
| `exec` | Shell |
| `file` / `git` / `codeIntelligence` | Local code/fs |
| `scraper` / `browser` | Fetch / Playwright |
| `searchWeb` / `searchLocal` | Search |
| `embedding` / `loader` / `vectorStore` | Knowledge / RAG |
| `email` / `telephony` / `botReply` | Messaging |
| `agent` | Call another agent — [Agencies](/agencies) |
| `transcribe` | Speech-to-text |
| `apiResponse` | HTTP response |
| `component` | Registry pack — [Components](/components) |

## Validation sketch

```yaml
validations:
  methods: [POST]
  routes: [/api/v1/chat]
  headers: [Authorization]
  skip:
    - get('dry_run') == true
  check:
    - get('q') != ''
  error:
    code: 400
    message: required
```

## Registry components

```bash
kdeps registry install <name>
```

```yaml
component:
  name: botreply
  with:
    platform: telegram
    message: "Hello!"
```

[Inputs](/inputs) · [Workflow](/workflow) · [CLI](/cli).
