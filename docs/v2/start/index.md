---
title: What is kdeps?
description: What kdeps is, the problem it solves, and the smallest mental model before you run anything.
---

# What is kdeps?

kdeps runs serverless AI agents on your own hardware. Each event starts one bounded run of a `workflow.yaml`, as with a serverless function: an API call, a chat message, a webhook, a file, or a prompt. It ships as a Docker image, a binary, an ISO, or a Kubernetes deployment, and runs offline.

```text
workflow.yaml + resources/ -> kdeps run -> kdeps bundle -> appliance
```

## Read next

- [Full text](/reference/start/) - the complete page
- [Installation](/start/installation) - install the CLI
- [Quickstart](/workflow/quickstart) - build your first workflow
