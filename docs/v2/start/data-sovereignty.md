---
title: Data Sovereignty
description: Run your own LLM and coding agent on servers you control, in the region you choose. Prompts, code, and data stay inside your boundary - no third-party AI provider in the loop.
---

# Data sovereignty

kdeps lets you run the whole AI stack - the model, the coding agent, the API in front of it - on servers you control, in the region you choose. Your prompts, source code, and customer data never have to leave that boundary or reach a third-party AI provider.

Works in workflow mode, agent mode and agencies.

## Where your data goes

```d2
direction: right

U: "Desktop app and CLI\non each machine" {shape: oval}
W: "Workflows\non your server"
L: "LLM server\nyou run"
D: "Your data\n(files, DBs, repos)"
X: "Third-party hosted AI API" {style.stroke-dash: 4}

U -> L: "/v1 (your network)"
W -> L: "/v1 (your network)"
U -> D: tools
W -> D: tools
U -> X: "off by default" {style.stroke-dash: 4}
W -> X: "off by default" {style.stroke-dash: 4}
```

Everything on the solid path stays inside your boundary. The dashed path only exists if you add a hosted backend to `~/.kdeps/config.yaml` - kdeps never adds it for you.

## Same tools, your server

kdeps is open source. Anyone can run it on their own machine. A public agency or a private institution is one use case.

The same [desktop app](/desktop/) and [coding CLI](/agent/) run for every row below. Point them at a model server you run, or at a local model. The name of the organization does not matter.

Four jobs, in this order:

1. The [LLM server](/llm-server/) they run, on machines they control. One open model, or several.
2. The [desktop app](/desktop/). It talks to that server, or to a local model.
3. The [coding CLI](/agent/). Same server, or the same local model.
4. The rule on this page: prompts, code, and files stay inside that boundary.

| Who | Data that stays in | Lead with |
|---|---|---|
| Anyone on their own machine | Their files | A local model |
| Tax, customs, benefits, courts | Citizen records | Their datacenter server |
| Private institutions | Client, member, and student records | Their own server |
| Hospitals and insurers | Patient records | Their datacenter server |
| Banks | Customer and trading data | Their datacenter server |
| Defense and intelligence | Classified work | The air-gapped appliance |
| A company that keeps customer files off a public chat | Customer files | The desktop, then where the prompt goes |

The first row is the default. The other rows are the same four jobs with a different boundary.

## Desktop app to a private LLM server

The desktop app sends prompts to an LLM server you run. Files stay on the machine where the app is open.

```d2
direction: down

W: "1. On the server\nkdeps llm wizard" {shape: oval}
L: "2. Private LLM server\n/v1 on your network"
Y: "3. Client config\nllm.base_url"
D: "4. Desktop app\non each machine" {shape: oval}
F: "Files and tools\nstay on that machine"

W -> L: "Docker, ISO, or Kubernetes"
L -> Y: "kdeps llm client-config"
Y -> D: "Settings, or config.yaml"
D -> L: "prompts over /v1"
D -> F: "tools"
```

On the server, pick one engine and one model:

```bash
kdeps llm wizard                                              # build, run, or export
kdeps llm run --engine ollama --model llama3.2 -p 8000        # or run it here
kdeps llm client-config --url http://192.168.1.50:8000/v1     # prints the block below
```

On each desktop, open Settings and set `llm.base_url`. Settings writes `~/.kdeps/config.yaml`, the same file the CLI reads. Or paste this:

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1   # that server, your network
  # openai_api_key: "..."                 # only if that server asks for a key
  models:
    - llama3.2                            # the model that server is running
```

Leave `base_url` unset and the desktop keeps its local model. Engines, GPU, and the full contract are on [kdeps LLM server](/llm-server/#client-contract).

## Three ways to keep it inside your boundary

### 1. Coding agent on your own machine

Run `kdeps` and you get an autonomous coding agent REPL - file edits, shell, search, memory - against a local model. Nothing is sent to an external API. The [desktop app](/desktop/) is the same agent in a window, with the same local default.

```bash
kdeps                      # agent REPL, default local llamafile model
kdeps --model llama3.1:8b  # bigger local model, still fully local
```

### 2. Your own LLM server in your own datacenter

Provision a standalone OpenAI-compatible inference appliance on a server you control, then point every client at it. The clients are the [desktop app](/desktop/), the [coding CLI](/agent/), and any workflow host. One `base_url` serves all of them. `models:` names one model or several. The desktop steps are [above](#desktop-app-to-a-private-llm-server).

```bash
kdeps llm wizard           # pick engine + model, then build/run/export Docker, ISO, or Kubernetes
```

```yaml
# ~/.kdeps/config.yaml on each desktop, CLI, and workflow host
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

- Keep `llm.backend` on a local engine or on `openai` with a `base_url` you control.
- Never set a hosted-provider API key on a machine that handles regulated data.
- Pin the model inside the image with `kdeps bundle build` so a rebuild never re-downloads weights.
- Host the LLM server in the region your data-residency rules require, and put the kdeps hosts in the same region.
- Commit the YAML to a git host you control - the agent's behavior is auditable text, not a vendor console.

## See also

- [Local models](/start/local-models) - llamafile and Ollama, fully offline
- [kdeps LLM server](/llm-server/) - run your own inference appliance
- [Desktop app](/desktop/) - the same agent in a window, pointed at that server
- [Coding CLI](/agent/) - the same agent in the terminal, pointed at that server
- [kdeps deploy](/deploy/) - Docker, Kubernetes, ISO, single binary
- [Why kdeps?](/start/why-kdeps) - the case for self-hosted, git-native AI
