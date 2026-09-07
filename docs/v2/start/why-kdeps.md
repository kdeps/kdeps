# Why kdeps?

kdeps exists because most AI tooling is built for prototyping, not for running unattended in production. This page applies to both workflow mode and agent mode - it explains why the two exist and when to reach for each. New to kdeps? Read [What is kdeps?](/start/) first.

## The problem

There is no streamlined way to build **AI agents and custom APIs - not chatbots** - on open-source, self-hosted LLMs. Every project re-implements the same glue: retrieval, deterministic pipelines, typed inputs and outputs, dependency ordering, retries, validation, and a way to deploy the result anywhere.

kdeps is an **AI Appliance Builder**. You define what the agent does in YAML, and it runs as a self-contained unit - an HTTP API, a bot, a file processor - without a human in the loop. Everything a retrieval-augmented agent needs ships in one Dockerized image.

Because it runs open-source models by default (llamafile, Ollama, any HuggingFace GGUF), the built appliance has **no per-token cost and no AI subscription** - it is free to run forever, on or off the cloud. Cloud providers work too when you want them; the backend is one line of config, not baked into the workflow.

## Three levels of investment

You don't need Docker or a workflow file to start. kdeps works at three levels - a local REPL, a YAML workflow, a deployed appliance - and the [two modes](#two-modes-one-workflow-file) run at every level:

**1. Local AI agent** - run `kdeps` in your terminal right now

```bash
kdeps                            # open-source AI agent REPL, zero config
kdeps --model llama3.2           # swap to any local or cloud model
kdeps ./my-workflow/             # load your workflows as tools
```

Works with any model: local llamafile (default, no API key), Ollama, or any cloud provider. See [Run locally in 30 seconds](/agent/quickstart).

**2. Workflow runner** - define what the agent does in YAML, run it locally or share it

```bash
kdeps run workflow.yaml          # run a workflow as a one-shot pipeline
kdeps ./my-agent/                # load it as tools in the agent REPL
```

One file describes inputs, resources, and outputs. Run it on your laptop or on a server - same file, same behavior. See [Quickstart](/workflow/quickstart).

**3. Production API** - deploy to Docker, Kubernetes, or a standalone binary

```bash
kdeps bundle build .              # package workflow + model into a Docker image
docker run -p 16395:16395 ...     # serve as an HTTP API
```

The workflow you ran locally becomes a self-contained deployable unit. See [Deployment guide](/deploy/).

---

## Deterministic by design

An LLM is probabilistic - ask Claude or GPT the same question twice and you can get two different answers. kdeps workflow mode wraps that in a deterministic shell: the same request always takes the same path through the same resources, `requires:` fixes the order, validations fire before any model is called, and the output is shaped to a fixed schema. The model's wording varies; the pipeline around it does not.

If the input is wrong, the workflow fails fast with a clear error instead of hallucinating a response. Output is reproducible in structure, auditable, and safe to run unattended - that is what makes kdeps suitable for production, not just demos.

Agent mode is the opposite: there the model decides which resources run and in what order.

## Two modes, one workflow file

```d2
direction: right

A: workflow.yaml
B: "kdeps run\nworkflow mode\nDAG pipeline, deterministic\nships to production"
C: "kdeps [path]\nagent mode\ninteractive LLM loop\ntools on demand"

A -> B
A -> C
```

Workflow mode is for production: inputs are validated, resources execute in a fixed order, output is predictable and auditable. Agent mode is for exploration: the LLM decides which workflows to call and in what order, with each workflow running as a complete pipeline.

The same `workflow.yaml` works in both. You do not need to rewrite anything to switch.

## Agencies

Single-agent workflows have limited scope. kdeps [agencies](/reference/glossary#agency) let you compose multiple specialized agents into a single system. Each agent has its own model, resources, and logic. They communicate via the `agent:` resource type, which runs another agent's full pipeline and returns its output - every step is version-controlled, testable, and independently deployable.

## Built to last

Most AI tooling has a short half-life. A workflow written against a popular AI SDK in 2023 is unlikely to run without modification today. Model APIs deprecate. SDK interfaces churn. Libraries get abandoned.

kdeps reduces how many moving parts can break on you. The thing you archive is the **built appliance** - the Docker image, ISO, or self-contained binary you produce with `kdeps bundle`. It pins the kdeps runtime, the executors, and (for local models) the model itself into one frozen unit. Hand that image to a new engineer years later and it runs exactly as it did the day you built it.

What stays stable:

- **Local LLMs do not break underneath you.** With Ollama or any self-hosted model, the interface changes only when you update it. No vendor deprecation notices, no sunset dates.
- **The backend is decoupled from the workflow.** Cloud model names live in `~/.kdeps/config.yaml`, not in `workflow.yaml`. When a model is deprecated, you change one line in config; the workflow is untouched.
- **Your logic is code you own.** SQL, LLM prompts, Python, exec, email, inter-agent calls - these change when you change them.

What you should expect to maintain:

| Thing | Why it moves | What to do |
|---|---|---|
| The `workflow.yaml` schema | kdeps does not carry legacy schema shims - a newer `kdeps` binary may reject an older file. `apiVersion: kdeps.io/v1` marks the current schema, not a frozen one. | Keep the source YAML with the image; re-validate with `kdeps validate` before upgrading the runtime, or just keep running the built image. |
| `httpClient:` targets | External APIs change schema, auth, endpoints | Target a versioned path (`/v2/users`, not `/users`) |
| `browser:` selectors | Website DOM changes without notice | Use stable selectors (ARIA roles, data attributes) over structural CSS |

The guarantee is not that your YAML runs forever on any future kdeps. It is that the appliance you ship is a self-contained unit you can freeze, archive, and redeploy without a live dependency on any vendor.

## Who it is for

| Role | Use case |
|------|----------|
| Developers | Ship AI features into products (APIs, bots, internal tools) without glue code |
| Operations teams | Automate repetitive work: log analysis, PR reviews, triage, ticket creation, incident messaging |
| SMEs without an AI hire | Add AI to existing business processes and APIs with near-zero code and no recurring AI bill |
| Marketing and growth | Content pipelines, SEO automation, campaign reporting |
| Any team | Replace a human clicking through tabs and copy-pasting between tools |

Concretely: log-file analysis, automated code reviews, JIRA ticket creation from an alert, an incident summary posted to MS Teams - each is a workflow that calls a model and one or two external APIs, deployed as one image.

## The name

kdeps is short for **knowledge dependencies**. It grew out of earlier work on [Kartographer](https://github.com/kdeps/kartographer), a graph library for resolving dependent nodes: knowledge - from a model, a machine, or a person - can be represented and orchestrated as a graph. A kdeps workflow is exactly that, a dependency graph of resources, run in order.


## See also

- [Run locally](/agent/quickstart) - agent REPL in 30 seconds
- [Quickstart](/workflow/quickstart) - build your first workflow API
- [Load a workflow as a tool](/workflow/as-a-tool) - same file, agent mode
- [Workflow mode](/workflow/) - deterministic DAG pipelines
- [Agent mode](/agent/) - autonomous LLM loop
