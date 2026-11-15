---
layout: home

hero:
  name: kdeps
  text: Build git-native AI appliances
  tagline: "Your agent is YAML in your repository. You review it as a pull request. kdeps ships it with an open-source model to machines that you control: cloud, on-prem, edge, or air-gapped."
  announcement: Make kdeps YAML with Claude Code, Cursor, or Grok
  announcementLink: /agent/ai-assisted-authoring
  actions:
    - theme: brand
      text: What is kdeps?
      link: /start/
    - theme: alt
      text: Build a workflow
      link: /workflow/quickstart
    - theme: alt
      text: Run locally
      link: /agent/quickstart

features:
  - title: kdeps workflow
    details: A deterministic YAML pipeline. Serve it as an HTTP API, a bot, or a file processor. The same input always runs the same steps in the same order.
    link: /workflow/
    linkText: Build a workflow
  - title: kdeps agent
    details: Run `kdeps` to get an AI REPL with tools and memory. It uses a local model or an LLM server that you host.
    link: /agent/
    linkText: Explore the agent
  - title: kdeps deploy
    details: Ship the tested workflow as a Docker image, Kubernetes manifests, a bootable ISO, or one binary.
    link: /deploy/
    linkText: Ship it
---
