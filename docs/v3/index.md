---
layout: home
title: kdeps
hero:
  name: kdeps
  text: AI Appliances
  tagline: The book is the docs. Coding CLI agent first, then workflows and agencies in YAML. Release v2.1.11.
  image:
    src: /book-cover.jpg
    alt: AI Appliances book cover
  actions:
    - theme: brand
      text: Coding agent
      link: /chapter-05-agent-mode
    - theme: brand
      text: Getting started
      link: /chapter-02-getting-started
    - theme: alt
      text: Preface
      link: /frontmatter
    - theme: alt
      text: GitHub
      link: https://github.com/kdeps/kdeps
features:
  - title: Coding CLI agent
    details: Agent mode — REPL, tools, goals, permissions, workflows as tools. Start with chapter 5.
  - title: Workflow mode
    details: Deterministic DAGs, resources, expressions, bot and file inputs, agencies.
  - title: Ship
    details: Docker, Kubernetes, standalone binary, web server, LLM server appliances.
---

Source of truth: local `./book/` synced into `docs/v3/`. Re-sync with `docs/v3/scripts/sync-book.sh`. Also on [LeanPub](https://leanpub.com/kdeps).
