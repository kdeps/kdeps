---
layout: home

hero:
  name: kdeps
  text: Build git-native AI appliances
  tagline: "Your agent is YAML in your repo. Review it as a pull request, version it with a tag, install it with `owner/repo`. kdeps packages the workflow, tools, and open-source model into one self-contained deployment for cloud, on-prem, edge, or air-gapped - no per-token cost, no external AI dependency."
  announcement: Scaffold YAML from Claude Code, Cursor, or Grok
  announcementLink: /agent/ai-assisted-authoring
  actions:
    - theme: brand
      text: What is kdeps?
      link: /start/
    - theme: alt
      text: Run locally
      link: /agent/quickstart
    - theme: alt
      text: Build a workflow
      link: /workflow/quickstart

features:
  - title: kdeps agent
    details: Run `kdeps` and you are in an AI REPL - an autonomous agent with tool use and memory that works fully offline against a local model.
    link: /agent/
    linkText: Explore the agent
  - title: kdeps workflow
    details: A deterministic YAML pipeline. Each resource declares its dependencies and runs in a fixed DAG order - same input, same path, safe to run unattended.
    link: /workflow/
    linkText: Build a workflow
  - title: kdeps agencies
    details: One agent calls another declaratively via the agent resource type. Compose agents like functions - each runs independently, results flow back.
    link: /agencies/
    linkText: Compose agents
  - title: kdeps LLM server
    details: Provision a standalone OpenAI-compatible inference appliance - llamafile, Ollama, vLLM, TGI, and more. Any kdeps host uses it as a client over `/v1`.
    link: /llm-server/
    linkText: Run the appliance
  - title: kdeps deploy
    details: Export the workflow you tested locally as a Docker image, Kubernetes manifests, a bootable ISO, or a single binary. Same file, no rewrites, no re-config.
    link: /deploy/
    linkText: Ship it
  - title: kdeps registry
    details: Find, install, and publish shared agents and components. Pull a published workflow into your project with one command, or publish your own.
    link: /registry/
    linkText: Browse the registry
---
