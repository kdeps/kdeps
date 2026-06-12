# Installation

Install the `kdeps` CLI to start building agents locally. Docker is optional -- only needed if you want to build container images for deployment.

## Installing KDeps CLI

### macOS (Homebrew)

```bash
brew install kdeps/tap/kdeps
```

### Linux, macOS, and Windows (curl)

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

### Windows (wget in WSL or Git Bash)

```bash
wget -qO- https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

> **Note for Windows Users**: For optimal functionality, run the installation command using either [Git Bash](https://git-scm.com/downloads/win) or [WSL](https://learn.microsoft.com/en-us/windows/wsl/install).

### From Source

**Option 1: Go Install (Recommended)**

```bash
go install github.com/kdeps/kdeps/v2@latest
```

**Option 2: Build Manually**

```bash
git clone https://github.com/kdeps/kdeps.git
cd kdeps
go build -o kdeps main.go
./kdeps --version
```

## Verify Installation

```bash
kdeps --version
```

You should see output like:
```
kdeps version 2.0.0
```

## Docker (Optional)

Docker is only needed if you want to build container images for deployment. For local development and testing, KDeps runs natively without Docker.

### Install Docker

- **macOS**: [Docker Desktop for Mac](https://docs.docker.com/desktop/install/mac-install/)
- **Windows**: [Docker Desktop for Windows](https://docs.docker.com/desktop/install/windows-install/)
- **Linux**: [Docker Engine](https://docs.docker.com/engine/install/)

### Verify Docker Installation

```bash
docker --version
```

## Local LLMs (no install needed)

For local LLM inference, KDeps uses [llamafile](https://github.com/Mozilla-Ocho/llamafile)
as the default backend (`file`): models are single self-contained binaries that
kdeps downloads to `~/.kdeps/models/` and serves locally - no server install,
no GPU, no API key. The default model alias `llama3.2:1b` resolves to Mozilla's
Llama 3.2 1B Instruct llamafile (~1.1 GB, downloaded on first run).

```bash
kdeps llamafile list      # see all known model aliases
kdeps llamafile update    # refresh the registry from HuggingFace
```

## Ollama (Optional)

To use [Ollama](https://ollama.ai/) instead of the default llamafile backend:

```bash
# macOS / Linux
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull llama3.2:1b
```

Then select it in `~/.kdeps/config.yaml`:

```yaml
llm:
  backend: ollama  # default is "file" (llamafile)
```

## Troubleshooting

### Permission Denied Error

If you encounter a `Permission Denied` error during installation:

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sudo sh
```

### Command Not Found

If `kdeps` is not found after installation, add `~/.local/bin` to your PATH:

```bash
# Add to ~/.bashrc or ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"
```

Then reload your shell:
```bash
source ~/.bashrc  # or source ~/.zshrc
```

### Docker Permission Issues (Linux)

If you get permission errors when running Docker commands:

```bash
sudo usermod -aG docker $USER
# Log out and back in for changes to take effect
```

## See Also

- [Quickstart Guide](/getting-started/quickstart) - Build your first AI agent
- [CLI Reference](/reference/cli/) - Complete command reference
- [Workflow Configuration](../configuration/workflow) - Learn about workflow settings
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples) - Browse example workflows
