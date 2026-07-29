# Chapter 13: Agencies — Multi-Agent Systems

An **agency** is a collection of kdeps agents that cooperate to handle complex tasks. Where a single agent handles one workflow, an agency coordinates multiple specialized agents — each independently deployable and testable — into a unified system that can tackle multi-step problems autonomously.

This is the architecture for serious AI automation: not a monolithic agent trying to do everything, but a team of specialists coordinated by an orchestrating agent.

## Why Agencies

Single-agent limitations become apparent quickly:

- All resources are coupled in one workflow file
- The LLM context window fills up with unrelated capabilities
- You cannot reuse an agent's logic in a different project without copying files
- Testing one capability requires running the entire agent

Agencies solve this by making agents composable:

| Single Agent | Agency |
|---|---|
| One workflow file | Multiple specialized agents with clear boundaries |
| All resources coupled | Each agent independently deployable and testable |
| Hard to reuse logic | Agents packaged as `.kdeps` archives and reused |
| No inter-agent delegation | Agents delegate to each other via `agent:` resource |
| Limited scope | Self-governing system handles end-to-end tasks |

## Directory Structure

```
my-agency/
├── agency.yaml               # agency manifest — describes the whole system
└── agents/
    ├── greeter/
    │   ├── workflow.yaml     # entry-point agent
    │   └── resources/
    │       ├── understand.yaml
    │       └── respond.yaml
    ├── researcher/
    │   ├── workflow.yaml
    │   └── resources/
    │       ├── search.yaml
    │       ├── scrape.yaml
    │       └── summarize.yaml
    └── writer/
        ├── workflow.yaml
        └── resources/
            ├── draft.yaml
            └── polish.yaml
```

Each agent directory is a complete, standalone kdeps workflow. Every agent can be run independently with `kdeps run agents/researcher/`. The `agency.yaml` manifest defines how they work together.

## The Agency Manifest

```yaml
# agency.yaml
apiVersion: kdeps.io/v1
kind: Agency

metadata:
  name: my-agency
  version: "1.0.0"
  description: "A multi-agent research and writing system"
  targetAgentId: greeter-agent    # entry point; must match metadata.name of an agent

agents:
  - agents/greeter          # directory-based agent
  - agents/researcher       # directory-based agent
  - agents/writer           # directory-based agent
```

**`targetAgentId`** is the entry-point agent — the one that receives the initial request. It must match the `metadata.name` of one of the listed agents.

**`agents:`** lists the agent directories. If omitted, kdeps auto-discovers all `agents/` subdirectories and `.kdeps` archives.

### Auto-Discovery Mode

```yaml
# agency.yaml (minimal)
apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: my-agency
  version: "1.0.0"
  targetAgentId: greeter-agent
# agents: omitted — auto-discovers everything in agents/
```

With auto-discovery, you drop new agents into `agents/` and they are automatically included. No manifest update needed.

## Calling One Agent from Another

The `agent:` resource type delegates a task to another agent:

```yaml
# agents/greeter/resources/delegate-research.yaml
actionId: delegateResearch
requires: [understand]
agent:
  target: researcher-agent          # metadata.name of the target agent
  input: "&#123;&#123; get('userQuery') &#125;&#125;"   # passed as the target agent's 'input'
```

When this resource executes:
1. The `researcher-agent` workflow is loaded
2. It receives `input` as if it came from an HTTP request body
3. Its full resource DAG executes
4. `apiResponse.response` from the researcher is returned as the `delegateResearch` resource's output

The caller agent reads the result:

```yaml
# agents/greeter/resources/delegate-writing.yaml
actionId: delegateWriting
requires: [delegateResearch]
agent:
  target: writer-agent
  input: "&#123;&#123; get('delegateResearch') &#125;&#125;"   # passes researcher's output to writer
```

This is just another `requires:` dependency. The agency orchestration is declared in YAML, not in code.

## A Complete Multi-Agent Example

**Scenario:** Users ask the agency a research question. The greeter understands the question, the researcher gathers information, the writer produces the final answer.

### agent/greeter/workflow.yaml

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: greeter-agent
  description: "Entry point: understands user requests and orchestrates research and writing"
  targetActionId: respond
settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/ask
        methods: [POST]
```

### agents/greeter/resources/understand.yaml

```yaml
actionId: understand
validations:
  check:
    - get('q') != ''
  error:
    code: 400
    message: "question 'q' is required"
chat:
  model: llama3.2:1b
  systemPrompt: |
    Extract a concise research query from the user's question.
    Return only the search query, no explanation.
  prompt: "User question: &#123;&#123; get('q') &#125;&#125;"
```

### agents/greeter/resources/research.yaml

```yaml
actionId: research
requires: [understand]
agent:
  target: researcher-agent
  input: "&#123;&#123; get('understand') &#125;&#125;"
```

### agents/greeter/resources/write.yaml

```yaml
actionId: write
requires: [research]
agent:
  target: writer-agent
  input: |
    Original question: &#123;&#123; get('q') &#125;&#125;
    Research findings: &#123;&#123; get('research') &#125;&#125;
```

### agents/greeter/resources/respond.yaml

```yaml
actionId: respond
requires: [write]
apiResponse:
  success: true
  response:
    answer: get('write')
    research_summary: get('research')
```

### agents/researcher/workflow.yaml

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: researcher-agent
  description: "Searches the web and summarizes findings"
  targetActionId: summarize
```

### agents/researcher/resources/search.yaml

```yaml
actionId: search
searchWeb:
  query: "&#123;&#123; get('input') &#125;&#125;"
  maxResults: 5
```

### agents/researcher/resources/scrape.yaml

```yaml
actionId: scrape
requires: [search]
scraper:
  url: "&#123;&#123; get('search')[0].url &#125;&#125;"
```

### agents/researcher/resources/summarize.yaml

```yaml
actionId: summarize
requires: [search, scrape]
apiResponse:
  success: true
  response:
    findings: get('scrape')
    sources: map(get('search'), {.url})
```

### agents/writer/workflow.yaml

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: writer-agent
  description: "Takes research and writes a clear, structured answer"
  targetActionId: respond
```

### agents/writer/resources/draft.yaml

```yaml
actionId: draft
chat:
  model: llama3.2:1b
  systemPrompt: "Write clear, factual answers. Use the research provided. Cite sources."
  prompt: "&#123;&#123; get('input') &#125;&#125;"
```

### agents/writer/resources/respond.yaml

```yaml
actionId: respond
requires: [draft]
apiResponse:
  success: true
  response: get('draft')
```

## Running the Agency

```bash
$ kdeps run agency.yaml
```

kdeps loads the agency manifest, discovers all agents, and starts the entry-point agent's HTTP server.

```bash
$ curl -X POST http://localhost:16395/api/v1/ask \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": "What are the main causes of the 2008 financial crisis?"}'
```

The request flows: greeter → researcher → writer → greeter → caller. The caller sees only the final response. All inter-agent delegation is transparent.

## Packed Agent Archives

Individual agents can be pre-packaged as `.kdeps` archives and used in an agency without unpacking:

```yaml
agents:
  - agents/greeter
  - agents/researcher
  - packaged-writer-2.1.0.kdeps    # packed archive
```

This lets you:
- Use versioned, shared agents from your organization's artifact repository
- Combine in-development agents (directories) with stable released agents (archives)
- Publish agents as reusable libraries

Pack an individual agent:

```bash
$ kdeps bundle package agents/writer/workflow.yaml
# Creates writer-agent-1.0.0.kdeps
```

## Packaging the Entire Agency (.kagency)

To ship a complete agency as a single portable file, pack the whole `my-agency/` tree into a `.kagency` archive:

```bash
$ kdeps bundle package my-agency/
# Creates my-agency-1.0.0.kagency
```

The `.kagency` archive contains `agency.yaml` plus all `agents/` sub-trees. It can be used directly everywhere a directory would be used:

```bash
# Run the full agency from the archive
$ kdeps run my-agency-1.0.0.kagency

# Build a Docker image (entry-point agent becomes the container)
$ kdeps bundle build my-agency-1.0.0.kagency --tag myregistry/my-agency:latest

# Export a bootable ISO
$ kdeps export iso my-agency-1.0.0.kagency --output my-agency.iso

# Embed in a standalone binary
$ kdeps bundle prepackage my-agency-1.0.0.kagency --output dist/
```

The difference in a nutshell:

| Archive | Contains | Run command |
|---|---|---|
| `.kdeps` | Single agent (workflow + resources) | `kdeps run myagent-1.0.0.kdeps` |
| `.kagency` | Full agency (agency.yaml + all agents) | `kdeps run my-agency-1.0.0.kagency` |

Use `.kdeps` when you want to package and share an individual agent. Use `.kagency` when you want to ship the entire multi-agent system as one deployable unit.

## Independent Deployment

Every agent in an agency can be deployed independently:

```bash
# Run the researcher agent standalone
$ kdeps run agents/researcher/workflow.yaml

# Run the full agency
$ kdeps run agency.yaml
```

This makes testing straightforward: you can verify each agent's behavior with curl before wiring them together. When debugging agency behavior, you can isolate which agent is producing unexpected output.

## Agency Design Principles

**Specialize agents.** Each agent should have a clear, narrow responsibility. "The researcher" and "the writer" are good agent identities. "The one that does everything" is not.

**Define clear interfaces.** The `input` that passes between agents is a contract. Document what format `input` should be in for each agent in its `metadata.description`. The greeter agent is responsible for passing input in the format the researcher expects.

**Make agents independently testable.** If you cannot test an agent with a simple curl command, its `validations:` are probably not catching bad inputs properly. Fix that before wiring it into an agency.

**Use descriptions for agent mode.** In agent mode, `metadata.description` is what the LLM reads to decide whether to call a given agent. Write descriptions that specify the agent's capability precisely — including what format of input it expects and what format of output it returns.

**Version agents separately.** Agents packaged as `.kdeps` archives have their own versions. A workflow that uses a packed agent pins to a specific version. This means you can upgrade one agent without affecting others.

X> ## Exercise
X>
X> Build a two-agent research agency: one agent that fetches and summarizes a web page, and one orchestrating agent that decides which URLs to fetch based on a user question.
X>
X> 1. Create `agents/fetcher/` — a workflow that accepts a `url` parameter, scrapes the page with `scraper:`, and returns a summary via `chat:`. Write a `metadata.description` that tells the LLM exactly what this agent expects as input.
X> 2. Create `agents/orchestrator/` — a workflow in agent mode (`kdeps [path]`) that receives a user question and calls the `fetcher` agent one or more times to gather information before producing a final answer.
X> 3. Create `agency.yaml` that registers both agents.
X> 4. Run `kdeps agency.yaml` and test:
X> ```bash
X> curl -X POST localhost:16395/api/v1/research \
X>   -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
X>   -H "Content-Type: application/json" \
X>   -d '{"q":"Compare kdeps and LangChain"}'
X> ```
X>
X> Observe from the trace logs which agent was called and how many times.
X>
X> **Stretch goal:** Package the entire agency as a `.kagency` archive with `kdeps bundle package my-research-agency/` and verify it runs from the archive with `kdeps run my-research-agency-1.0.0.kagency`.
