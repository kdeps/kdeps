---
outline: deep
---

# System Configuration

Kdeps requires system-level configuration to determine how it operates on your machine. This configuration controls runtime behavior, GPU settings, file storage locations, and other core system parameters.

## Configuration File Setup

When you execute the `kdeps` command for the first time, it automatically creates a configuration file at `~/.kdeps.pkl` with default settings optimized for most development environments.

**Default Configuration:**
```apl
amends "package://schema.kdeps.com/core@0.1.30#/Kdeps.pkl"

Mode = "docker"
dockerGPU = "cpu"
kdepsDir = ".kdeps"
kdepsPath = "user"
```

These settings define key operational parameters for Kdeps runtime behavior, including execution mode, GPU configuration, and directory structure.

## Configuration Properties

### Mode

Specifies the execution environment for Kdeps agents.

```apl
Mode = "docker"  // Currently the only supported mode
```

**Current Support:**
- **`docker`**: Runs agents in Docker containers (default and recommended)

**Future Planned Modes:**
- **`local`**: Direct execution on bare-metal systems (planned)
- **`cloud`**: Cloud-based execution environments (planned)

> **Note:**
> Currently, Kdeps only supports Docker mode. This provides isolated, reproducible environments for AI agents with consistent dependency management across different systems.

### DockerGPU

Configures GPU acceleration support for Docker containers running AI models.

```apl
dockerGPU = "cpu"     // CPU-only execution (default)
dockerGPU = "nvidia"  // NVIDIA GPU acceleration
dockerGPU = "amd"     // AMD GPU acceleration
```

**GPU Configuration Options:**

- **`cpu`**: CPU-only execution, compatible with all systems but slower for AI workloads
- **`nvidia`**: NVIDIA GPU acceleration using CUDA, requires NVIDIA drivers and Docker GPU support
- **`amd`**: AMD GPU acceleration using ROCm, requires AMD drivers and ROCm support

**Important Considerations:**
- The Docker image will be built specifically for the specified GPU type
- Ensure your system has appropriate drivers installed before using GPU acceleration
- GPU acceleration significantly improves performance for LLM inference and AI processing
- Test with CPU mode first if you're unsure about GPU compatibility

**GPU Setup Requirements:**

For **NVIDIA GPUs:**
```bash
# Install NVIDIA Container Toolkit
sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit
sudo systemctl restart docker
```

For **AMD GPUs:**
```bash
# Install ROCm support
sudo apt-get update
sudo apt-get install -y rocm-dkms rocm-dev
```

### KdepsDir

Defines the directory name where Kdeps stores its files and cache data.

```apl
kdepsDir = ".kdeps"  // Default hidden directory
```

**Directory Structure:**
The Kdeps directory contains several important subdirectories:

```
.kdeps/
├── agents/          # Downloaded and installed agents
├── cache/           # Cached models and data
├── logs/            # Execution logs
├── temp/            # Temporary files
└── config/          # Agent-specific configurations
```

**Custom Directory Examples:**
```apl
kdepsDir = "kdeps-data"      // Visible directory
kdepsDir = ".ai-agents"      // Alternative hidden directory
kdepsDir = "workspace"       // Project-specific workspace
```

### KdepsPath

Determines the base location where the Kdeps directory is created.

```apl
kdepsPath = "user"     // $HOME/.kdeps (default)
kdepsPath = "project"  // Current project directory
kdepsPath = "xdg"      // XDG-compliant directory
```

**Path Options Explained:**

- **`user`**: Creates Kdeps directory in user home directory (`$HOME/.kdeps`)
  - Best for: Personal development, shared configurations across projects
  - Example: `/home/username/.kdeps`

- **`project`**: Creates Kdeps directory in current working directory
  - Best for: Project-specific configurations, team collaboration
  - Example: `/home/username/Projects/my-ai-agent/.kdeps`

- **`xdg`**: Uses XDG Base Directory specification
  - Best for: Linux systems following XDG standards
  - Example: `$XDG_DATA_HOME/.kdeps` or `$HOME/.local/share/.kdeps`

**Choosing the Right Path:**

| Use Case | Recommended Setting | Benefits |
|----------|-------------------|----------|
| Personal Development | `user` | Shared across all projects, easy management |
| Team Projects | `project` | Project-specific, version controllable |
| Linux Standards | `xdg` | Follows system conventions, better organization |

## Environment Variables

### Global Timeout Configuration

The `TIMEOUT` environment variable sets the global default timeout for all resource operations, overriding individual `TimeoutDuration` settings in PKL files.

**Configuration:**
```bash
# In .env file or shell environment
TIMEOUT=120  # Wait up to 120 seconds for operations
TIMEOUT=0    # Unlimited timeout (use with caution)
```

**Timeout Behavior:**
- **`TIMEOUT=<n>`** (n > 0): Wait up to *n* seconds for operations
- **`TIMEOUT=0`**: Unlimited timeout (no timeout enforcement)
- **Not set**: Falls back to default 60-second timeout

**Affected Operations:**
- LLM chat and inference requests
- HTTP client requests to external APIs
- Python script execution
- Shell command execution (`exec` resources)

**Use Cases:**
- **Slow Networks**: Increase timeout for high-latency environments
- **Large Models**: Allow more time for model loading and inference
- **Development**: Use longer timeouts during testing and debugging
- **Production**: Set conservative timeouts for reliability

**Example Configurations:**

```bash
# Development environment - longer timeouts
TIMEOUT=300

# Production environment - conservative timeouts  
TIMEOUT=30

# High-performance environment - quick failures
TIMEOUT=10

# Research environment - unlimited time for complex operations
TIMEOUT=0
```

## Configuration Examples

### Development Setup

Optimized for local development with CPU-only execution:

```apl
amends "package://schema.kdeps.com/core@0.1.30#/Kdeps.pkl"

Mode = "docker"
dockerGPU = "cpu"
kdepsDir = ".kdeps"
kdepsPath = "project"  // Project-specific configurations
```

### Production Setup with NVIDIA GPU

Configured for production deployment with GPU acceleration:

```apl
amends "package://schema.kdeps.com/core@0.1.30#/Kdeps.pkl"

Mode = "docker"
dockerGPU = "nvidia"
kdepsDir = ".kdeps"
kdepsPath = "user"     // Shared configurations
```

### Multi-User Server Setup

Configured for multi-user environments with XDG compliance:

```apl
amends "package://schema.kdeps.com/core@0.1.30#/Kdeps.pkl"

Mode = "docker"
dockerGPU = "nvidia"
kdepsDir = "kdeps"     // Visible directory for easier management
kdepsPath = "xdg"      // XDG-compliant paths
```

## Configuration Management

### Per-Environment Configuration

You can maintain different configurations for different environments:

```bash
# Development
cp ~/.kdeps.pkl ~/.kdeps.dev.pkl

# Production  
cp ~/.kdeps.pkl ~/.kdeps.prod.pkl

# Switch configurations
cp ~/.kdeps.prod.pkl ~/.kdeps.pkl
```

### Version Control Considerations

**Include in Version Control:**
- Project-specific configurations (when using `kdepsPath = "project"`)
- Documentation of required configuration settings
- Environment variable templates

**Exclude from Version Control:**
- User-specific configurations (`~/.kdeps.pkl`)
- Cache directories (`.kdeps/cache/`)
- Sensitive environment variables

### Backup and Recovery

```bash
# Backup current configuration
cp ~/.kdeps.pkl ~/.kdeps.pkl.backup

# Restore configuration
cp ~/.kdeps.pkl.backup ~/.kdeps.pkl

# Reset to defaults (requires restart)
rm ~/.kdeps.pkl
kdeps  # Will regenerate with defaults
```

## Troubleshooting

### Common Configuration Issues

**GPU Not Working:**
1. Verify GPU drivers are installed
2. Check Docker GPU support
3. Confirm `dockerGPU` setting matches your hardware

**Permission Errors:**
1. Check directory permissions for Kdeps path
2. Ensure Docker daemon is running
3. Verify user is in `docker` group (Linux)

**Timeout Issues:**
1. Increase `TIMEOUT` environment variable
2. Check network connectivity for external APIs
3. Monitor system resources during execution

### Configuration Validation

```bash
# Test configuration
kdeps --version

# Verify Docker GPU support
docker run --gpus all nvidia/cuda:11.0-base nvidia-smi  # For NVIDIA
docker run --device=/dev/kfd --device=/dev/dri rocm/rocm-terminal  # For AMD
```

## Next Steps

- **[Workflow Configuration](./workflow.md)**: Configure individual AI agents
- **[CORS Configuration](./cors.md)**: Set up cross-origin request handling
- **[Web Server Configuration](./webserver.md)**: Configure frontend integration
- **[Quickstart Guide](../introduction/quickstart.md)**: Build your first AI agent

Proper system configuration is essential for optimal Kdeps performance. Take time to configure GPU acceleration and appropriate timeout settings for your specific environment and use case.
