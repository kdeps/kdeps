# Installation

Install the `kdeps` CLI to build and run workflows locally. Docker is optional: you need it only to build container images.

```bash
brew install kdeps/tap/kdeps                                   # macOS
scoop bucket add kdeps https://github.com/kdeps/scoop-bucket   # Windows
scoop install kdeps
```

Linux packages (deb, rpm, apk, Arch), a curl script and `go install` are on the [installation reference](/reference/start/installation).

The first run asks what you use kdeps for and downloads the matching asset profile (prompts, tool definitions, templates). Change it later with `kdeps profile`. Then check the setup:

```bash
kdeps --version
kdeps config      # backend, models, folders and registries
kdeps doctor      # config, Ollama, Python and agents
```

## Read next

- [Installation reference](/reference/start/installation) - installing the kdeps CLI, macOS (Homebrew), windows (Scoop), linux packages (deb, rpm, apk, Arch), linux, macOS, and Windows (curl), windows (PowerShell), windows (wget in WSL or Git Bash), from source
- [Quickstart](/workflow/quickstart) - build your first workflow API
- [Agent loop](/workflow/agent-loop/) - a chat that calls tools
- [CLI reference](/reference/cli) - Complete command reference
- [Workflow configuration](/workflow/configuration) - Learn about workflow settings
- [Examples](https://github.com/kdeps/kdeps/tree/main/examples) - Browse example workflows
