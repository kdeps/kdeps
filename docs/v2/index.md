---
layout: home

hero:
  name: kdeps
  text: One YAML file, offline
  tagline: "kdeps turns one workflow.yaml into an offline API, a binary, an image, or an ISO. You review the YAML as a pull request, and kdeps test runs the evals. A check returns 400 before any model call. A chat with loop: is a bounded step in that file."
  announcement: Your AI on a USB stick
  announcementLink: /reference/deploy/cli#kdeps-bundle-prepackage
  actions:
    - theme: brand
      text: What is kdeps?
      link: /start/
    - theme: alt
      text: Build a workflow
      link: /workflow/quickstart
    - theme: alt
      text: Ship it offline
      link: /deploy/

features:
  - title: One file, four outputs
    details: The same workflow.yaml runs as an HTTP API, one binary, a container image, or a bootable ISO.
    link: /deploy/
    linkText: Ship the file
  - title: Behavior you can review
    details: A YAML change is a pull request. kdeps test runs the evals before you tag a release.
    link: /workflow/testing
    linkText: Run the evals
  - title: Checks, then a bounded agent
    details: A failed check returns 400 before any model call. A chat with loop is one step in that file, and maxRounds stops it.
    link: /workflow/resources/llm
    linkText: See the step
---
