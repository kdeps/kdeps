---
layout: home

hero:
  name: kdeps
  text: Self-hosted apps with models inside
  tagline: Compose small YAML blocks into a dependency graph—models, scripts, HTTP, SQL. Run offline with open-source models, or call cloud APIs. Package once; ship the same artifact as Docker, Kubernetes, or a binary. Proud member of the NVIDIA Inception program.
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
  - title: Graph workflows
    details: Chain steps with requires:. kdeps orders the run and passes data between resources—no glue code for control flow.
  - title: Small reusable blocks
    details: Each resource is one unit (model call, script, HTTP, SQL). Remix the same blocks across projects.
  - title: Offline-ready
    details: llamafile and Ollama run on your machine. Keep data local for privacy, latency, and predictable cost.
  - title: Docker-first packaging
    details: Bundle workflow, deps, and model runtime into one image. Develop locally, deploy the same package anywhere.
  - title: Swap model backends
    details: Mix local and cloud endpoints in config. Change provider without rewriting the graph.
  - title: Full-stack endpoints
    details: Expose HTTP APIs and optional web UI from the same workflow. Validations and error paths built in.
  - title: Validate and doctor
    details: kdeps validate checks schema and dependencies. kdeps doctor checks the host before you run.
  - title: Dev reload
    details: kdeps run --dev reloads on file change so iteration stays fast.
  - title: Registry components
    details: Install scraper, search, browser, and embedding packs and wire them into the graph.
---
