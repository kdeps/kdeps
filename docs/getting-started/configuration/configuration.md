---
outline: deep
---

# Kdeps Configuration

Before running Kdeps, it requires a configuration that determines how it will operate on your system.

Initially, when you execute the `kdeps` command for the first time, it automatically creates a configuration file at `~/.kdeps.pkl`.

This file contains the following default settings:

```apl
amends "package://schema.kdeps.com/core@0.1.30#/Kdeps.pkl"

RunMode = "docker"
dockerGPU = "cpu"
kdepsDir = ".kdeps"
kdepsPath = "user"
```

These settings define key operational parameters for Kdeps, such as the runtime mode, GPU configuration, and directory paths.

## RunMode

The mode of execution for Kdeps, defaulting to `docker`.

> **Note:**
> At the moment, Kdeps only supports `docker` run mode. Future versions allows `local` for running Kdeps locally (for dedicated
> bare-metal AI agent systems) or in the `cloud`.

## DockerGPU

Specifies the type of GPU available for the Docker image. Supported values include `nvidia`, `amd`, or `cpu`. The default is set to `cpu`.

> **Note:**
> The Docker image will use the specified GPU type, so it's important to set this correctly if you're building an image for a specific GPU.

## KdepsDir

The directory where Kdeps files are stored defaults to `.kdeps`. This folder contains subdirectories such as
`.kdeps/agents` and `.kdeps/cache`. The parent directory is determined by the `kdepsPath` configuration.

## KdepsPath

The path where Kdeps configurations are stored defaulting to `user`, and it supports three options: `user`, `project`, and `xdg`.

- `user` refers to the `$HOME/.kdeps` directory.
- `project` refers to the current working directory of the project, e.g., `$HOME/Projects/aiagentx/.kdeps`.
- `xdg` refers to the XDG directory path, e.g., `$XDGPATH/.kdeps`.

## TIMEOUT (environment variable)

If you add `TIMEOUT=<seconds>` to your local `.env` file, Kdeps will use that value as the global default timeout for exec, HTTP-client, chat or Python steps **and will override any `timeoutDuration` already set in the PKL**.

* `TIMEOUT=<n>` (n > 0) → wait up to *n* seconds.
* `TIMEOUT=0` → **unlimited** (no timeout at all).
* Absent → falls back to 60 s.

Example:

```bash
# .env or shell
TIMEOUT=120  # Kdeps will wait up to 120 s by default
```

Handy for slow machines or high-latency networks.
