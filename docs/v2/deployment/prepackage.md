# Prepackage — Standalone Executables

The `prepackage` command bundles a `.kdeps` workflow archive together with the
entire kdeps runtime into a single, self-contained executable binary per
architecture.  No separate kdeps installation is required to run a prepackaged
binary — just copy it to the target machine and execute it.

## Overview

```bash
# Bundle for all architectures
kdeps bundle prepackage myagent-1.0.0.kdeps

# Bundle for a single target
kdeps bundle prepackage myagent-1.0.0.kdeps --arch linux-amd64

# Write to a custom directory
kdeps bundle prepackage myagent-1.0.0.kdeps --output dist/

# Pin a specific kdeps runtime version
kdeps bundle prepackage myagent-1.0.0.kdeps --kdeps-version 2.0.1
```

## How It Works

1. The `.kdeps` archive is appended to the kdeps binary.
2. A small 24-byte magic trailer is written after the archive so the binary can
   detect its own embedded payload at startup.
3. When a prepackaged binary is executed, it automatically detects the embedded
   `.kdeps` archive and runs `kdeps run` on it — no flags required.

```
┌──────────────────────┐
│  kdeps runtime (ELF) │  ← identical to a normal kdeps binary
├──────────────────────┤
│  .kdeps archive data │  ← your workflow, resources, data
├──────────────────────┤
│  [8-byte size field] │  ← uint64 big-endian: size of archive
│  [16-byte magic]     │  ← "KDEPS_PACK\0\0\0\0\0\0"
└──────────────────────┘
```

## Workflow

The typical workflow is:

```bash
# 1. Create your agent
kdeps new my-agent

# 2. Develop and test locally
kdeps run my-agent/

# 3. Package the workflow
kdeps bundle package my-agent/ --output dist/

# 4. Prepackage as standalone binaries
kdeps bundle prepackage dist/my-agent-1.0.0.kdeps --output dist/

# 5. Distribute and run
./dist/my-agent-1.0.0-linux-amd64
```

## Supported Targets

| Target          | Output filename                    |
|-----------------|------------------------------------|
| `linux-amd64`   | `<name>-<version>-linux-amd64`     |
| `linux-arm64`   | `<name>-<version>-linux-arm64`     |
| `darwin-amd64`  | `<name>-<version>-darwin-amd64`    |
| `darwin-arm64`  | `<name>-<version>-darwin-arm64`    |
| `windows-amd64` | `<name>-<version>-windows-amd64.exe` |

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--output`, `-o` | `.` | Directory where binaries are written |
| `--arch` | *(all)* | Limit to one target: `linux-amd64`, `darwin-arm64`, … |
| `--kdeps-version` | *(running version)* | Specific kdeps release to download as base binary |

## Cross-Architecture Builds

For the **host architecture**, `prepackage` reuses the running kdeps binary as
the base.  For all other architectures, it downloads the corresponding release
binary from [GitHub Releases](https://github.com/kdeps/kdeps/releases).

This means cross-arch builds require:
- A published kdeps release (dev builds cannot be downloaded)
- Internet access during the `prepackage` step

```bash
# Works immediately for the host arch — no download required
kdeps bundle prepackage myagent.kdeps --arch linux-amd64  # on a Linux/amd64 host

# Requires download from GitHub Releases
kdeps bundle prepackage myagent.kdeps --arch linux-arm64
```

::: tip Dev builds
When running a development build (version ends in `-dev`), only the host
architecture binary is produced.  Specify `--kdeps-version <release>` to
produce cross-arch binaries even from a dev build.
:::

## Re-Prepackaging

`prepackage` is idempotent.  If the input binary already contains an embedded
`.kdeps` archive, the old archive is stripped before the new one is attached.
This means you can safely re-prepackage a binary with an updated workflow:

```bash
# First release
kdeps bundle prepackage myagent-1.0.0.kdeps --output dist/

# Updated workflow — re-prepackage the same output binary
kdeps bundle prepackage myagent-1.1.0.kdeps --output dist/
```

## Using with CI/CD

A typical GitHub Actions workflow:

```yaml
- name: Package workflow
  run: kdeps bundle package my-agent/ --output dist/

- name: Prepackage for all architectures
  run: |
    VERSION=$(cat my-agent/workflow.yaml | grep 'version:' | head -1 | awk '{print $2}' | tr -d '"')
    kdeps bundle prepackage dist/my-agent-${VERSION}.kdeps \
      --output dist/ \
      --kdeps-version ${{ env.KDEPS_VERSION }}

- name: Upload artifacts
  uses: actions/upload-artifact@v4
  with:
    name: my-agent-binaries
    path: dist/my-agent-*
```

## Running a Prepackaged Binary

A prepackaged binary behaves exactly like `kdeps run myagent.kdeps` — it
starts an API server, processes requests, and handles all workflow steps.

```bash
# Linux/macOS
chmod +x my-agent-1.0.0-linux-amd64
./my-agent-1.0.0-linux-amd64

# Windows
my-agent-1.0.0-windows-amd64.exe
```

The workflow's configured port (`settings.apiServer.portNum`, default 16395) is
automatically used:

```bash
curl http://localhost:16395/api/chat -d '{"message":"Hello"}'
```

## Next Steps

- [Package Command](../getting-started/quickstart) — Creating `.kdeps` archives
- [Docker Deployment](docker) — Containerized deployment
