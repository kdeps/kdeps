# Install

Latest release: **[v2.1.11](https://github.com/kdeps/kdeps/releases/tag/v2.1.11)**.

## macOS (Homebrew)

```bash
brew install kdeps/tap/kdeps
```

## macOS / Linux / Windows (curl)

```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

On Windows use [Git Bash](https://git-scm.com/downloads/win) or [WSL](https://learn.microsoft.com/en-us/windows/wsl/install).

## Go

```bash
go install github.com/kdeps/kdeps/v2@latest
```

## Verify

```bash
kdeps --version
# kdeps version 2.1.11 …
```

## First run — coding agent

```bash
kdeps
```

Optional project tools:

```bash
kdeps .
# or
kdeps ./my-agent/
```

First launch may ask for a model backend. **llamafile** needs no API key; files go under `~/.kdeps/models/`.

Cloud keys when you want them:

```bash
export ANTHROPIC_API_KEY=...
export OPENAI_API_KEY=...
export DEEPSEEK_API_KEY=...
```

Or `llm:` in `~/.kdeps/config.yaml`.

## Workflow later

```bash
kdeps new my-agent
export KDEPS_API_AUTH_TOKEN=dev-token
kdeps run workflow.yaml --dev
```

## Docker

Only if you build images (`kdeps bundle build`). Install [Docker](https://docs.docker.com/get-docker/), then `docker --version`.

Next: [Quickstart](/quickstart) · [Coding agent](/agent).
