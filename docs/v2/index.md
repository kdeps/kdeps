---
layout: home

hero:
  name: kdeps
  text: Coding agent, then workflows
  tagline: Run kdeps for a tool-using CLI agent. Point it at a project for workflows-as-tools. Ship YAML pipelines as Docker, Kubernetes, ISO, or a binary when you need a fixed API. Proud member of the NVIDIA Inception program.
  actions:
    - theme: brand
      text: Coding agent
      link: /getting-started/local-agent
    - theme: brand
      text: Install
      link: /getting-started/installation
    - theme: alt
      text: Agent mode
      link: /modes/agent-loop-mode
    - theme: alt
      text: Workflow mode
      link: /modes/workflow-mode

features:
  - title: Coding CLI agent
    details: Run kdeps for an AI REPL with tools, goals, and memory. Use llamafile or Ollama offline — no API key required.
  - title: Workflows
    details: Write a workflow.yaml once. Run with kdeps run. Deploy Docker, Kubernetes, or a binary — same files.
  - title: Agencies
    details: Multi-agent systems that call each other via the agent resource. Compose specialists, not monoliths.
  - title: Any backend
    details: llamafile and Ollama out of the box, or OpenAI, Anthropic, Groq, and any OpenAI-compatible endpoint.
  - title: Component registry
    details: Install scraper, search, browser, and embedding components. Compose with one line.
  - title: Ship anywhere
    details: Docker, Kubernetes, bootable ISO, or a self-contained binary. Apache 2.0. Standard YAML in git.
  - title: Validate and doctor
    details: kdeps validate checks schema, deps, and expressions. kdeps doctor checks the environment before you hit run.
  - title: Dev reload
    details: kdeps run --dev watches files and reloads. Iterate without restarting the server.
  - title: Agent skills
    details: A coding-agent skill teaches Claude Code, Cursor, and friends how to scaffold kdeps projects.
---
