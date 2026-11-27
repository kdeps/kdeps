---
layout: home

hero:
  name: kdeps
  text: One YAML file, offline
  tagline: "kdeps turns one workflow.yaml into an offline AI appliance. An API call, a chat message or a webhook starts one bounded run."
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
    details: Ship the same file as an API, a binary, an image, or an ISO.
    link: /deploy/
    linkText: Ship the file
  - title: Behavior you can review
    details: A change is a pull request. kdeps test runs the evals.
    link: /workflow/testing
    linkText: Run the evals
  - title: Checks, then a bounded agent
    details: Checks run before the model. The agent step has a round limit.
    link: /workflow/resources/llm
    linkText: See the step
---
