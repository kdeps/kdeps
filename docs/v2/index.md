---
layout: home

hero:
  name: kdeps
  text: YAML pipelines for models
  tagline: Define steps in YAML, run them locally, ship the same files as Docker, Kubernetes, or a binary. Works with local models or cloud APIs. Proud member of the NVIDIA Inception program.
  actions:
    - theme: brand
      text: Install
      link: /getting-started/installation
    - theme: brand
      text: Try locally
      link: /getting-started/local-agent
    - theme: alt
      text: Quickstart
      link: /getting-started/quickstart
    - theme: alt
      text: Why kdeps?
      link: /concepts/why-kdeps

features:
  - title: Terminal REPL
    details: Run kdeps for a local chat loop with tools. llamafile or Ollama work offline without an API key.
  - title: Workflow YAML
    details: Declare steps and dependencies once. kdeps run executes the graph. Same files for local and deploy.
  - title: Multi-agent packages
    details: Split work across agents and call them with the agent resource. Reuse packaged units.
  - title: Swap backends
    details: llamafile and Ollama by default, or OpenAI-compatible, Anthropic, Groq, and similar endpoints via config.
  - title: Component registry
    details: Install scraper, search, browser, and embedding components and wire them into a workflow.
  - title: Package and deploy
    details: Docker, Kubernetes, ISO, or a single binary. Apache 2.0. Plain YAML in git.
  - title: Validate and doctor
    details: kdeps validate checks schema and dependencies. kdeps doctor checks the host environment.
  - title: Dev reload
    details: kdeps run --dev reloads on file change so you can iterate without a full restart.
  - title: Editor skills
    details: Optional skill files help Claude Code, Cursor, and similar tools scaffold kdeps projects.
---
