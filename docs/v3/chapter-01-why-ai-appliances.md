{sample: true}
# Chapter 1: Why AI Appliances?

## The Prototype Problem

Most AI tooling is built for the demo, not the shift.

You open a Jupyter notebook, call an LLM API, get back something impressive, and suddenly someone wants to "put it in production." Then the questions start: How do you validate inputs before wasting an API call? How do you retry a flaky embedding service? How do you make the output deterministic enough to audit? How do you deploy it somewhere that isn't your laptop? How do you switch from OpenAI to a self-hosted Ollama instance without rewriting half the codebase?

The prototype has no answers to these questions because it was never designed with them in mind.

This is not a failure of the developers who wrote the prototype. It is a failure of the tools they used. Chat interfaces, notebook environments, and AI SDK wrappers are all optimized for fast exploration. They are deliberately open-ended. That is exactly what you want when you are figuring out whether an idea works. It is exactly what you do not want when an idea has proven out and needs to run unattended at 3am.

**kdeps** was built to close this gap.

## The AI Appliance Model

An appliance has a defined function. A dishwasher washes dishes. It does not sometimes wash dishes and sometimes give you a philosophical reflection on cleanliness. Its behavior is declared in its design, tested at the factory, and predictable every time you run it.

kdeps brings the same discipline to AI systems. You define what your agent does in YAML — its inputs, its processing steps, its outputs, and its deployment target. kdeps runs it as a self-contained unit: an HTTP API, a Telegram bot, a file processor, a scheduled report generator. There is no human in the loop. It just runs.

The insight behind this model is that **AI is a step in a pipeline, not the whole pipeline**. A useful AI system almost always needs to validate inputs, fetch data, call external services, store results, and return a structured response. The LLM call is one node in a dependency graph. kdeps makes you declare that entire graph explicitly, in YAML, and then runs it in a deterministic order.

## What Makes Production AI Hard

Building AI for production requires solving problems that have nothing to do with model quality:

**Typed, validated inputs.** An LLM that receives malformed input will hallucinate a response rather than fail cleanly. Production systems need to validate at the boundary — before a single token is generated — and return structured errors when inputs are wrong.

**Dependency ordering.** Step B often depends on the output of Step A. When A is an LLM call and B is a database write, you need a mechanism to declare that dependency explicitly and enforce it at runtime.

**Reproducibility.** If the same input can produce wildly different processing paths, debugging is nearly impossible and auditing is out of the question. Production pipelines need to be predictable.

**Backend independence.** If your pipeline is tightly coupled to OpenAI's API, you cannot switch to a self-hosted model without a rewrite. This is both a cost risk and a reliability risk.

**Deployment flexibility.** AI workloads run in many environments: developer laptops, Docker containers, Kubernetes clusters, air-gapped servers, edge devices. Your agent definition should not change based on where it deploys.

kdeps addresses all five of these directly:

| Problem | kdeps answer |
|---|---|
| Typed inputs | `validations:` block; workflow fails fast with structured error |
| Dependency ordering | `requires:` DAG; resources run in dependency order, never before deps are resolved |
| Reproducibility | Workflow mode: fixed execution order, all inputs declared |
| Backend independence | `~/.kdeps/config.yaml` sets the backend; `workflow.yaml` sets the model name |
| Deployment flexibility | `kdeps bundle build` (Docker), `kdeps export k8s` (Kubernetes), `kdeps bundle prepackage` (binary), ISO export |

## Two Modes, One Workflow File

kdeps has two operating modes. You choose the mode when you run the command, not when you write the workflow.

**Workflow mode** (`kdeps run`) is for production pipelines. Inputs come in via HTTP, the resource DAG executes in dependency order, a terminal resource returns the response. Every step is predictable. If validation fails, the pipeline aborts with a clear error. This mode is what you ship.

**Agent mode** (`kdeps [path]`) is for interactive, autonomous operation. The LLM is in the loop. Every workflow you point at is registered as a callable tool. The model decides which tools to invoke and in what order, based on the user's prompt. This mode is what you use when you want the LLM to decide the processing path.

The critical point: **you do not write different workflow files for the two modes**. The same `workflow.yaml` works in both. Switching from deterministic pipeline to autonomous agent is a change to the run command, not to your workflow definition.

## Who Is kdeps For?

kdeps is for technical people who are responsible for what they ship:

| Role | Typical use case |
|---|---|
| Backend developers | Shipping AI features into products — APIs, bots, internal tools — without glue code |
| Platform / DevOps engineers | Deploying and operating AI workloads on existing infrastructure (Docker, K8s, bare metal) |
| Operations teams | Automating repetitive, judgment-intensive work: report generation, triage, data entry, document processing |
| Full-stack developers | Building products where the AI component is one part of a larger system |
| Anyone inheriting a prototype | Turning "it works on my machine" into something with an SLA |

If you have never shipped AI to production and are trying to learn how, this book is for you. If you have shipped AI to production and are tired of the accidental complexity, this book is also for you.

## No Lock-In, By Design

kdeps is Apache 2.0 licensed. Your workflows are standard YAML files in a git repository. There is no kdeps cloud, no managed service, no proprietary runtime you are required to run. If you stop using kdeps tomorrow, you have a directory of YAML files that clearly document what your agent does. There is no migration script you need to run.

The framework separates the workflow definition (what the agent does) from the backend configuration (who runs the LLMs). Out of the box every model runs locally as a llamafile — no server install, no API key. You can switch to Ollama, OpenAI, Anthropic, Groq, or any OpenAI-compatible API. You switch backends by editing one line in a config file. Your workflow files do not change.

This is a deliberate design choice. The value of an AI appliance is that it does a specific thing reliably. That value should not be held hostage by a vendor relationship.

## Built to Last

Most AI tooling has a short half-life. A LangChain workflow written in 2023 is unlikely to run without modification today. Model APIs deprecate. SDK interfaces churn. Libraries that were "the standard" get abandoned or forked into incompatible branches.

kdeps is designed around a different premise: a workflow you deploy today should still be running in 10 to 15 years.

Three properties make this possible.

**The schema is versioned.** Every `workflow.yaml` begins with `apiVersion: kdeps.io/v1`. Future breaking changes ship under a new API version. Your existing workflows do not move until you explicitly migrate. A junior engineer reading a kdeps YAML file in 2040 will see the same structure you wrote today.

**Local LLMs never break underneath you.** If you deploy your agent with a local or edge model (Ollama, llama.cpp, any OpenAI-compatible endpoint you control), the model interface does not change unless you update it. There is no vendor deprecation notice, no sunset date, no quota. The model you ran in 2025 is still on disk in 2040 if you want it.

**The workflow is decoupled from the backend.** Cloud LLM model names live in `~/.kdeps/config.yaml`, not in `workflow.yaml`. When a model is deprecated, you change one line in config. The workflow file does not change.

### Two Things That Can Break

No abstraction is infinite. Two resource types in kdeps are coupled to external systems that change on their own timeline:

| Resource | What can change | How to protect yourself |
|---|---|---|
| `httpClient:` | External APIs change their schema, auth, and endpoints | Always target a versioned API path (e.g. `/v2/users`, not `/users`). Pin the API version in the URL. |
| `browser:` | Website DOM structure changes without notice | Use stable selectors (data attributes, ARIA roles) over structural CSS paths. Test selectors in CI. |

Everything else -- SQL schemas, LLM prompts, Python scripts, exec commands, email, inter-agent calls -- is code you control. It changes when you change it, and not before.

This is what it means to build an AI appliance: you hand a company a YAML file and a Docker image. They run it in 2025. They run the same thing in 2035. The only calls they need to make are to you, not to a platform that no longer exists.

## If You Are Coming From Another Framework

If you have used LangChain, LlamaIndex, or OpenAI Assistants, the concepts in kdeps map closely to ones you already know — they just have different names and a different philosophy about where logic lives.

### LangChain

| LangChain concept | kdeps equivalent | Key difference |
|---|---|---|
| Chain | Workflow (DAG of resources) | kdeps chains are declared as `requires:` edges, not code |
| Agent | Agent mode (`kdeps [path]`) | The LLM invokes whole workflows as tools, not individual Python functions |
| Tool | Workflow registered as a tool | A kdeps "tool" is a full YAML workflow, not a function definition |
| Memory | Session store (`set(..., 'session')`) | Sessions are HTTP cookies; no in-memory state |
| Retriever | `searchLocal:` or `embedding:` resource | First-class resource types, not objects you construct in code |
| PromptTemplate | String with `&#123;&#123; expr &#125;&#125;` interpolation | Expressions draw from the data store, not Python variables |
| LLM | `chat:` resource | Backend configured in `~/.kdeps/config.yaml`, not in the chain |
| OutputParser | `jsonResponse: true` on `chat:` | Schema enforcement built into the resource, not a separate step |

**The biggest shift:** In LangChain, you write Python that constructs and connects objects. In kdeps, you write YAML that declares what should happen and lets kdeps determine the execution order. There is no Python object to instantiate, no `chain.run()` to call.

### LlamaIndex

| LlamaIndex concept | kdeps equivalent | Key difference |
|---|---|---|
| QueryEngine | Workflow with `searchLocal:` + `chat:` | Static pipeline declared in YAML |
| Index | `embedding:` resource (index operation) | Index is a resource step, not an object you maintain |
| Retriever | `embedding:` resource (search operation) | Same resource type, different `operation:` field |
| Node | Resource output stored in data store | Accessed via `get('actionId')` |
| Pipeline | Workflow DAG | Topology declared via `requires:`, not code |
| StorageContext | `sqlConnections:` + session store | Storage configured in `workflow.yaml`, not in code |

**The biggest shift:** LlamaIndex gives you fine-grained control over indexing and retrieval internals. kdeps abstracts those into an `embedding:` resource and lets you focus on the pipeline topology. If you need sub-chunk-level retrieval control, kdeps calls out to your existing embedding infrastructure via `httpClient:`.

### OpenAI Assistants API

| Assistants concept | kdeps equivalent | Key difference |
|---|---|---|
| Assistant | Agent mode workflow | Defined in YAML, not via API call; no vendor storage |
| Thread | Session (sqlite or postgres) | You own the storage; no OpenAI thread retention |
| Message | `get('message')` in bot mode, `get('q')` in API mode | Consistent regardless of platform |
| Run | Workflow execution | One HTTP request = one workflow execution |
| Tool / Function | Workflow registered as a tool | A whole workflow, not a JSON schema + Python function |
| File search | `searchLocal:` or `embedding:` | You control the index; no OpenAI vector store |
| Code interpreter | `python:` or `exec:` resource | Runs in your environment, not OpenAI's sandbox |

**The biggest shift:** OpenAI Assistants live on OpenAI's servers. You interact with them via API. kdeps agents live on your infrastructure. You run them. The capability is similar; the operational model is entirely different.

### Mental Model for All Three

If you have used any of these frameworks, the mental model for kdeps is:

> Replace every Python class or function call with a YAML resource file. Replace every `.run()` or `.invoke()` with `requires:`. The LLM call is one resource in a graph, not the center of the universe.

The first time you write a kdeps workflow, it feels like you have less control than Python gives you. The payoff is that the workflow is now a plain text file, testable with curl, deployable with a single command, and readable by anyone on your team — not just the person who wrote the Python.

## What You Will Build in This Book

By the time you finish this book, you will have built and deployed:

1. A simple LLM API that validates inputs and returns structured JSON
2. A multi-resource workflow that chains an HTTP call, an LLM call, and a database write
3. An autonomous agent that decides its own tool-calling path based on user prompts
4. A multi-agent agency where specialized agents collaborate on a complex task
5. All of the above packaged as a Docker image, a Kubernetes deployment, and a standalone binary

Let's get to work.

X> ## Exercise
X>
X> Take an AI prototype you have built (or imagine a simple one: a chatbot that answers questions about a product). Write down the answers to these five questions about it:
X>
X> 1. How does it validate that the user's input is not empty or malicious before sending it to the LLM?
X> 2. What happens if the LLM API is down — does it fail silently or return a useful error?
X> 3. How would you deploy it so it runs unattended on a server, not your laptop?
X> 4. If you needed to switch from OpenAI to a self-hosted model tomorrow, how many files would you change?
X> 5. How do you know if a request failed in production three days ago?
X>
X> For each question where the answer is "I don't know" or "it would break," that is a gap the AI appliance model fills. Keep this list — you will build the solutions to each item over the course of the book.
X>
X> **Stretch goal:** Read the kdeps documentation home page at kdeps.com and identify which of your five gaps maps to which kdeps feature.
