# Chapter 19: Standalone Binary

`kdeps bundle prepackage` embeds your workflow archive directly into the kdeps binary, producing a self-contained executable. Copy it to a machine and run it. No kdeps installation required, no Docker, no runtime dependencies beyond the binary itself.

This is the deployment target for edge devices, air-gapped servers, embedded systems, and any environment where you cannot run containers or install packages.

## Overview

```bash
# Bundle for all architectures
$ kdeps bundle prepackage myagent-1.0.0.kdeps

# Bundle for a single target
$ kdeps bundle prepackage myagent-1.0.0.kdeps --arch linux-amd64

# Write to a custom directory
$ kdeps bundle prepackage myagent-1.0.0.kdeps --output dist/

# Pin a specific kdeps runtime version
$ kdeps bundle prepackage myagent-1.0.0.kdeps --kdeps-version 2.0.1
```

## How It Works

The `.kdeps` archive is appended to the kdeps binary. A 24-byte magic trailer marks where the archive starts, so the binary can locate it at startup:

```
[kdeps runtime binary]          standard kdeps binary; identical behavior when run normally
[.kdeps archive data]           your workflow, resources, data
[24-byte trailer]               8-byte size field + 16-byte magic "KDEPS_PACK"
```

When you run the prepackaged binary, kdeps detects the embedded archive and runs it automatically — exactly as if you had run `kdeps run workflow.yaml`. No flags required. No separate workflow file needed. The binary is the entire deployment unit.

## Supported Architectures

```bash
$ kdeps bundle prepackage myagent-1.0.0.kdeps
```

By default, builds for all supported architectures:
- `linux-amd64` (x86_64 Linux)
- `linux-arm64` (ARM64 Linux — Raspberry Pi 4, AWS Graviton, Apple M1 via Rosetta)
- `darwin-amd64` (Intel Mac)
- `darwin-arm64` (Apple Silicon Mac)
- `windows-amd64` (Windows x86_64)

Single architecture:

```bash
$ kdeps bundle prepackage myagent-1.0.0.kdeps --arch linux-arm64
```

Output files are named `myagent-1.0.0-linux-amd64`, `myagent-1.0.0-linux-arm64`, etc. (`.exe` for Windows).

## Typical Workflow

```bash
# 1. Create the workflow
$ kdeps new my-agent

# 2. Build and test locally
$ export KDEPS_API_AUTH_TOKEN=dev-token
$ kdeps run workflow.yaml
$ curl -X POST http://localhost:16395/api/v1/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -d '{"q": "test"}'

# 3. Package the workflow
$ kdeps bundle package workflow.yaml
# → my-agent-1.0.0.kdeps

# 4. Create self-contained executables
$ kdeps bundle prepackage my-agent-1.0.0.kdeps --output dist/
# → dist/my-agent-1.0.0-linux-amd64
# → dist/my-agent-1.0.0-linux-arm64
# → dist/my-agent-1.0.0-darwin-arm64
# → ...

# 5. Deploy to target machine
$ scp dist/my-agent-1.0.0-linux-amd64 user@server:/usr/local/bin/my-agent
$ ssh user@server chmod +x /usr/local/bin/my-agent

# 6. Run on target machine (no kdeps install needed)
$ ssh user@server my-agent
```

## Running the Prepackaged Binary

```bash
# The binary starts the workflow server automatically
$ ./my-agent-1.0.0-linux-amd64
kdeps: embedded workflow detected: my-agent v1.0.0
kdeps: starting server on 127.0.0.1:16395
kdeps: ready

# Override the port at runtime
$ ./my-agent-1.0.0-linux-amd64 --port 8080

# Override the bind address
$ ./my-agent-1.0.0-linux-amd64 --host 0.0.0.0

# Run in dev mode (hot reload from embedded workflow)
$ ./my-agent-1.0.0-linux-amd64 --dev
```

Environment variables work exactly as in the packaged workflow — pass them in the shell:

```bash
$ DATABASE_URL="postgresql://..." \
  KDEPS_API_AUTH_TOKEN="secret" \
  KDEPS_MANAGEMENT_TOKEN="mgmt-secret" \
  ./my-agent-1.0.0-linux-amd64
```

When `apiServer` is configured, `KDEPS_API_AUTH_TOKEN` is required — the process exits on startup without it.

## Edge Device Deployment

For ARM-based edge devices (Raspberry Pi, Jetson Nano, industrial PLCs):

```bash
# Build for ARM64
$ kdeps bundle prepackage myagent-1.0.0.kdeps --arch linux-arm64

# Copy to device
$ scp myagent-1.0.0-linux-arm64 pi@raspberrypi:/home/pi/my-agent

# SSH to device and run
$ ssh pi@raspberrypi
$ chmod +x /home/pi/my-agent
$ /home/pi/my-agent
```

The binary includes the complete kdeps runtime. It does not need Python installed (if you use `python:` resources, those dependencies are bundled), it never needs an LLM server (the default file backend downloads and self-serves llamafiles), and it does not need any system packages beyond a standard Linux userland.

For workflows that use `python:` resources, kdeps bundles a minimal Python runtime in the prepackage. For workflows that only use `httpClient:`, `exec:`, `sql:`, and `chat:` with remote backends, the binary is maximally lean.

## Embedding the Model: --include-models

By default the binary downloads its llamafile on first run. For a truly
self-contained artifact, embed the model too:

```bash
$ kdeps bundle prepackage myagent-1.0.0.kdeps --include-models
```

Every literal `chat.model` in the package is resolved through the llamafile
registry and embedded into the executable. At run time the embedded models
become the llamafile cache, so aliases like `llama3.2:1b` resolve with zero
network. One file carries the runtime, the workflow, and the model — expect
roughly +1.1 GB per model. Agencies work the same way:

```bash
$ kdeps bundle prepackage my-agency-1.0.0.kagency --include-models
```

## Air-Gapped Environments

For networks with no internet access, build with the model embedded on a
connected machine, then transfer one file:

```bash
# Connected machine
$ kdeps bundle prepackage myagent-1.0.0.kdeps --arch linux-amd64 --include-models

# Transfer and run - no network needed, ever
$ scp myagent-1.0.0-linux-amd64 airgapped:/opt/my-agent
$ ssh airgapped /opt/my-agent
```

A single prepackaged binary with embedded models is a complete AI appliance
with zero external dependencies. (If you prefer Ollama, pre-pull the model on
a connected machine, transfer the model data, and point the config at the
local Ollama instance instead.)

## Systemd Service

For running the agent as a system service on Linux:

```ini
# /etc/systemd/system/my-agent.service
[Unit]
Description=My AI Agent
After=network.target

[Service]
Type=simple
User=myagent
WorkingDirectory=/opt/my-agent
ExecStart=/opt/my-agent/my-agent-1.0.0-linux-amd64
Restart=always
RestartSec=5
Environment="DATABASE_URL=postgresql://user:pass@localhost:5432/mydb"
Environment="KDEPS_API_AUTH_TOKEN=secret"
Environment="KDEPS_MANAGEMENT_TOKEN=mgmt-secret"
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

```bash
$ sudo systemctl enable my-agent
$ sudo systemctl start my-agent
$ sudo systemctl status my-agent
```

The agent runs as a daemon, restarts automatically on failure, and logs to the system journal.

## Pinning the Runtime Version

The prepackaged binary includes the kdeps runtime at whatever version you ran `prepackage` with. To pin to a specific version:

```bash
$ kdeps bundle prepackage myagent-1.0.0.kdeps --kdeps-version 2.0.1
```

This downloads and embeds kdeps runtime version 2.0.1 specifically, regardless of what version you have installed. This ensures the binary behaves identically whether built today or in six months.

## Comparing Deployment Targets

| Target | Command | Best for |
|---|---|---|
| Docker | `kdeps bundle build` | Container orchestration, CI/CD pipelines |
| Kubernetes | `kdeps export k8s` | Cloud infrastructure, autoscaling |
| Binary | `kdeps bundle prepackage` | Edge devices, air-gapped systems, simple VMs |
| ISO | `kdeps bundle iso` | Bootable appliances, dedicated hardware |

The same `.kdeps` archive can produce all of these targets. Choose based on your deployment environment, not the workflow content.

X> ## Exercise
X>
X> Produce a self-contained binary from the chatbot workflow and run it on a machine without kdeps installed.
X>
X> 1. Package the workflow: `kdeps bundle package workflow.yaml`
X> 2. Prepackage for your current platform: `kdeps bundle prepackage myagent-1.0.0.kdeps --arch linux-amd64` (adjust `--arch` to match your machine).
X> 3. Locate the output binary in `dist/`. Run `file dist/myagent-linux-amd64` and confirm it is a statically linked ELF executable (or Mach-O on macOS).
X> 4. Copy the binary to a temp directory that has no kdeps installation on PATH. Run it directly:
X>    ```bash
X>    ./myagent-linux-amd64
X>    ```
X>    Confirm the HTTP server starts and responds to curl.
X> 5. Run `kdeps bundle prepackage myagent-1.0.0.kdeps` without `--arch` to produce binaries for all platforms. List the `dist/` directory and verify binaries for at least three target architectures were produced.
X>
X> Compare the binary size with and without Python dependencies in `agentSettings.pythonPackages`.
X>
X> **Stretch goal:** Embed the binary in a systemd unit file so the agent starts automatically on Linux boot. Write the `.service` file and verify `systemctl status` shows the agent as active.

## LLM server appliances vs prepackage

Prepackage embeds a **workflow agent** into a kdeps binary. A **LLM server appliance** (`kdeps llm`) is the opposite shape: only the inference server, no agent binary. If you need a shared model farm for many clients, use Chapter 26 rather than embedding models into every prepackage.

```bash
$ kdeps llm build --engine ollama --model llama3.2 --tag myorg/llm:1
$ kdeps llm client-config --url http://llm-host:8000/v1
```
