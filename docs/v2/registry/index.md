---
title: kdeps registry
description: Find, install, and publish shared kdeps agents and components.
---

# kdeps registry

**An optional package registry for kdeps agents and components.** Pull a
published workflow, agency, or component into your project with one command, or
publish your own for others to reuse. Building, running, and deploying your own
agent never touches the registry - it is purely a way to share work by name.

```bash
kdeps registry search scraper            # find packages
kdeps registry install scraper           # add one to the current project
kdeps registry submit --tag v1.2.0       # generate a formula to publish yours via PR
```

**Not this?** To build the agent you want to publish, start with
[kdeps workflow](/workflow/). To ship a private agent to your own infrastructure
instead of the registry, see [kdeps deploy](/deploy/docker).

---

| Topic | Page |
|-------|------|
| Every `kdeps registry` subcommand and flag | [Registry commands](/registry/cli) |
| The formula file format a published package uses | [Formula spec](/registry/formula-spec) |
