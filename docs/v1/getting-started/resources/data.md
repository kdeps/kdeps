---
outline: deep
---

# Data Folder

The `data` folder serves as a dedicated space within your AI agent project to store files and resources that the agent requires during its operations. This structure ensures access to these assets during runtime.

## Packaging and Deployment
When you package your AI agent, the `data` folder is automatically included in the resulting `.kdeps` package. During the Docker image build process, the folder's contents are extracted, guaranteeing that all necessary files are available and ready for use when the agent runs in its containerized environment.

## File Organization
To maintain clarity and consistency, the `data` folder is organized based on the AI agent's name and version. For instance:

```bash
data/aiagentx/1.0.0/<files_here>
```

### Including Dependencies
If your AI agent depends on other AI agents through the `Workflows {...}` configuration, the data folders for those agents are also packaged. Each agent’s files are stored separately, structured by its name and version:

```bash
data/anotheragent/2.1.3/<files_here>
data/aiagentx/1.0.0/<files_here>
```

## Accessing `data` Files

To simplify file access within the resources and the containerized environment, KDeps provides a helper function:
`data.filepath("agentName/version", "filename")`. This function takes two parameters: the agent name with its version
and the file name.

For example, to access a `file.txt` from a dependent AI agent named `anotheraiagent`:

```apl
local fileTxt = "@(data.filepath("anotheraiagent/2.1.3", "file.txt"))"
```

You can use the retrieved file path with the `read` function to access its content. For instance, you might want to
source a `python` or `shell` script stored in the `data` folder and execute it within each resource.

```apl
local fileContent = """
@(read("\(fileTxt)")?.text)
"""
```
