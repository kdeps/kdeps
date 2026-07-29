# Quickstart

**Agent first.** Then workflow if you need a fixed API.

## 1. Coding agent

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
# or: brew install kdeps/tap/kdeps

kdeps --version    # latest release: 2.1.11
kdeps              # REPL — local model by default
```

Talk to the model. Use `/help`, `/model list`, `!make test`.

### Agent on a project

```bash
cd ~/Projects/acme
kdeps .            # workflows under . become tools
```

Or a single agent dir:

```bash
kdeps ./my-agent/
```

Useful:

```bash
kdeps --model llama3.2:1b --system "You are a careful code reviewer."
kdeps --resume <session-id>
kdeps --skill ~/.kdeps/skills/
```

Details: [Coding agent](/agent) · [CLI](/cli).

## 2. Workflow API (when you need one)

```bash
kdeps new my-agent
cd my-agent
kdeps validate .
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml --dev
```

```bash
curl -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q":"hello"}'
```

Scaffold layout:

```text
my-agent/
├── workflow.yaml
└── resources/
```

`requires:` orders steps; `get('id')` reads outputs. Full shape: [Workflow mode](/workflow).

### Run flags (workflow)

```bash
kdeps run workflow.yaml --port 16395
kdeps run workflow.yaml --events
kdeps run workflow.yaml --interactive   # workflow + agent REPL
kdeps run workflow.yaml --file ./doc.txt
```

## 3. Ship later

```bash
kdeps bundle package .
kdeps bundle build . --tag myregistry/agent:latest
```

[Deploy](/deploy) · [Resources](/resources) · [Config](/config).
