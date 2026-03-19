# Shared Workflow Example

Demonstrates the **workflow import** pattern: a base workflow defines shared
resources (e.g. authentication) that any other workflow can inherit by
listing it under `metadata.workflows`.

## What it does

`POST /api/v1/chat` requires a valid `Authorization: Bearer <token>` header.
The auth check is defined **once** in `auth-base/` and reused by `chat-agent/`
— no copy-paste.

```
Authorization: Bearer my-token
{"message": "What is Kubernetes?"}
→ {"success": true, "data": {"reply": "..."}}

Authorization: (missing)
→ 401 Unauthorized
```

## Directory layout

```
shared-workflow/
├── auth-base/                   # base workflow — auth resources only
│   ├── workflow.yaml
│   └── resources/
│       └── auth-check.yaml      # actionId: authCheck
└── chat-agent/                  # entry-point workflow
    ├── workflow.yaml            # imports @auth-base
    └── resources/
        ├── chat.yaml            # requires: [authCheck]  ← from auth-base
        └── chat-response.yaml
```

## How the import works

```yaml
# chat-agent/workflow.yaml
metadata:
  name: chat-agent
  workflows:
    - "@auth-base"   # ← strip "@", look for sibling dir "auth-base/"
```

The parser resolves `@auth-base` by looking for a sibling directory named
`auth-base` (or a file `auth-base.yaml`), parses it, and **prepends** its
resources to `chat-agent`'s resource list.  If `chat-agent` defines a
resource with the same `actionId` as one from `auth-base`, the **local
definition wins**.

After merging, `chat-agent` effectively runs:

```
authCheck  →  chat  →  chatResponse
```

## Running

```bash
kdeps run examples/shared-workflow/chat-agent/workflow.yaml
```

Authenticated request:

```bash
curl -s -X POST http://localhost:16395/api/v1/chat \
  -H 'Authorization: Bearer my-secret-token' \
  -H 'Content-Type: application/json' \
  -d '{"message": "What is Kubernetes?"}' | jq .
```

Unauthenticated request (should return 401):

```bash
curl -s -X POST http://localhost:16395/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"message": "What is Kubernetes?"}' | jq .
```

## Import rules

| Rule | Behaviour |
|------|-----------|
| Local resource wins | If `chat-agent` defines `actionId: authCheck`, it overrides the imported one |
| Order matters | `workflows: ["@a", "@b"]` — `@a` resources are prepended first, then `@b` |
| Transitive imports | `@b` can itself import `@c`; resolved recursively |
| Circular imports | Detected and reported as an error |
| Path resolution | `@name` → sibling dir `name/` → `name.yaml` → `name.yml` |

## Requirements

- [Ollama](https://ollama.com) installed and running (or set `installOllama: true` to have kdeps install it)
- `llama3.2:1b` model pulled: `ollama pull llama3.2:1b`
