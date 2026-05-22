---
layout: home

hero:
  name: kdeps
  text: Build and deploy AI agents in YAML.
  tagline: Workflow pipelines and autonomous agents in YAML. Export as Docker, Kubernetes, ISO, or a single binary. Works with Ollama, OpenAI, Anthropic, and any OpenAI-compatible backend.
  image: false
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
  - icon: 🏗️
    title: Workflow Mode
    details: DAG-deterministic pipelines. Declare dependencies with `requires:`, compose chat, SQL, HTTP, Python, and shell into a single YAML workflow.
  - icon: 🤖
    title: Agent Mode
    details: Run `kdeps serve` to turn your workflow into an autonomous LLM loop. Every resource auto-registers as a callable tool. The LLM plans and executes multi-step tasks.
  - icon: 📦
    title: Self-Contained Exports
    details: Export AI workflows as a single binary, Docker image, bootable ISO, or Kubernetes manifests. Ship to production without glue code.
---
