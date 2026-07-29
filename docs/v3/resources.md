# Resources

A resource is one step. It has an id, optional dependencies, optional validation, and exactly one primary action.

Works in **both** modes: DAG steps in workflow mode; inside a whole-workflow tool in agent mode.

## Shape

<div v-pre>

```yaml
actionId: fetch
name: Fetch URL
requires:
  - validate
validations:
  methods: [POST]
  check:
    - get('url') != ''
  error:
    code: 400
    message: url required
httpClient:
  method: GET
  url: "{{ get('url') }}"
```

</div>

- `requires:` — run these first; read their output with `get('id')`
- `items:` — run once per item (fan-out)
- `before` / `after` — expression hooks around the action
- `apiResponse:` — may sit with another action to shape the HTTP reply

## Built-in actions

| Key | Role |
|-----|------|
| `chat` | LLM call — reply at `.message.content` |
| `httpClient` | HTTP request |
| `sql` | Database query |
| `python` | Python script (stdout, often JSON) |
| `exec` | Shell command |
| `file` | Read / write / list / delete |
| `git` | Status, diff, commit, push, pull |
| `codeIntelligence` | Search symbols, defs, diagnostics |
| `scraper` | Fetch URL + CSS selectors |
| `browser` | Playwright automation |
| `searchWeb` | Web search providers |
| `searchLocal` | Local file search |
| `embedding` | Local keyword index |
| `email` | SMTP send / IMAP read |
| `telephony` | Voice / TwiML actions |
| `botReply` | Reply on the inbound bot platform |
| `agent` | Call another agent workflow |
| `loader` | Load docs into chunks (RAG) |
| `vectorStore` | Vector add / similarity search |
| `transcribe` | Speech-to-text |
| `apiResponse` | HTTP response body for the caller |

## Registry components

Installable packs from the registry:

```bash
kdeps registry install <name>
```

Then call them from a resource with `component:` instead of a built-in key. See [Components](/components).

## Chat sketch

<div v-pre>

```yaml
actionId: llm
chat:
  model: llama3.2:1b
  role: user
  prompt: "{{ get('q') }}"
  timeout: 60s
```

</div>

Backend (local vs cloud) comes from global config, not from each resource — unless you override it for a profile. See [Config](/config).
