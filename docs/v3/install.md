# Install

Get the CLI, then run `kdeps`.

## macOS (Homebrew)

```bash
brew install kdeps/tap/kdeps
```

## macOS / Linux / Windows (curl)

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

On Windows, use [Git Bash](https://git-scm.com/downloads/win) or [WSL](https://learn.microsoft.com/en-us/windows/wsl/install).

## Go

```bash
go install github.com/kdeps/kdeps/v2@latest
```

## Verify

```bash
kdeps --version
```

## First run

```bash
kdeps
```

On a fresh machine, kdeps may ask which language model to use. Pick **llamafile** for a local model with no server install. The default model downloads into `~/.kdeps/models/` on first use.

Cloud keys (when you want them):

```bash
export ANTHROPIC_API_KEY=...
export OPENAI_API_KEY=...
export DEEPSEEK_API_KEY=...
```

Or set `llm:` in `~/.kdeps/config.yaml`.

## Docker

Only needed if you will build container images. Install [Docker Desktop](https://docs.docker.com/get-docker/) or Docker Engine, then:

```bash
docker --version
```

Next: [Quickstart](/quickstart).
