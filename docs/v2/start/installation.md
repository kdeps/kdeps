# Installation

Install the `kdeps` CLI to start building agents locally. Docker is optional - only needed if you want to build container images for deployment.

Works in workflow and agent mode.

Already installed? [Quickstart](/workflow/quickstart) or [Run locally](/agent/quickstart).

## Installing the kdeps CLI

## macOS (Homebrew)

```bash
brew install kdeps/tap/kdeps
```

Desktop app (standalone, no CLI needed): `brew install --cask kdeps/tap/kdeps-desktop`. Linux and Windows (Scoop, packages, or manual) install steps are in [Desktop app](../agent/desktop#install).

## Windows (Scoop)

```powershell
scoop bucket add kdeps https://github.com/kdeps/scoop-bucket
scoop install kdeps           # CLI
scoop install kdeps-desktop   # desktop app, standalone
```

## Linux packages (deb, rpm, apk, Arch)

Each release attaches native CLI packages, so the package manager tracks the install. Pick the file for your distro from the [releases page](https://github.com/kdeps/kdeps/releases/latest):

```bash
sudo apt install ./kdeps_<version>_linux_amd64.deb            # Debian, Ubuntu
sudo dnf install ./kdeps_<version>_linux_amd64.rpm            # Fedora, RHEL
sudo pacman -U kdeps_<version>_linux_amd64.pkg.tar.zst        # Arch
sudo apk add --allow-untrusted ./kdeps_<version>_linux_amd64.apk   # Alpine
```

Use `linux_arm64` in the name on ARM machines. The desktop app has its own `.deb`, `.rpm` and Arch package that pull in the GTK and WebKit libraries for you. See [Desktop app](../agent/desktop#install).

## Linux, macOS, and Windows (curl)

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

## Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1 | iex
```

Installs `kdeps.exe` into `%USERPROFILE%\.local\bin` and adds it to your user `PATH`. To pin a version or choose a different directory:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1))) -Tag v3.0.0 -BinDir C:\tools\bin
```

## Windows (wget in WSL or Git Bash)

```bash
wget -qO- https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

> Note: [Git Bash](https://git-scm.com/downloads/win) or [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) also work if you prefer the shell-based installer above.

## From source

Recommended: `go install`

```bash
go install github.com/kdeps/kdeps/v3@latest
```

Or build manually:

```bash
git clone https://github.com/kdeps/kdeps.git
cd kdeps
make build
./kdeps --version
```

On Windows without `make` (or Git Bash/WSL), `build.ps1` and `build.bat` in the
repo root do the same build with version/commit info embedded, matching
`make build`:

```powershell
.\build.ps1
```
```cmd
build.bat
```

## Verify installation

```bash
kdeps --version
```

The output is similar to this:
```
kdeps version 2.21.0
```

## First run: pick a profile

The binary ships no prompts, schemas, templates or tool definitions. The first command downloads them from kdeps.io (about 1.5 MB for everything). It asks how much:

```text
What will you use kdeps for? (change it later with kdeps profile)
  1) minimum                kdeps run and validate: chat, httpClient, python, exec, apiResponse
  2) workflows  ~3.0k tok    minimum, kdeps new templates and the components (sql, ...)
  3) agent      ~11.1k tok   minimum and the agent loop: prompts, slash commands, themes, file/shell/web tools
  4) deploy     ~3.0k tok    workflows, and kdeps build/export: Docker and Kubernetes templates, LLM server recipes
  5) full       ~19.4k tok   every asset set
  6) custom                 tick the asset sets, or single items, yourself
  (~tok: what the profile adds to the LLM context - prompts and tool definitions)
Choice [5]:
```

If a later command needs more, kdeps installs only the parts that its loader needs. A small profile stays small and does not stop you. With `minimum`, the agent loop adds 23 core commands and these core tools:

- Files: `read_file`, `write_file`, `edit_file`, `list_files`, `tail_file`.
- Shell: `bash_exec` and its job tools.
- `search_local`, memory, and knowledge.

- Web and search tools: `kdeps install tools`, or one at a time (`kdeps install tools/web_search`).
- `kdeps install <set>` adds a set. `kdeps profile <name>` switches profile and prunes the rest.
- CI or Docker: no prompt. Set `KDEPS_PROFILE=agent` (or another profile), or leave it unset to install only what each command loads.
- A [shelf](/reference/glossary#shelf) files the same items by category: `kdeps list rag` shows one. The id stays `set/name`.

Details: [Updating assets](/agent/repl-updating#updating-assets).

## Docker (optional)

You need Docker only to build container images for deployment. For local development and tests, kdeps runs natively without Docker.

### Install Docker

- **macOS**: [Docker Desktop for Mac](https://docs.docker.com/desktop/install/mac-install/)
- **Windows**: [Docker Desktop for Windows](https://docs.docker.com/desktop/install/windows-install/)
- **Linux**: [Docker Engine](https://docs.docker.com/engine/install/)

### Verify Docker installation

```bash
docker --version
```

## Local LLMs (no install needed)

Default backend is llamafile: no server, no GPU, no API key. The alias
`llama3.2:1b` (~1.1 GB) downloads on first run. Full aliases, Ollama, and GGUF:
[Local models](/start/local-models).

```bash
kdeps llamafile list      # see all known model aliases
kdeps update models       # newest registry (harvested nightly from HuggingFace)
```

## First-run setup

The first time you run kdeps in a terminal with no `~/.kdeps/config.yaml`, it
asks if you want to set up language models. Press Enter to skip the setup. If
you select yes, kdeps opens the backend list and writes the configuration:

```text
  Set up language models in ~/.kdeps/config.yaml?
    [1] Yes
    [0] Not now

  Enter number [0]: 1

  How should kdeps run language models?
    [1] llamafile  local, self-contained, no server install (recommended)
    [2] gguf       local GGUF models via llama.cpp
    [3] cloud      OpenAI, Anthropic, DeepSeek, Groq, xAI, ... (needs an API key)
    [4] ollama     connect to an Ollama server
    [5] router     route across multiple models by strategy (advanced)
    [6] m365       Microsoft 365 Copilot (browser login, no API key)
    [0] Skip (configure later)
```

Each choice fills in the matching `llm:` fields:

- **llamafile / gguf** - sets `backend` and a default model.
- **cloud** - sets `backend`, prompts for the provider's API key, and stores it under `llm.<provider>_api_key`.
- **ollama** - sets `backend: ollama` and the host URL.
- **router** - collects the models to route across and a strategy (`fallback`, `round_robin`, `token_threshold`, `cost_optimized`), written as `models:` and `strategy:` under `llm:`.
- **m365** - sets `backend: m365`. It does not use an API key. It authenticates
  with a Microsoft 365 sign-in that the browser keeps in its cache. For CI and
  servers, it uses headless credentials. See
  [LLM Provider Reference - M365 Copilot](/llm-server/m365).

In non-interactive environments (CI, pipes) kdeps skips the wizard and writes a
fully commented template instead. To run the wizard again, remove
`~/.kdeps/config.yaml`, or edit it directly with `kdeps edit`.

## Ollama (optional)

To use [Ollama](https://ollama.com) instead of the default llamafile backend:

```bash
# macOS / Linux
curl -fsSL https://ollama.com/install.sh | sh
ollama pull llama3.2:1b
```

Then select it in `~/.kdeps/config.yaml`:

```yaml
llm:
  backend: ollama  # default is "file" (llamafile)
```

## Upgrading from v2

v3 groups the YAML keys and drops native resource kinds. A v2 file fails with an error naming each key's new place (`workflow.settings is now deploy, session and repl`), so `kdeps validate` lists every edit to make.

| v2 | v3 |
|---|---|
| `settings:` (`apiServer`, `webServer`, `input`, TLS) | `deploy:` |
| `settings.agentSettings` | `deploy.runtime` |
| `settings.session` / `settings.llm` | top-level `session:` / `repl:` |
| `tests:` | `evals:` (plus `judges`, `repeat`, `threshold`) |
| resource `requires`, `items`, `loop`, `onError` | under `flow:` |
| chat `model`, `prompt`, `tools`, `jsonResponse`, ... | grouped under `model`, `context`, `harness`, `tools`, `rag`, `guardrails` ([LLM resource](/workflow/resources/llm#key-groups)) |
| `exec:`, `python:`, `httpClient:`, `sql:`, `scraper:`, ... | `component: {name: exec, with: {...}}` (`http_request`, `web_scraper`, ...) |
| `botReply:` | the target resource's `apiResponse` |
| `go install github.com/kdeps/kdeps/v2@...` | `go install github.com/kdeps/kdeps/v3@...` |

`browser`, `email`, `telephony` and `codeIntelligence` resources are gone.

## Troubleshooting

### Permission denied error

If you encounter a `Permission Denied` error during installation:

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sudo sh
```

### Command not found

If `kdeps` is not found after installation, add `~/.local/bin` to your PATH:

```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"
```

Then reload your shell:
```bash
source ~/.bashrc  # or source ~/.zshrc
```

### Docker permission issues (Linux)

If you get permission errors when you run Docker commands:

```bash
sudo usermod -aG docker $USER
# Log out and back in for changes to take effect
```

## See also

- [Quickstart](/workflow/quickstart) - build your first workflow API
- [Run locally](/agent/quickstart) - agent REPL in 30 seconds
- [CLI reference](/reference/cli) - Complete command reference
- [Workflow configuration](/workflow/configuration) - Learn about workflow settings
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples) - Browse example workflows
