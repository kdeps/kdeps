---
layout: home
title: kdeps
hero:
  name: kdeps
  text: Workflows from the CLI
  tagline: Scaffold, validate, run. YAML DAGs for APIs and agents — local models or cloud, same commands.
  actions:
    - theme: brand
      text: Quickstart
      link: /quickstart
    - theme: brand
      text: CLI
      link: /cli
    - theme: alt
      text: Workflow mode
      link: /workflow
    - theme: alt
      text: GitHub
      link: https://github.com/kdeps/kdeps
features:
  - title: CLI loop
    details: "`kdeps new` → `validate` → `run --dev`. Package with `kdeps bundle` when you ship."
  - title: Workflow mode
    details: Deterministic DAG. Resources declare `requires:`; one `kdeps run` serves the API or other inputs.
  - title: Agent mode
    details: "`kdeps [path]` for a tool-using REPL. Workflows under the path become callable tools."
---
