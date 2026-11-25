---
title: Data Sovereignty
description: Run your own LLM and the workflow on servers you control, in the region you choose. Prompts, code, and data stay inside your boundary. No third-party AI provider is in the loop.
---

# Data sovereignty: full text

This page has every step and detail. The [overview](/start/data-sovereignty) is the short version.


With kdeps, you run the full AI stack on servers that you control, in the region that you select. The stack includes the model, the workflow, and the API in front of them. Your prompts, source code, and customer data stay in that boundary. They do not go to a third-party AI provider.

## Where your data goes

```d2
direction: right

W: "Workflows\non your server"
L: "LLM server\nyou run"
D: "Your data\n(files, DBs, repos)"
X: "Third-party hosted AI API" {style.stroke-dash: 4}

W -> L: "/v1 (your network)"
W -> D: tools
W -> X: "off by default" {style.stroke-dash: 4}
```

Everything on the solid path stays inside your boundary. The dashed path exists only when you add a hosted backend to `~/.kdeps/config.yaml`. kdeps never adds it for you.

## Same tools, your server

kdeps is open source. Anyone can run it on their own machine. A public agency or a private institution is one use case.

| Who | Data that stays in | Lead with |
|---|---|---|
| Anyone on their own machine | Their files | A local model |
| Tax, customs, benefits, courts | Citizen records | Their datacenter server |
| Private institutions | Client, member, and student records | Their own server |
| Hospitals and insurers | Patient records | Their datacenter server |
| Banks | Customer and trading data | Their datacenter server |
| Defense and intelligence | Classified work | The air-gapped appliance |

The first row is the default. The other rows use the same workflow with a different boundary.

## Three ways to keep it inside your boundary

### 1. A local model on your own machine

A workflow `chat` with no `model:` runs on PyBot, a local rule-based chatbot. A `model:` alias runs a local llamafile. Neither sends data to an external API.

```bash
kdeps new my-agent --template assistant   # a chat with loop: on a local model
cd my-agent && kdeps run workflow.yaml
```

### 2. Your own LLM server in your own datacenter

Run a standalone OpenAI-compatible inference appliance on a server you control. Then point every workflow host at it. One `base_url` serves all of them. `models:` names one model or several.

```bash
kdeps llm run --engine ollama --model llama3.2 -p 8000        # run it on the server
kdeps llm client-config --url http://192.168.1.50:8000/v1     # prints the block below
```

```yaml
# ~/.kdeps/config.yaml on each workflow host
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1   # your LLM server, your network
  models:
    - llama3.2                            # one model, or several
```

That block is the [client contract](/llm-server/#client-contract). Engines and GPU options are on [kdeps LLM server](/llm-server/).

### 3. Air-gapped appliance

Bundle the workflow, runtime, and model into one artifact and move it onto a network with no internet at all.

```bash
kdeps bundle build         # Docker image with the model pinned inside
kdeps bundle prepackage    # single self-contained binary per architecture
```

See [kdeps deploy](/deploy/) for Docker, Kubernetes, ISO, and binary targets.

## What stays local, what does not

| Item | Where it runs | Leaves your boundary? |
|---|---|---|
| Prompts, code, tool results | Your kdeps host and LLM server | No |
| Model inference | llamafile / Ollama / vLLM / TGI you host | No |
| Agent memory and sessions | `~/.kdeps` on your host | No |
| Model weights, first download | Fetched once from the model host (e.g. HuggingFace) | Yes - weights only, no prompts. Pre-stage or bundle them to avoid it. |
| Hosted backends (`openai`, `anthropic`, ...) | Provider's servers | Yes - only if you configure one |

kdeps itself contains no usage telemetry.

## Compliance checklist

- Keep `backend:` under `llm:` on a local engine or on `openai` with a `base_url` you control.
- Never set a hosted-provider API key on a machine that handles regulated data.
- Pin the model inside the image with `kdeps bundle build` so a rebuild never re-downloads weights.
- Host the LLM server in the region your data-residency rules require, and put the kdeps hosts in the same region.
- Commit the YAML to a git host you control - the agent's behavior is auditable text, not a vendor console.

## See also

- [Local models](/start/local-models) - llamafile and Ollama, fully offline
- [kdeps LLM server](/llm-server/) - run your own inference appliance
- [Agent loop](/workflow/agent-loop/) - a chat that calls tools, on that server
- [kdeps deploy](/deploy/) - Docker, Kubernetes, ISO, single binary
- [Why kdeps?](/start/why-kdeps) - the case for self-hosted, git-native AI
