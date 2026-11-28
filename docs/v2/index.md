---
layout: home

hero:
  name: kdeps
  text: Serverless AI agents
  tagline: "Lambda-style AI agents on your own hardware. One workflow.yaml runs once per event, offline, and ships as an API, a binary, an image, or an ISO."
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
  - title: A function per event
    details: A request starts one bounded run, and an idle agent scales to zero. The logic is YAML only.
    link: /workflow/input-sources
    linkText: See the events
  - title: Behavior you can review
    details: A change is a pull request. kdeps test runs the evals.
    link: /workflow/testing
    linkText: Run the evals
  - title: Checks, then a bounded agent
    details: Checks run before the model. The agent step has a round limit.
    link: /workflow/resources/llm
    linkText: See the step
---
