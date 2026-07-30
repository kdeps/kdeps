---
layout: home

hero:
  name: kdeps
  text: Run AI workflows locally. Or deploy them anywhere.
  tagline: Install kdeps, run `kdeps`, get an AI agent - no API key needed with Ollama or llamafile. Build your workflow in YAML. Deploy as Docker, Kubernetes, or a single binary when you're ready. Proud member of the NVIDIA Inception program.
  announcement: Skills for AI agents are here
  announcementLink: /getting-started/agent-skills
  actions:
    - theme: brand
      text: Try locally
      link: /getting-started/local-agent
    - theme: brand
      text: Get Started
      link: /getting-started/installation
    - theme: alt
      text: Why kdeps?
      link: /concepts/why-kdeps
    - theme: alt
      text: Registry
      link: https://kdeps.io

features:
  - title: Local AI agent
    details: Run `kdeps` and you are in an AI REPL. Use Ollama or llamafile for a fully offline, private coding agent - no API key, no cloud dependency.
  - title: Portable workflows
    details: Write a workflow.yaml once. Run it locally with `kdeps run`. Deploy it as Docker, Kubernetes, or a binary when you are ready - same file, no rewrites.
  - title: Agencies
    details: Multi-agent orchestration. One agent calls another declaratively via the `agent:` resource type. Compose agents like functions.

  - title: Any backend, no lock-in
    details: llamafile and Ollama work out of the box - no server install, no API key. Or use OpenAI, Anthropic, Groq, and any OpenAI-compatible endpoint. Switch backends in config without touching workflow files.
  - title: Component registry
    details: Install pre-built scraper, search, browser, and embedding components. Compose them into your workflow with one line.
  - title: Deploy anywhere
    details: Export as a Docker image, Kubernetes manifests, bootable ISO, or a self-contained single binary. Apache 2.0 license. Standard YAML in a git repo - no proprietary format.

  - title: Validate and doctor
    details: kdeps validate checks schema, dependencies, and expressions. kdeps doctor diagnoses environment issues before you hit run.
  - title: Hot reload dev mode
    details: kdeps run --dev watches your files and reloads on change. Iterate fast without restarting the server.
  - title: Skills for AI agents
    details: A coding-agent skill teaches Claude Code, Cursor, and other agents how to scaffold kdeps workflows, components, and agencies automatically.
---
