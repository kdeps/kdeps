# Why kdeps?

kdeps exists because most AI tooling is built for prototyping, not for running unattended in production.

## Three ways to use kdeps

You don't need Docker or a workflow file to start. kdeps works at three levels of investment:

**1. Local AI agent** - run `kdeps` in your terminal right now

```bash
kdeps                            # open-source AI agent REPL, zero config
kdeps --model llama3.2           # swap to any local or cloud model
kdeps serve ./my-workflow/       # load your workflows as tools
```

Works with any model: local llamafile (default, no API key), Ollama, or any cloud provider. See [Run Locally in 30 Seconds](/getting-started/local-agent).

**2. Workflow runner** - define what the agent does in YAML, run it locally or share it

```bash
kdeps run workflow.yaml          # run a workflow as a one-shot pipeline
kdeps serve ./my-agent/          # serve it as tools in the agent REPL
```

One file describes inputs, resources, and outputs. Run it on your laptop or on a server - same file, same behavior. See [Quickstart](/getting-started/quickstart).

**3. Production API** - deploy to Docker, Kubernetes, or a standalone binary

```bash
kdeps build                      # package workflow + model into a Docker image
docker run -p 16395:16395 ...    # serve as an HTTP API
```

The workflow you ran locally becomes a self-contained deployable unit. See [Deployment Guide](/guides/deployment-guide).

---

## The problem

Shipping AI into production means more than calling an API. You need deterministic pipelines, typed inputs and outputs, dependency ordering, retries, validation, and the ability to deploy anywhere -- not a chat session that ends when the browser tab closes.

kdeps is an **AI appliance builder**. You define what the agent does in YAML, and it runs as a self-contained unit -- an HTTP API, a bot, a file processor -- without a human in the loop.

## Two modes, one workflow file

```d2
direction: right

A: workflow.yaml
B: "kdeps run\nworkflow mode\nDAG pipeline, deterministic\nships to production"
C: "kdeps serve\nagent mode\ninteractive LLM loop\ntools on demand"

A -> B
A -> C
```

Workflow mode is for production: inputs are validated, resources execute in a fixed order, output is predictable and auditable. Agent mode is for exploration: the LLM decides which workflows to call and in what order, with each workflow running as a complete pipeline.

The same `workflow.yaml` works in both. You do not need to rewrite anything to switch.

## Defined control flow

Chat interfaces are deliberately open-ended. kdeps workflow mode is the opposite: inputs are declared, dependencies are explicit, and validations fire before any LLM is called. If the input is wrong, the workflow fails fast with a clear error instead of hallucinating a response.

This is what makes workflow output reproducible, auditable, and safe to run unattended.

## Agencies

Single-agent workflows have limited scope. kdeps [agencies](/reference/glossary#agency) let you compose multiple specialized agents into a single system. Each agent has its own model, resources, and logic. They communicate via the `agent:` resource type, which runs another agent's full pipeline and returns its output -- every step is version-controlled, testable, and independently deployable.

## Built to last

Most AI tooling has a short half-life. A workflow written against a popular AI SDK in 2023 is unlikely to run without modification today. Model APIs deprecate. SDK interfaces churn. Libraries get abandoned.

kdeps is designed around a different premise: a workflow you deploy today should still be running in 10-15 years.

Three properties make this possible:

- **Versioned schema.** Every `workflow.yaml` declares `apiVersion: kdeps.io/v1`. Breaking changes ship under a new API version. Your existing workflows do not move until you explicitly migrate.
- **Local LLMs never break underneath you.** If you use Ollama or any self-hosted model, the interface does not change unless you update it. No vendor deprecation notices, no sunset dates.
- **Backend decoupled from workflow.** Cloud model names live in `~/.kdeps/config.yaml`, not in `workflow.yaml`. When a model is deprecated, you change one line in config. The workflow is untouched.

The two resource types coupled to external systems that can change on their own timeline:

| Resource | Risk | Mitigation |
|---|---|---|
| `httpClient:` | External APIs change schema, auth, endpoints | Always target a versioned path (`/v2/users`, not `/users`) |
| `browser:` | Website DOM changes without notice | Use stable selectors (ARIA roles, data attributes) over structural CSS |

Everything else -- SQL, LLM prompts, Python, exec, email, inter-agent calls -- is code you own. It changes when you change it.

A company that commissions a kdeps agent today can hand the YAML files and Docker image to a new engineer in 2035 and expect it to still run.

## Who it is for

| Role | Use case |
|------|----------|
| Developers | Ship AI features into products (APIs, bots, internal tools) without glue code |
| Operations teams | Automate repetitive work: reports, triage, data entry, document processing |
| Marketing and growth | Content pipelines, SEO automation, campaign reporting |
| Any team | Replace a human clicking through tabs and copy-pasting between tools |


## See Also

- [Quick Start](/getting-started/quickstart) - Build your first workflow in minutes
- [Workflow Mode](/modes/workflow-mode) - Deterministic DAG pipelines
- [Agent Mode](/modes/agent-loop-mode) - Autonomous LLM loop
