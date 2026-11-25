---
title: kdeps registry
description: Find, install, and publish shared kdeps agents and components.
---

# kdeps registry

A registry is a GitHub repository of formula files that shares agents and
components by name. kdeps reads `kdeps/registry` and any registry you list
under `registries:` in `~/.kdeps/config.yaml`, from a local copy, with no
server.

```bash
kdeps search scraper            # find packages
kdeps install scraper-native    # add one to the current project
kdeps create <archive-url>      # print a formula to publish yours via PR
```

## Read next

- [kdeps registry reference](/reference/registry/) - every key, option and detail
