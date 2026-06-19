# Agent Loop Mode

Agent loop mode starts an interactive LLM REPL where whole workflows and components are registered as callable tools. The LLM decides which tool to invoke based on the user's prompt. Workflow tools run the full pipeline atomically so all `requires:` dependencies resolve correctly.

Running `kdeps` with no arguments starts a model-only REPL with no workflow tools. Pass a path to load workflows/agencies as tools.

## Starting the agent loop

```bash
kdeps                              # model-only REPL (no tools)
kdeps ./my-agent/                  # one workflow = one tool
kdeps ./agents/                    # folder = every workflow inside becomes a tool
kdeps ./my-agent/ --model llama3.2 --system "You are a DevOps assistant."
kdeps --skill ~/.kdeps/skills/     # load skill files
kdeps --resume <session-id>        # continue a saved session
```

## REPL slash commands

Inside the REPL, type `/help` for the full list:

| Command | Description |
|---------|-------------|
| `/help` | Show available commands |
| `/clear` | Summarize and clear the current conversation |
| `/model [name]` | Show or switch LLM model mid-session |
| `/models` | List all available models with provider status |
| `/skills` | List loaded skills |
| `/prompts` | List loaded prompt templates |
| `/<skill-name> [prompt]` | Invoke a skill or prompt template directly |
| `/compact` | Summarize history to free context |
| `/history` | Show conversation history |
| `/thinking [off\|low\|medium\|high\|auto]` | Enable extended reasoning (Claude) |
| `/session list\|save\|load\|delete\|checkpoint\|goto` | Manage saved sessions |
| `/copy` | Copy last assistant response to clipboard |
| `/reload` | Reload skills and prompt templates from disk |
| `/settings` | Open the tool/skill selector |
| `/exit` | Exit the REPL |
| `! <cmd>` | Run a shell command; result is added to LLM context |
| `!! <cmd>` | Run a shell command without adding it to LLM context |

## Multimodal input

Attach images and other binary files to your prompt using `@`:

```bash
# Attach a local image
describe @photo.png what is in this image?

# Attach multiple images
compare @before.jpg @after.jpg what changed?

# Attach a remote image URL
analyze @https://example.com/chart.png what trend does this show?

# Embed a text file inline (text files expand inline, not as attachments)
review @notes.txt and summarize the key points
```

- Image/binary refs (`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, `.tiff`, `.pdf`, `.mp3`, `.mp4`, `.wav`) are sent as multimodal content to the LLM
- Text file refs are expanded inline in the prompt
- Unresolvable refs (file not found, access denied) are left unchanged in the text

## Skills

Skills are markdown files with optional YAML frontmatter that teach the agent how to behave in specific contexts. Place them in `~/.kdeps/skills/` or pass `--skill <path>` at startup.

```markdown
---
name: code-review
description: Guidelines for reviewing Go code
---

Always check for error handling. Prefer early returns over nested conditions.
```

Skills are discovered from:
- `~/.kdeps/skills/` (global)
- `./.kdeps/skills/` (project-local)
- Paths passed with `--skill` (explicit, repeatable)

Invoke a skill from the REPL with `/<skill-name>` or `/<skill-name> extra context here`.

## Prompt templates

Prompt templates are reusable named prompts loaded from `.md` files. They work exactly like skills: invoke them by name from the REPL.

```markdown
---
name: review-pr
description: Review a GitHub pull request
argument-hint: <PR number or URL>
---

Review the pull request at $1. Check for: correctness, test coverage, and breaking changes.
```

Place templates in `~/.kdeps/prompts/` or `./.kdeps/prompts/`. Templates use the same placeholder syntax as skills: `$1`, `$2`, `$@`, `${1:-default}`.

```bash
/review-pr 1234
/summarize this document for a technical audience
```

## Instructions

The agent automatically discovers instruction files by walking up the directory tree from CWD:

- `CLAUDE.md`, `CLAUDE.local.md` at any ancestor directory
- `.kdeps/CLAUDE.md`, `.kdeps/instructions.md` at any ancestor directory

Duplicate content (by hash) is deduplicated. Total injected context is capped at ~12 KB. Instructions are injected into the system prompt at startup.

## Session persistence

Every conversation is saved as a JSONL file under `~/.kdeps/sessions/`. To resume a previous session:

```bash
kdeps --resume <session-id>
```

Session IDs are shown at the start of each run.

## Single workflow vs folder

```bash
kdeps ./my-agent/     # One workflow = one tool (named after metadata.name)
kdeps ./agents/       # Folder = every workflow and agency inside becomes a separate tool
```

When you point to a folder, kdeps discovers every workflow and agency file inside it (recursively). Each becomes a separate tool. The tool name is `metadata.name` from the workflow's manifest -- not the filename.

## Concrete example

Given this workflow:

```yaml
# my-agent/workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: my-agent          # this becomes the tool name the LLM sees
  version: "1.0.0"
  description: "Answers questions about our product"
  targetActionId: response

settings:
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/chat
        methods: [POST]
```

Running:

```bash
kdeps ./my-agent/
```

The LLM receives one tool named `my-agent`. When it calls that tool, kdeps runs the full workflow DAG -- every resource in dependency order -- and returns `apiResponse.response` to the LLM.

## How it works

```d2
direction: down

A: user prompt {shape: oval}
B: "LLM receives prompt\ntool registry: one tool per workflow, one per agency, one per component"
C: tool type? {shape: diamond}
D: "kdeps runs full workflow pipeline\nall requires: deps resolve in order"
E: "kdeps runs agency entry-point pipeline\ninternal agents resolve via agent: resource type"
H: "kdeps runs component in isolation\ninputs map to component interface fields"
F: more tools needed? {shape: diamond}
G: final answer {shape: oval}

A -> B
B -> C: LLM picks a tool
C -> D: workflow
C -> E: agency
C -> H: component
D -> F: apiResponse returned to LLM
E -> F: result returned to LLM
H -> F: result returned to LLM
F -> C: yes
F -> G: no
```

## Tool registration

| Target | Tools registered |
|--------|-----------------|
| No path (model-only) | None -- pure LLM conversation |
| Single workflow file/dir | One tool (`metadata.name`) + one tool per component |
| Single agency file | One tool (`agency metadata.name`) |
| Folder | One tool per workflow/agency found recursively + component tools |

## Command

```bash
kdeps [path] [flags]
```

`[path]` is optional. When provided it must be a workflow/agency file or directory. The tool name comes from `metadata.name` -- not the filename.

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `KDEPS_AGENT_MODEL` or `llama3.2` | LLM model name |
| `--backend` | `KDEPS_AGENT_BACKEND` or `file` | LLM backend (`file`, `gguf`, `ollama`, `openai`, ...) |
| `--base-url` | `KDEPS_AGENT_BASE_URL` | LLM API base URL |
| `--system` | (none) | System prompt injected at conversation start |
| `--skill` | (none) | Path to a skill file or directory (repeatable) |
| `--prompt` | (none) | Path to a prompt templates directory (repeatable) |
| `--resume` | (none) | Session ID to resume a previous conversation |
| `--debug` | false | Enable debug logging |

### Environment variables

```bash
KDEPS_AGENT_MODEL=llama3.2
KDEPS_AGENT_BACKEND=file              # default: local llamafile
# KDEPS_AGENT_BACKEND=gguf           # llama.cpp via llama-server
# KDEPS_AGENT_BACKEND=ollama         # requires ollama server
# KDEPS_AGENT_BASE_URL=http://localhost:11434
```

## Examples

```bash
# Pure LLM REPL, no workflows
kdeps

# Single workflow -- one tool
kdeps ./my-agent/

# All workflows in a folder
kdeps ./agents/

# Specify model and system prompt
kdeps ./agents/ --model mistral --system "You are a data analyst."

# GGUF backend with local model file
kdeps --backend gguf --model qwen3.5-4b

# OpenAI backend
KDEPS_AGENT_BACKEND=openai kdeps ./agents/ --model gpt-4o

# Load a skill directory
kdeps --skill ~/.kdeps/skills/

# Resume a previous session
kdeps --resume abc123def456
```

## Differences from workflow mode

| | Workflow mode (`kdeps run`) | Agent loop mode (`kdeps [path]`) |
|--|-----------------------------|---------------------------------|
| Execution | DAG, deterministic | LLM loop, tool-driven |
| Entry point | `metadata.targetActionId` | User prompt |
| Unit of work | Individual resources | Whole workflows |
| Tools exposed | N/A | One per workflow + one per component |
| Input | Single workflow path | Optional file or folder |
| Session memory | None | Multi-turn, persistent JSONL |

## See Also

- [Workflow Mode](workflow-mode) - Deterministic DAG pipelines
- [LLM Provider Reference](/reference/llm-providers) - Backend config and model names
- [Agencies](/concepts/agency) - Multi-agent orchestration
