---
layout: home

hero:
  name: kdeps
  text: AI Appliance Builder
  tagline: YAML-defined AI agents and workflow pipelines. Ship as Docker, K8s, ISO, or a single binary.
  announcement: Scaffold YAML from Claude Code, Cursor, or Grok
  announcementLink: /getting-started/agent-skills
  actions:
    - theme: brand
      text: What is kdeps?
      link: /getting-started/introduction
    - theme: brand
      text: Run locally
      link: /getting-started/local-agent
    - theme: alt
      text: Build a workflow
      link: /getting-started/quickstart

features:
  - title: Local AI agent
    details: Run `kdeps` and you are in an AI REPL - an autonomous agent with tool use and memory that works fully offline against a local model.
  - title: Deterministic workflows
    details: Each resource declares its dependencies and runs in a fixed DAG order. Same input, same execution path - auditable, testable, safe to run unattended.
  - title: Multi-agent agencies
    details: One agent calls another declaratively via the agent resource type. Compose agents like functions - each runs independently, results flow back.
  - title: Any LLM backend
    details: Switch backends with one line. llamafile and Ollama need no server install or API key; OpenAI, Anthropic, Groq, and any OpenAI-compatible endpoint work too. Auto-router picks the best installed model with cloud fallback.
  - title: Deploy anywhere
    details: Export the workflow you tested locally as a Docker image, Kubernetes manifests, a bootable ISO, or a single binary. Same file, no rewrites, no re-config.
  - title: Build with AI assistance
    details: Install the kdeps skill and ask Claude Code, Cursor, or any coding agent to scaffold workflows, components, and agencies. It knows the full schema.
---
