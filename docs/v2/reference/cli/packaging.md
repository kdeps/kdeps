# Packaging Commands

Package workflows for distribution and generate deployment artifacts.

## `kdeps bundle package`

Package workflow or component into an archive for distribution.

```bash
kdeps bundle package [directory] [flags]
```

**Output by detected manifest:**

| Detected file | Output format | Extension |
|---|---|---|
| `workflow.yaml` | Workflow package | `.kdeps` |
| `agency.yaml` | Agency package | `.kagency` |
| `component.yaml` | Component package | `.komponent` |

**Flags:**

| Flag | Description | Default |
|---|---|---|
| `--output, -o` | Output directory | `.` (current) |
| `--name` | Package name | From metadata (name-version) |

**What it packages:**
- Main manifest (`workflow.yaml`, `agency.yaml`, or `component.yaml`)
- All resource files (`resources/`)
- Python requirements (`requirements.txt`)
- Data files and scripts
- HTML/CSS/JS assets (for components)
- Respects `.kdepsignore` exclusions

**Examples:**

```bash
kdeps bundle package my-agent/                # Creates my-agent-1.0.0.kdeps
kdeps bundle package my-agency/               # Creates my-agency-1.0.0.kagency
kdeps bundle package my-component/            # Creates greeter-1.0.0.komponent
kdeps bundle package my-agent/ --output dist/
kdeps bundle package my-agent/ --name custom-agent
```

---

## `kdeps bundle build`

Build Docker image from workflow.

```bash
kdeps bundle build [path] [flags]
```

**Accepts:** directory, workflow file, or `.kdeps` package.

**Flags:**

| Flag | Description | Default |
|---|---|---|
| `--gpu` | GPU type: `cuda`, `rocm`, `intel`, `vulkan` (auto-selects Ubuntu base) | None (CPU, Alpine) |
| `--tag, -t` | Docker image tag | From workflow metadata |
| `--no-cache` | Build without cache | `false` |

**Examples:**

```bash
kdeps bundle build examples/chatbot                            # CPU, Alpine
kdeps bundle build examples/chatbot --gpu cuda                 # NVIDIA GPU
kdeps bundle build examples/chatbot --gpu rocm                 # AMD GPU
kdeps bundle build examples/chatbot --tag my-agent:v1.0.0
kdeps bundle build myapp-1.0.0.kdeps                          # From package
```

---

## `kdeps export iso`

Export a workflow as a bootable image (ISO, raw disk, or qcow2) using LinuxKit.

```bash
kdeps export iso [path] [flags]
```

See `kdeps export iso --help` for the full list of formats and flags.

---

## `kdeps export k8s`

Generate Kubernetes Deployment and Service manifests from a workflow.

```bash
kdeps export k8s [path] [flags]
```

**Arguments:** directory, workflow file, or `.kdeps` package.

**Flags:**

| Flag | Short | Description | Default |
|---|---|---|---|
| `--image` | `-i` | Container image name | `{name}:{version}` |
| `--output` | `-o` | Output file path | stdout |
| `--replicas` | `-r` | Number of replicas | From workflow |
| `--network-policy` | | Also generate an ingress-restricting NetworkPolicy | Off |

**Examples:**

```bash
kdeps export k8s examples/chatbot              # Print to stdout
kdeps export k8s examples/chatbot \
  --image my-registry/chatbot:v1.0.0 \
  --output k8s.yaml                            # Save to file
kdeps export k8s examples/chatbot --replicas 5 # Override replicas
```

Manifests are driven by `agentSettings` in `workflow.yaml`:
- `replicas` -- number of pod replicas
- `resources` -- CPU/memory limits and requests
- `env` -- container environment variables
- `networkPolicy: true` -- appends a NetworkPolicy restricting ingress to the configured ports
- `portNum` inside `apiServer:`/`webServer:` -- exposed ports
- `installOllama: true` -- adds Ollama backend port (11434)

See [Kubernetes Deployment](/deployment/kubernetes) for full details.

## See Also

- [CLI Overview](/reference/cli/) -- global flags, exit codes, env vars
- [Dev Commands](/reference/cli/dev) -- run, serve, validate, new
- [Docker Deployment](/deployment/docker)
- [Kubernetes Deployment](/deployment/kubernetes)
