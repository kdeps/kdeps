# About the Author

**Joel Bryan Juliano** is a Senior Software Engineer with over 20 years of experience building production systems. He has worked on payment platforms at DAZN (scaling to 10M+ daily transactions across 200+ countries), analytics infrastructure at the Belastingdienst (Dutch Tax Authority, handling ~€300B in annual transactions), and security telemetry pipelines at VIPRE Security.

He has been writing open-source software since 2004, contributing to Linux distributions, Ruby on Rails, and dozens of published packages with over 387,000 combined downloads.

Joel is the creator of **kdeps** — the framework this book is about. He built kdeps to solve the problem described in Chapter 1: shipping AI into production reliably, without vendor lock-in, on infrastructure you control.

He is also the author of:

- *From Ruby to Golang: A Ruby Programmer's Guide to Learning Go* (Leanpub, 2020) — Amazon #3 in its category
- *AWS in Production: Building and Operating Real Systems on AWS* (Leanpub, 2026)
- *Kubernetes for All: Build a Datacenter from Scratch* (Leanpub, 2026)
- *Emacs for Life: Build It Yourself, Own It Forever* (Leanpub, 2026)

All books are available at [leanpub.com/u/jjuliano](https://leanpub.com/u/jjuliano).

**Website:** [joeljuliano.com](https://joeljuliano.com)
**GitHub:** [github.com/jjuliano](https://github.com/jjuliano)
**LinkedIn:** [linkedin.com/in/joeljuliano](https://linkedin.com/in/joeljuliano)

---

# Resources

**kdeps documentation:** [kdeps.com](https://kdeps.com)

**kdeps component registry:** [kdeps.io](https://kdeps.io)

**kdeps GitHub repository:** [github.com/kdeps/kdeps](https://github.com/kdeps/kdeps)

**Report issues:** [github.com/kdeps/kdeps/issues](https://github.com/kdeps/kdeps/issues)

---

# Quick Reference

## Key Commands

```bash
# Create a project
kdeps new my-agent

# Run in workflow mode
kdeps run workflow.yaml
kdeps run workflow.yaml --dev        # hot reload
kdeps run workflow.yaml --instrument  # call-chain instrumentation tracing

# Run in agent mode
kdeps ./my-agent/
kdeps ./agents/                      # folder mode

# Package and deploy
kdeps bundle package workflow.yaml   # create .kdeps archive
kdeps bundle build myagent.kdeps --tag myregistry/myagent:latest   # Docker
kdeps export k8s ./my-agent --output k8s.yaml                      # Kubernetes
kdeps bundle prepackage myagent.kdeps                               # standalone binary

# Validate and diagnose
kdeps validate workflow.yaml
kdeps doctor

# Components
kdeps registry install scraper
kdeps registry list
kdeps registry uninstall scraper
```

## Resource Types Summary

| Resource | Field | Purpose |
|---|---|---|
| LLM | `chat:` | Call a language model |
| HTTP Client | `httpClient:` | Make outbound HTTP requests |
| SQL | `sql:` | Run database queries |
| Python | `python:` | Execute Python scripts |
| Shell | `exec:` | Run shell commands |
| Scraper | `scraper:` | Fetch and extract page text |
| Web Search | `searchWeb:` | Search the web |
| Local Search | `searchLocal:` | Search local files |
| Embeddings | `embedding:` | Index/search a text store |
| Browser | `browser:` | Drive a real browser |
| Component | `component:` | Invoke a reusable bundle |
| Agent | `agent:` | Delegate to another agent |
| File | `file:` | Read, write, patch, list, and manage files |
| Git | `git:` | Status, diff, log, commit, push, pull, and branch operations |
| Code Intelligence | `codeIntelligence:` | Symbol search, definitions, references, and diagnostics |
| Loader | `loader:` | Load PDF, HTML, CSV, text, or a directory into text chunks for RAG |
| Vector Store | `vectorStore:` | Add and similarity-search documents (Qdrant, Chroma, Pinecone, pgvector, and more) |
| Transcribe | `transcribe:` | Speech-to-text via Whisper (OpenAI, Groq, or local) |
| Response | `apiResponse:` | Build HTTP response (terminal) |

## Expression Quick Reference

```yaml
# Read values
get('key')               # from data store / request body
get('resource').field    # field access
get('list')[0]           # array index
info('ID')               # request ID
env('ENV_VAR')           # environment variable

# Write values
set('key', value)                  # request-scoped
set('key', value, 'session')       # session-scoped

# Strings
trim(get('q'))
lower(get('name'))
upper(get('code'))
len(get('text'))
split(get('csv'), ',')
join(get('array'), ', ')
get('text')[0:500]
get('text') contains 'keyword'
get('email') matches '^[^@]+@[^@]+\\.[^@]+$'

# Numbers
int(get('page')) or 1
min(int(get('limit')) or 10, 100)

# Arrays
filter(get('items'), {.active == true})
map(get('results'), {.url})
len(get('list'))
get('list')[0]

# JSON
json({"key": "value"})
fromJSON(get('rawJson'))

# Null safety
get('value') or 'default'
get('obj') != null and get('obj').field != ''
```

## Deployment Comparison

| Target | Command | File | Use case |
|---|---|---|---|
| Local run | `kdeps run` | `workflow.yaml` | Development, testing |
| Docker | `kdeps bundle build` | `.kdeps` → Docker image | Containers, CI/CD |
| Kubernetes | `kdeps export k8s` | `workflow.yaml` → k8s YAML | Cloud infrastructure |
| Binary | `kdeps bundle prepackage` | `.kdeps` → executable | Edge, air-gapped |
| Agent loop | `kdeps [path]` | `workflow.yaml` | Interactive/autonomous |
