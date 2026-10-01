---
title: Data Sovereignty
description: Run your own LLM and coding agent on servers in your own country. Prompts, code, and data stay inside your jurisdiction - no foreign cloud in the loop.
---

# Data sovereignty

kdeps lets you run the whole AI stack - the model, the coding agent, the API in front of it - on **servers you choose, in the country you choose**. Your prompts, source code, and customer data never have to cross a border or reach a foreign cloud provider.

*Applies to workflow mode, agent mode, and agencies.*

Think of it as the difference between renting an AI from someone else's datacenter and owning an appliance in yours. `kdeps` is the appliance.

## Where your data goes

```d2
direction: right

U: "Your users\nand developers" {shape: oval}
A: "kdeps agent / workflow\non YOUR server"
L: "LLM server\non YOUR server"
D: "Your data\n(files, DBs, repos)"
X: "Foreign hosted AI API" {style.stroke-dash: 4}

U -> A: request
A -> L: "/v1 (local network)"
A -> D: tools
A -> X: "off by default" {style.stroke-dash: 4}
```

Everything on the solid path stays inside your boundary. The dashed path only exists if you add a hosted backend to `~/.kdeps/config.yaml` - kdeps never adds it for you.

## Three ways to keep it in-country

### 1. Coding agent on your own machine

Run `kdeps` and you get an autonomous coding agent REPL - file edits, shell, search, memory - against a local model. Nothing is sent to an external API.

```bash
kdeps                      # agent REPL, default local llamafile model
kdeps --model llama3.1:8b  # bigger local model, still fully local
```

### 2. Your own LLM server in your own datacenter

Provision a standalone OpenAI-compatible inference appliance on a server in your country, then point every kdeps host at it.

```bash
kdeps llm wizard           # pick engine + model, then build/run/export Docker, ISO, or Kubernetes
```

```yaml
# ~/.kdeps/config.yaml on each client machine
llm:
  backend: openai                        # speaks the OpenAI-compatible /v1 API
  base_url: https://llm.internal.example # YOUR server, YOUR network
```

See [kdeps LLM server](/llm-server/) for engines (llamafile, Ollama, vLLM, TGI, and more) and GPU options.

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

- Keep `llm.backend` on a local engine or on `openai` with a `base_url` you control.
- Never set a hosted-provider API key on a machine that handles regulated data.
- Pin the model inside the image with `kdeps bundle build` so a rebuild never re-downloads weights.
- Host the LLM server in the region your data-residency rules require, and put the kdeps hosts in the same region.
- Commit the YAML to a git host you control - the agent's behavior is auditable text, not a vendor console.

## See also

- [Local models](/start/local-models) - llamafile and Ollama, fully offline
- [kdeps LLM server](/llm-server/) - run your own inference appliance
- [kdeps deploy](/deploy/) - Docker, Kubernetes, ISO, single binary
- [Why kdeps?](/start/why-kdeps) - the case for self-hosted, git-native AI
