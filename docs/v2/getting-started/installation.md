# Installation

KDeps requires minimal dependencies to get started. The only requirements are:

- **KDeps CLI** - The command-line interface
- **Docker** (optional) - Only needed for building container images

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

## Ollama (Optional)

For local LLM inference, KDeps uses [Ollama](https://ollama.ai/) as the default backend. If you want to use local LLMs:

### Install Ollama

```bash
# macOS / Linux
curl -fsSL https://ollama.ai/install.sh | sh

# Or download from https://ollama.ai/download
```

### Pull a Model

```bash
ollama pull llama3.2:1b
```

> **Note**: KDeps can automatically download models when you run a workflow, so manual model pulling is optional.

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

## Next Steps

- [Quickstart Guide](quickstart) - Build your first AI agent
- [CLI Reference](cli-reference) - Complete command reference
- [Workflow Configuration](../configuration/workflow) - Learn about workflow settings
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples) - Browse example workflows
