---
outline: deep
---

# Workflow

The `workflow.pkl` contains configuration about the AI Agent, namely:

- AI agent `name`, `description`, `website`, `authors`, `documentation` and `repository`.
- *required*: The [semver](https://semver.org) `version` of this AI agent.
> **Note on version:**
> kdeps uses the version for mapping the graph-based dependency workflow execution order. For this reason, the version
> is *required*.

- *required*: The `action` resource to be executed when running the AI agent. This is the ID of the resource.
- *optional*: Existing AI agents `workflows` to be reused in this AI agent. The agent needed to be installed first via `kdeps
  install` command.
