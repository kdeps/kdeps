# Installation

Install the `kdeps` CLI to start building agents locally. Docker is optional - only needed if you want to build container images for deployment.

*Applies to both workflow mode and agent mode.*

Already installed? [Run locally](/getting-started/local-agent) or [Quickstart](/getting-started/quickstart).

## Installing the kdeps CLI

### macOS (Homebrew)

```bash
brew install kdeps/tap/kdeps
```

### Linux, macOS, and Windows (curl)

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1 | iex
```

Installs `kdeps.exe` into `%USERPROFILE%\.local\bin` and adds it to your user `PATH`. To pin a version or choose a different directory:

```powershell
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/kdeps/kdeps/main/install.ps1))) -Tag v2.21.0 -BinDir C:\tools\bin
```

### Windows (wget in WSL or Git Bash)

```bash
wget -qO- https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

> Note: [Git Bash](https://git-scm.com/downloads/win) or [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) also work if you prefer the shell-based installer above.

### From source

Recommended: `go install`

```bash
go install github.com/kdeps/kdeps/v2@latest
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

You should see output like:
```
kdeps version 2.21.0
```

## Docker (optional)

Docker is only needed if you want to build container images for deployment. For local development and testing, kdeps runs natively without Docker.

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
[Local models](/getting-started/local-models).

```bash
kdeps llamafile list      # see all known model aliases
kdeps llamafile update    # refresh the registry from HuggingFace
```

## First-run setup

The first time you run kdeps in a terminal with no `~/.kdeps/config.yaml`, an
interactive wizard asks how you want to run language models and writes the
config for you:

```text
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
- **router** - collects the models to route across and a strategy (`fallback`, `round_robin`, `token_threshold`, `cost_optimized`), written as `llm.models` + `llm.strategy`.
- **m365** - sets `backend: m365`. No API key: authenticates via a browser-cached
  Microsoft 365 sign-in (or headless credentials for CI/servers). See
  [LLM Provider Reference - M365 Copilot](/reference/llm-providers-m365).

In non-interactive environments (CI, pipes) kdeps skips the wizard and writes a
fully commented template instead. Re-run the wizard any time by removing
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

If you get permission errors when running Docker commands:

```bash
sudo usermod -aG docker $USER
# Log out and back in for changes to take effect
```

## See also

- [Run locally](/getting-started/local-agent) - agent REPL in 30 seconds
- [Quickstart](/getting-started/quickstart) - build your first workflow API
- [CLI reference](/reference/cli/) - Complete command reference
- [Workflow configuration](../configuration/workflow) - Learn about workflow settings
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples) - Browse example workflows
