# kdeps

[![Build and Test](https://github.com/kdeps/kdeps/actions/workflows/build-test.yml/badge.svg?branch=main)](https://github.com/kdeps/kdeps/actions/workflows/build-test.yml)
[![Coverage](https://codecov.io/gh/kdeps/kdeps/branch/main/graph/badge.svg)](https://codecov.io/gh/kdeps/kdeps)
[![Release](https://img.shields.io/github/v/tag/kdeps/kdeps?sort=semver&label=release)](https://github.com/kdeps/kdeps/releases)
[![Go](https://img.shields.io/github/go-mod/go-version/kdeps/kdeps)](https://go.dev/)
[![License](https://img.shields.io/github/license/kdeps/kdeps)](https://github.com/kdeps/kdeps/blob/main/LICENSE)
[![CodeQL](https://github.com/kdeps/kdeps/actions/workflows/codeql.yml/badge.svg?branch=main)](https://github.com/kdeps/kdeps/actions/workflows/codeql.yml)
[![Docs](https://github.com/kdeps/kdeps/actions/workflows/docs.yml/badge.svg?branch=main)](https://kdeps.com)
[![Documentation](https://img.shields.io/badge/docs-kdeps.com-00E5FF)](https://kdeps.com)
[![Registry](https://img.shields.io/badge/registry-kdeps.io-00E5FF)](https://kdeps.io)
[![GitHub stars](https://img.shields.io/github/stars/kdeps/kdeps)](https://github.com/kdeps/kdeps/stargazers)

**Git-native AI appliance builder**

Define your AI agent in YAML. Build it once. Run it where you need it.

Kdeps helps you build AI workflows, agents, and AI services using YAML files stored in your Git repository.

It brings the instructions, tools, and model configuration together into a project you can run locally or package for deployment.

Your YAML files are the source of truth. Review changes in pull requests, track them with Git, and build from a specific commit in your CI pipeline.

## What can you build?

With Kdeps, you can build:

- **AI workflows** — Run tasks in a predictable order.
- **AI agents** — Let an AI model choose which tools to use to complete a task.
- **Agent teams** — Connect specialized agents so they can work together.
- **AI APIs** — Make your AI project available through HTTP endpoints.
- **Local AI services** — Run supported open-source models on machines you control.

You can run a project in a terminal, expose it as an HTTP API, or package it for Docker, Kubernetes, a bootable ISO, or a standalone binary.

The goal is simple: write your AI project’s instructions once, then choose where to run it.

## Why Kdeps?

### Your project lives in Git

Your agent is defined in files you can read, review, and version.

Change the YAML to change how the project works. Use Git history to see what changed and roll back when needed.

### Keep control of your data

Kdeps supports locally hosted models and self-managed inference servers.

When configured to use your own infrastructure, your prompts, source code, and data can stay within your environment instead of being sent to an external AI provider.

Hosted AI services are also supported when you choose to configure them.

### Build and deploy from the same definition

Develop your workflow on a laptop, then package it for a server or another supported environment.

Kdeps supports several deployment targets without requiring you to rewrite the workflow for each one.

### Choose only the parts you need

You can start with one workflow and add agents, an inference server, deployment tools, or shared components as your project grows.

## The main parts

Kdeps brings together a few related tools.

| Part | What it does |
| --- | --- |
| [`kdeps workflow`](https://kdeps.com/workflow/) | Defines and runs tasks in a planned order. |
| [`kdeps agent`](https://kdeps.com/agent/) | Runs an interactive AI agent that can use tools. |
| [`kdeps agencies`](https://kdeps.com/agencies/) | Connects several agents into one system. |
| [`kdeps llm`](https://kdeps.com/llm-server/) | Helps set up a standalone AI inference server. |
| [`kdeps deploy`](https://kdeps.com/deploy/) | Packages projects for supported deployment targets. |
| [`kdeps registry`](https://kdeps.com/registry/) | Finds and shares reusable agents and components. |

The registry is optional. You can build and run your own projects without it.

## Quickstart

### 1. Install Kdeps

**macOS or Linux**

```bash
brew install kdeps/tap/kdeps
```

Or use the installation script:

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

**Windows PowerShell**

```powershell
irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1 | iex
```

Only run installation scripts you trust. See the [installation guide](https://kdeps.com/start/installation) for other options.

### 2. Create a workflow

Save the following as `workflow.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "1.0.0"
  targetActionId: response
settings:
  apiServer:
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
resources:
  - actionId: llm
    name: Ask the AI
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
      prompt: "{{ get('q') }}"
      timeout: 60s
  - actionId: response
    name: Return the answer
    requires: [llm]
    apiResponse:
      success: true
      response:
        answer: get('llm').message.content
```

This workflow accepts a question, sends it to an AI model, and returns the answer.

Here’s what the important parts mean:

- `workflow.yaml` describes the project.
- `resources` lists the tasks the project can run.
- `actionId` gives each task a name.
- `requires` tells Kdeps which task must run first.
- `chat` sends a prompt to an AI model.
- `apiResponse` defines the result returned to the caller.

The workflow uses a planned execution order. The AI-generated answer can vary, but the pipeline determines how the request is processed.

### 3. Start the workflow

Set a token for the HTTP API:

```bash
export KDEPS_API_AUTH_TOKEN=dev-token
```

Then run the workflow:

```bash
kdeps run workflow.yaml
```

The example uses `llama3.2:1b`. Make sure the required model is available and configured for your installation.

### 4. Send a question

In another terminal, run:

```bash
curl -s -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"q": "Explain entropy in one sentence."}'
```

The API returns a JSON response containing the AI's answer.

The example token is for local testing only. Use a secure token and appropriate network protections for real deployments.

### 5. Test the workflow

Add `tests:` to `workflow.yaml` to check the response your workflow returns:

```yaml
tests:
  - name: chat
    request:
      method: POST
      path: /api/v1/chat
      body:
        q: "Say hi"
    assert:
      status: 200
      body:
        jsonPath:
          - path: $.data.answer
            exists: true
```

```bash
kdeps test .
```

`kdeps test` runs each case in-process against the workflow's own router - the same auth, routes, and resources as `kdeps run` - with no port bound.

More examples: [kdeps.com/examples](https://kdeps.com/examples/)

## How workflows work

A workflow is a collection of tasks called resources.

Each resource describes a job, such as:

- Asking an AI model a question
- Sending an HTTP request
- Running Python or shell commands
- Reading or writing data
- Querying a database
- Searching the web
- Calling another agent

Resources can depend on one another. Kdeps uses those dependencies to determine the execution order.

For example:

```text
Get information
      |
      v
Ask the AI to summarize it
      |
      v
Return the result
```

This makes the overall process easier to understand and test.

Run a workflow:

```bash
kdeps run workflow.yaml
```

You can also run a project directory or use development mode:

```bash
kdeps run ./my-agent/
kdeps run workflow.yaml --dev
```

## How agent mode works

Agent mode is an interactive AI assistant you can use from your terminal.

Give it a task, and the AI can decide which available tools to call and what to do next.

For example, a coding agent might:

1. Read a source file.
2. Find a possible problem.
3. Edit the file.
4. Run a test.
5. Explain the result.

The agent can use available workflows as tools, too.

Start the agent:

```bash
kdeps
```

Start a new session:

```bash
kdeps --new
```

Choose a model:

```bash
kdeps ./my-agent/ --model llama3.2
```

Agent mode is less predictable than a fixed workflow because the AI chooses its actions. Limit its permissions and review important changes. File tools may always use `~/.kdeps` and `/tmp`. Any other path outside the working directory asks first, and the filesystem root stays blocked.

In this repository, `/auto-improve` runs an instrumented session and reports product bugs the log shows, without editing code. On OpenRouter it jumps to another free model at random before each task and each evaluate turn, and a provider 429 during that session does the same. The command files one GitHub issue as each report arrives, skipping any open issue with the same title. A daily workflow runs one task per process, files the issue if the process dies first, and never opens a pull request. Free-model requests wait so they stay under the account per-minute cap. Outside this command, a provider 429 moves to the next free id and the turn continues. An account 429 waits out the minute and retries the same id.

## Connect agents with agencies

An agency brings several specialized agents together.

For example, a software-development agency might include:

- A planning agent
- A coding agent
- A testing agent
- A reviewing agent

Each agent has its own workflow and resources. Kdeps connects them so one agent can call another.

Run an agency:

```bash
kdeps run agency.yaml
```

See the [agency documentation](https://kdeps.com/agencies/) for the required configuration.

## Run your own AI model

Kdeps can use supported local models or connect to an inference server you manage.

For example, you can configure an OpenAI-compatible server in `~/.kdeps/config.yaml`:

```yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1
  models:
    - llama3.2
```

Replace the example address with the address of your own server.

You can also use the LLM server wizard:

```bash
kdeps llm wizard
```

A locally hosted model can help reduce dependence on external services and keep data within your infrastructure. Actual privacy and network behavior depend on your configuration and the services you enable.

## Build and deploy

Once your workflow is ready, you can package it for a supported environment.

Build a Docker image:

```bash
kdeps bundle build .
```

Export Kubernetes manifests:

```bash
kdeps export k8s
```

Export a bootable ISO:

```bash
kdeps export iso
```

Build a standalone binary:

```bash
kdeps bundle prepackage
```

Check the [deployment documentation](https://kdeps.com/deploy/) for platform requirements, model packaging, and configuration options.

## Desktop app

Kdeps also provides a desktop application for macOS, Linux, and Windows.

It includes a chat interface, model selection, conversation history, file attachments, settings, and tools for creating and running workflows and agencies.

The desktop application is standalone and does not require the Kdeps CLI to be installed.

Installation and downloads: [Desktop app documentation](https://kdeps.com/agent/desktop#install)

## More features

Kdeps also includes:

- **Reusable components and providers** — Add tools and AI providers through configuration and shared assets.
- **Your own tools and slash commands** — Drop a YAML file in `~/.kdeps/tools/` (an `http:`, `exec:`, `sql:` or `python:` tool) or `~/.kdeps/commands/`; kdeps serves it on the kdeps API with no Go code and no registry entry.
- **Knowledge vault** — Help agents find useful notes and project knowledge.
- **Search and retrieval** — Find relevant information in supported document collections.
- **Session history and instrumentation** — Inspect agent sessions, tool calls, errors, and workflow events.
- **Configuration management** — Manage model settings, connections, and other machine-specific options.
- **Registry** — Discover and share reusable agents and components.

See the [documentation](https://kdeps.com) for the full feature list and configuration details.

## Security

AI agents can execute tools and interact with data, so security matters.

- Keep API keys and passwords out of Git.
- Restrict access to files, commands, and network resources.
- Protect HTTP endpoints with authentication.
- Use secure settings when exposing services to a network.
- Review agent-generated changes before deploying them.
- Test workflows before using them with important data.

For production deployments, follow the [security](https://kdeps.com/reference/security) and [HTTPS](https://kdeps.com/deploy/tls-https) documentation.

## Documentation

- Source code: [github.com/kdeps/kdeps](https://github.com/kdeps/kdeps)
- Introduction: [kdeps.com/getting-started/introduction](https://kdeps.com/getting-started/introduction)
- Examples: [kdeps.com/examples](https://kdeps.com/examples/)
- Installation: [kdeps.com/start/installation](https://kdeps.com/start/installation)

## Contributing

Kdeps is open source. You can explore the code, report bugs, suggest improvements, and contribute changes through GitHub.

Please read the repository’s [contribution guidelines](CONTRIBUTING.md) before submitting a change.

## License

Kdeps uses the Apache License 2.0. See the repository’s [license file](LICENSE) for the full terms.

---

**In short:** Kdeps lets you describe an AI workflow or agent in YAML, connect it to the tools and models it needs, and package it for the environment where you want to run it.
