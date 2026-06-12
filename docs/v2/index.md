---
layout: home

hero:
  name: kdeps
  text: Build and deploy AI agents in YAML.
  tagline: Workflow pipelines and autonomous agents in YAML. Export as Docker, Kubernetes, ISO, or a single binary. Local llamafile models out of the box - or Ollama, OpenAI, Anthropic, and any OpenAI-compatible backend.
  announcement: Skills for AI agents are here
  announcementLink: /getting-started/agent-skills
  actions:
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
  - title: Workflow mode
    details: Deterministic DAG pipelines. Resources run in dependency order defined by requires:. Predictable, auditable, testable.
  - title: Agent mode
    details: Autonomous LLM loop via kdeps serve. Every resource auto-registers as a tool. The LLM decides what to call and when.
  - title: Agencies
    details: Multi-agent orchestration. One agent calls another declaratively via the `agent:` resource type. Compose agents like functions.

  - title: Any backend
    details: Local llamafile models by default - no server install. Or Ollama, OpenAI, Anthropic, Groq, and any OpenAI-compatible endpoint. Switch backends in config without touching workflow files.
  - title: Component registry
    details: Install pre-built scraper, search, browser, and embedding components. Compose them into your workflow with one line.
  - title: No lock-in
    details: Apache 2.0 license. Standard YAML files in a git repo. Export your workflow and walk away — no proprietary format, no migration scripts.

  - title: Docker, K8s, ISO, binary export
    details: Export as a Docker image, Kubernetes manifests, bootable ISO, or a self-contained single binary. Same workflow, any target.
  - title: Validate and doctor
    details: kdeps validate checks schema, dependencies, and expressions. kdeps doctor diagnoses environment issues before you hit run.
  - title: Hot reload dev mode
    details: kdeps run --dev watches your files and reloads on change. Iterate fast without restarting the server.
---
