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
      text: Get started
      link: /getting-started/quickstart
    - theme: brand
      text: Run locally
      link: /getting-started/local-agent
    - theme: alt
      text: Why kdeps?
      link: /concepts/why-kdeps

features:
  - title: Local AI agent
    details: Run `kdeps` and you are in an AI REPL. Use Ollama or llamafile for a fully offline, private coding agent - no API key, no cloud dependency.
  - title: Deterministic workflows
    details: Each resource declares its dependencies and runs in a fixed DAG order. Same input, same execution path - auditable, testable, safe to run unattended.
  - title: Multi-agent agencies
    details: One agent calls another declaratively via the agent resource type. Compose agents like functions - each runs independently, results flow back.
  - title: Any LLM backend
    details: llamafile and Ollama work with no server install or API key. Or use OpenAI, Anthropic, Groq, and any OpenAI-compatible endpoint. Auto-router picks the best installed model with cloud fallback.
  - title: Deploy anywhere
    details: Export the workflow you tested locally as a Docker image, Kubernetes manifests, bootable ISO, or a single binary. Same file, no rewrites. Apache 2.0, standard YAML in a git repo.
  - title: Build with AI assistance
    details: Install the kdeps skill and ask Claude Code, Cursor, or any coding agent to scaffold workflows, components, and agencies. It knows the full schema.
---
