---
outline: deep
---

# Data Folder

The `data` folder is a dedicated location within your AI agent project for storing files required by the agent during its
operations. It ensures access to files and resources during execution.

## Packaging and Deployment
When you `package` your AI agent, the data folder is included in the `.kdeps` package. Upon building the Docker image,
the contents of the folder are extracted as well. This ensures that all necessary files are available and accessible
when the agent runs in its containerized environment.

## File Organization
During packaging, the data folder is organized by the name and version of the AI agent. For example:

```bash
data/aiagentx/1.0.0/<files_here>
```

### Included AI agents

If your AI agent uses other AI agents through the `workflows {...}` configuration, the data folders for those
agents are also included. Each agent's files are organized by its name and version, maintaining a consistent structure:

```bash
data/anotheragent/2.1.3/<files_here>
data/aiagentx/1.0.0/<files_here>
```
