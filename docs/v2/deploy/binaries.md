# Prepackage - standalone executables

`kdeps bundle prepackage` embeds a `.kdeps` workflow archive directly into the kdeps binary, producing a single self-contained executable per architecture. Copy it to the target machine and run it - no kdeps installation required.

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

## How it works

kdeps adds the archive to the end of the kdeps binary. A 24-byte magic trailer marks the start of the archive, so the binary can find it at startup. When you run the prepackaged binary, it finds the embedded `.kdeps` archive and runs it automatically, exactly like `kdeps run`. The archive also has the kdeps assets of the packaging machine (harness, tool definitions, model registries, ...) under `.kdeps-assets/`. As a result, the binary runs offline with these assets.

```d2
direction: down

A: "kdeps runtime (ELF)\nidentical to a normal kdeps binary"
B: ".kdeps archive data\nyour workflow, resources, data\n+ .kdeps-assets/ (kdeps assets)"
C: "24-byte trailer\n8-byte size field + 16-byte magic 'KDEPS_PACK'"

A -> B -> C
```

## Typical workflow

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

## Supported targets

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
| `--arch` | *(all)* | Limit to one target: `linux-amd64`, `darwin-arm64`, ... |
| `--kdeps-version` | *(running version)* | Specific kdeps release to download as base binary |

## Cross-architecture builds

For the host architecture, `prepackage` reuses the running kdeps binary as
the base.  For all other architectures, it downloads the corresponding release
binary from [GitHub Releases](https://github.com/kdeps/kdeps/releases).

This means cross-arch builds require:
- A published kdeps release (you cannot download dev builds)
- Internet access during the `prepackage` step

```bash
# Works immediately for the host arch - no download required
kdeps bundle prepackage myagent.kdeps --arch linux-amd64  # on a Linux/amd64 host

# Requires download from GitHub Releases
kdeps bundle prepackage myagent.kdeps --arch linux-arm64
```

::: tip Dev builds
With a development build (the version ends in `-dev`), kdeps makes only the
binary for the host architecture. To make cross-arch binaries from a dev build,
give `--kdeps-version <release>`.
:::

## Re-prepackaging

`prepackage` is idempotent. If the input binary already has an embedded
`.kdeps` archive, kdeps removes the old archive before it attaches the new one.
As a result, you can safely prepackage a binary again with an updated workflow:

```bash
# First release
kdeps bundle prepackage myagent-1.0.0.kdeps --output dist/

# Updated workflow - re-prepackage the same output binary
kdeps bundle prepackage myagent-1.1.0.kdeps --output dist/
```

## Using with CI/CD

A typical GitHub Actions workflow:

```yaml
# .github/workflows/deploy.yml
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

## Running a prepackaged binary

A prepackaged binary behaves exactly like `kdeps run myagent.kdeps` - it starts an API server, processes requests, and handles all workflow steps.

```bash
# Linux/macOS
chmod +x my-agent-1.0.0-linux-amd64
./my-agent-1.0.0-linux-amd64

# Windows
my-agent-1.0.0-windows-amd64.exe
```

The binary listens on `portNum` under `settings:` `apiServer:` in `workflow.yaml` (default 16395). Workflow routes require `KDEPS_API_AUTH_TOKEN` (or `api_auth_token` in `~/.kdeps/config.yaml`):

```bash
export KDEPS_API_AUTH_TOKEN=dev-token
./my-agent-1.0.0-linux-amd64

curl -X POST http://localhost:16395/api/chat \
  -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Hello"}'
```

## See also

- [Package command](/workflow/quickstart) - Creating `.kdeps` archives
- [Docker deployment](docker) - Containerized deployment
