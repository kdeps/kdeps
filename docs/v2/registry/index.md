---
title: kdeps registry
description: Find, install, and publish shared kdeps agents and components.
---

# kdeps registry

A registry is a GitHub repository of formula files that shares agents and
components by name. kdeps reads `kdeps/registry` and any registry you list
under `registries:` in `~/.kdeps/config.yaml`, from a local copy, with no
server. It is optional: building, running, and deploying an agent never
touch it.

```bash
kdeps search scraper            # find packages
kdeps install scraper-native    # add one to the current project
kdeps submit --tag v1.2.0       # generate a formula to publish yours via PR
```

**Not this?** To build the agent you want to publish, start with
[kdeps workflow](/workflow/). To ship a private agent to your own infrastructure
instead of the registry, see [kdeps deploy](/deploy/docker).

---

| Topic | Page |
|-------|------|
| Every registry command and flag | [Registry commands](/registry/cli) |
| The formula file format a published package uses | [Formula spec](/registry/formula-spec) |
