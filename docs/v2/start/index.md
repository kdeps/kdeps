---
title: What is kdeps?
description: What kdeps is, the problem it solves, and the smallest mental model before you run anything.
---

# What is kdeps?

kdeps turns a `workflow.yaml` into an offline AI appliance: an API, a listener (chat bots, webhooks), a file job, or a prompt. Each event starts one run. It ships as a Docker image, a binary, an ISO, or a Kubernetes deployment.

```text
workflow.yaml + resources/ -> kdeps run -> kdeps bundle -> appliance
```

## Read next

- [Full text](/reference/start/) - the complete page
- [Installation](/start/installation) - install the CLI
- [Quickstart](/workflow/quickstart) - build your first workflow
