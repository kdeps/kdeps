{sample: true}
# Chapter 2: Getting Started

This chapter gets kdeps running and a working LLM API endpoint up in under ten minutes. You will install kdeps, create a project, define two resources, and test the API with curl. There is nothing else to install: the model itself is downloaded automatically the first time you run.

## Installing kdeps

### macOS (Homebrew)

```bash
$ brew install kdeps/tap/kdeps
```

### Linux and macOS (curl)

```bash
$ curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

### Windows (WSL or Git Bash)

```bash
$ wget -qO- https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

For optimal functionality on Windows, run this inside [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) or [Git Bash](https://git-scm.com/downloads/win).

### From Source

If you have Go installed:

```bash
# Recommended
$ go install github.com/kdeps/kdeps/v2@latest

# Or build manually
$ git clone https://github.com/kdeps/kdeps.git
$ cd kdeps
$ go build -o kdeps main.go
```

### Verify the Installation

```bash
$ kdeps --version
kdeps 2.x.x

$ which kdeps
/usr/local/bin/kdeps
```

If `kdeps` is not found after installation, add `~/.local/bin` to your PATH:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

kdeps ships as a single binary. There is no daemon to manage, no background service to start, no configuration to set up before your first project.

## The Local LLM Comes Built In

kdeps needs no LLM server. By default, chat models run as
[llamafiles](https://github.com/Mozilla-Ocho/llamafile) — single
self-contained binaries that bundle the model weights with the inference
runtime. When a workflow references `model: llama3.2:1b`, kdeps downloads the
matching llamafile to `~/.kdeps/models/` (about 1.1 GB, once) and serves it
locally as an OpenAI-compatible endpoint. No install step, no API key, no
network dependency after the first download.

`llama3.2:1b` is a 1-billion parameter model that runs on modest hardware. It
is fast enough to iterate on locally and accurate enough for most workflows.
Quantization variants are part of the alias name — `llama3.2:1b-q6` and
`llama3.2:1b-q8` trade a larger download for better quality:

```bash
$ kdeps llamafile list      # every known model alias
$ kdeps llamafile update    # refresh the registry from HuggingFace
```

If you prefer Ollama, OpenAI, Anthropic, or another provider from the start,
that is a one-line config change. Chapter 6 covers backend configuration in
full.

## Creating Your First Project

```bash
$ kdeps new my-agent
$ cd my-agent
```

This creates the following structure:

```
my-agent/
├── workflow.yaml        # workflow entry point and server config
└── resources/           # one YAML file per resource
```

You can also create this structure manually. kdeps has no magic in the directory layout — it looks for `workflow.yaml` in the directory you point it at, and looks for resource files in `resources/`.

## The Workflow Entry Point

Open `workflow.yaml`. If you used `kdeps new`, you will see a pre-populated file. Let's look at the minimal version:

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: response

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

Four things to note:

**`metadata.name`** — The workflow's identity. Used as the tool name in agent mode. Must be alphanumeric with hyphens.

**`metadata.targetActionId`** — The `actionId` of the terminal resource. kdeps runs the DAG until this resource produces output, then uses that output as the HTTP response. Everything else in the graph is run because this resource depends on it (directly or transitively).

**`settings.apiServer`** — The HTTP server configuration. `hostIp` and `portNum` define where kdeps listens. `routes` declares the valid request paths and methods.

**`apiVersion` and `kind`** — These identify the schema version. Always use `kdeps.io/v1` and `Workflow` for standard workflows.

## Adding an LLM Resource

Create `resources/llm.yaml`:

```yaml
# resources/llm.yaml
actionId: llm
validations:
  methods: [POST]
  routes: [/api/v1/chat]
  check:
    - get('q') != ''
  error:
    code: 400
    message: "'q' is required"
chat:
  model: llama3.2:1b
  role: user
  prompt: "&#123;&#123; get('q') &#125;&#125;"
  timeout: 60s
```

This resource:

1. Only activates on `POST /api/v1/chat` requests (`validations.methods` and `validations.routes`)
2. Checks that the request body contains a non-empty `q` field (`validations.check`)
3. Returns a 400 with a clear message if it does not (`validations.error`)
4. Calls `llama3.2:1b` with the value of `q` as the user prompt (`chat.prompt`)
5. Times out after 60 seconds if the model is unresponsive (`chat.timeout`)
6. Stores the LLM response under the key `llm` (derived from `actionId`)

The `&#123;&#123; get('q') &#125;&#125;` syntax is kdeps expression interpolation. `get('q')` reads from the request body. Chapter 11 covers the full expression language.

## Adding a Response Resource

Create `resources/response.yaml`:

```yaml
# resources/response.yaml
actionId: response
requires: [llm]
apiResponse:
  success: true
  response:
    answer: get('llm')
```

This resource:

1. Declares that it depends on `llm` (`requires: [llm]`)
2. Will not execute until `llm` has produced output
3. Reads `llm`'s output via `get('llm')` and puts it in the response JSON

`apiResponse:` is the terminal resource type. It builds the HTTP response body. Because `workflow.yaml` has `targetActionId: response`, kdeps knows to use this resource's output as the final response.

## Running the Workflow

When `apiServer` is configured, kdeps requires an API auth token before it starts. Set one for local development:

```bash
$ export KDEPS_API_AUTH_TOKEN=dev-token
$ kdeps run workflow.yaml
```

You can also set `api_auth_token` in `~/.kdeps/config.yaml` (Chapter 14). The token never goes in `workflow.yaml`.

You should see:

```
kdeps: starting workflow server on 127.0.0.1:16395
kdeps: registered route POST /api/v1/chat
kdeps: ready
```

In another terminal, test it:

```bash
$ curl -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": "What is entropy?"}'
```

Expected response:

```json
{
  "success": true,
  "response": {
    "answer": "Entropy is a measure of disorder or randomness in a system..."
  }
}
```

Test the validation:

```bash
$ curl -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}'
```

```json
{
  "success": false,
  "error": {
    "code": 400,
    "message": "'q' is required"
  }
}
```

The validation fires before the LLM is called. No tokens were wasted on that request.

## How the Execution Flows

The DAG for this workflow is minimal:

```
POST /api/v1/chat
       |
       v
   [llm resource]
   validates request
   calls llama3.2:1b
   stores output as 'llm'
       |
       v
 [response resource]
 reads get('llm')
 builds HTTP response
       |
       v
  HTTP 200 JSON
```

`requires: [llm]` is the only dependency declaration you wrote. kdeps derives the full execution order from this. When you add more resources, you extend the DAG by adding more `requires:` entries. The framework handles ordering, concurrency, and error propagation.

## Hot Reload for Development

When iterating on resource files, you do not need to restart the server:

```bash
$ kdeps run workflow.yaml --dev
```

`--dev` watches your `resources/` directory and the `workflow.yaml` file. When you save a change, kdeps reloads the workflow without dropping the server. This makes the development loop fast.

## What You Just Built

In ten minutes you have built a validated LLM API with:
- Input validation that fails fast before touching the model
- A structured JSON response format
- A clear dependency declaration between resources
- A server that binds to a specific interface and port

This is the foundation. Every subsequent chapter builds on this structure by adding more resource types, more complex DAGs, autonomous agent behavior, and deployment targets.

In the next chapter, we will take a step back and understand the core concepts that make all of this work.

X> ## Exercise
X>
X> Build the minimal chatbot from this chapter without copying the code directly. Starting from `kdeps new my-first-agent`, recreate the workflow by writing the files yourself:
X>
X> 1. Write `workflow.yaml` with the metadata, `targetActionId`, and a route for `POST /ask`.
X> 2. Write `resources/validate.yaml` with a check that rejects empty `q` parameters.
X> 3. Write `resources/llm.yaml` that uses `llama3.2:1b` and passes the question as the prompt.
X> 4. Write `resources/respond.yaml` that returns the LLM's answer in a `response` field.
X> 5. Run `kdeps validate workflow.yaml` and fix any errors before starting the server.
X> 6. Export `KDEPS_API_AUTH_TOKEN=dev-token`, start the server with `kdeps run workflow.yaml`, and confirm you get a valid JSON response from curl with the `Authorization` header.
X>
X> **Stretch goal:** Change the model to `llama3.2:7b`, restart, and compare the response quality and latency for the same question.
