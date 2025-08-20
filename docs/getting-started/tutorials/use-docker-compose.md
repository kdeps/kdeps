---
outline: deep
---

# Use the auto-generated Docker Compose file

When you run or build a KDeps agent, a Docker Compose file is generated to help you start the service quickly with the correct volumes and (optionally) GPU settings.

## Where the file is generated

- The file is created in your current working directory.
- Naming convention: `<agentName>_docker-compose-<gpu>.yaml`
  - Examples: `whois_docker-compose-cpu.yaml`, `whois_docker-compose-nvidia.yaml`, `whois_docker-compose-amd.yaml`

## What’s inside

- Service entry with the agent image name (e.g., `whois:1.0.0`)
- Volumes (external):
  - `ollama:/root/.ollama` (models and cache shared across runs)
  - `kdeps:/.kdeps` (agent data/state)
- Port mappings (if API/Web servers are enabled in your workflow)
- GPU configuration for `nvidia` or `amd` variants

## One-time setup: create external volumes

Compose declares external volumes; create them once:

```bash
docker volume create ollama
docker volume create kdeps
```

## Start the service

Use Docker Compose v2 (preferred):

```bash
docker compose -f whois_docker-compose-cpu.yaml up -d
```

Or with Docker Compose v1:

```bash
docker-compose --file whois_docker-compose-cpu.yaml up -d
```

Pick the file that matches your hardware: `-cpu.yaml`, `-nvidia.yaml`, or `-amd.yaml`.

## Stop and remove

```bash
docker compose -f whois_docker-compose-cpu.yaml down
```

## View logs

```bash
docker compose -f whois_docker-compose-cpu.yaml logs -f
```

## Ports and servers

Ports are included only if enabled in your workflow settings:

- API server: `Settings { APIServerMode = true; APIServer { HostIP; PortNum } }`
- Web server: `Settings { WebServerMode = true; WebServer { HostIP; PortNum } }`

To change ports, update your workflow and rebuild/re-run, or edit the compose file directly under `services.<name>.ports`.

## Offline Mode note

If `AgentSettings.OfflineMode = true`, model files are baked into the image during build and copied at runtime into the mounted volume `ollama:/root/.ollama`. This enables fully offline operation—ensure the `ollama` volume is mounted as in the compose file.

## GPUs

- Use `*_docker-compose-nvidia.yaml` for NVIDIA (requires NVIDIA runtime/driver)
- Use `*_docker-compose-amd.yaml` for AMD (maps `/dev/kfd` and `/dev/dri`)
- Use `*_docker-compose-cpu.yaml` for CPU-only

If your platform requires additional GPU runtime flags, extend the generated YAML accordingly.


