# Registry formula specification

A formula is the submission format to publish a package to the kdeps registry. It is a YAML document that tells `kdeps install` where to download the package and how to verify it.

Works in workflow and agent mode.

**Formulas are the submission layer.** To put your package in the list, open a PR with a formula file. The registry at [kdeps.io](https://kdeps.io) runs on a database. On merge, the registry reads formulas into the database. It serves them from the database for browse, search, and install metadata.

Formulas live in `github.com/kdeps/registry` under `formulas/<name>.yaml`. To publish a package, open a PR adding your formula.

## Formula fields

```yaml
# formulas/my-agent.yaml
name: my-agent                         # required - unique package name in the registry
version: 1.2.0                         # required - semantic version
type: agent                            # required - component | workflow | agency
github: owner/repo                     # required - GitHub owner/repo containing the release
tarball: https://github.com/owner/repo/archive/refs/tags/v1.2.0.tar.gz  # required
sha256: abc123...                      # required - SHA256 of the tarball; computed by kdeps submit
description: A one-line summary.       # required
tags: [llm, chat]                      # optional - search keywords
license: Apache-2.0                    # optional - SPDX identifier
```

### Field reference

| Field | Required | Description |
|---|---|---|
| `name` | yes | Unique package name. Lowercase, alphanumeric + hyphens. |
| `version` | yes | Semantic version (`1.2.0`). One formula per version. |
| `type` | yes | `component`, `workflow`, or `agency`. Must match the manifest `kind` in the tarball. The asset types (`harness`, `event`, `action`, `preset`, `theme`, `tool`, `recipe`, `model`, `template`) are reserved for kdeps's own assets. |
| `github` | yes | `owner/repo` string. The repo containing the tagged release. |
| `tarball` | yes | Full URL to the release tarball. Use GitHub's `/archive/refs/tags/v{version}.tar.gz` pattern. |
| `sha256` | yes | Hex-encoded SHA256 of the tarball. Computed by `kdeps submit --tag`. |
| `description` | yes | One sentence describing what the package does. |
| `tags` | no | List of search keywords. Use lowercase, no spaces. |
| `license` | no | SPDX license identifier (`MIT`, `Apache-2.0`, etc.). |

## Package types

### Component

A reusable resource bundle. Call it with `component:` in a workflow. Manifest: `kind: Component`, archive: `.komponent`.

```yaml
type: component
```

Example: `search`, `tts`

### Workflow

A complete DAG pipeline that runs standalone with `kdeps run`. Manifest: `kind: Workflow`, archive: `.kdeps`.

```yaml
type: workflow
```

Example: `summarizer`, `classifier`, `invoice-extractor`

### Agency

A multi-agent orchestration bundle. Manifest: `kind: Agency`, archive: `.kagency`.

```yaml
type: agency
```

Example: `cv-matcher`, `research-pipeline`

### kdeps assets

kdeps publishes its own versioned assets through the same registry. The assets are harness sections, events, actions, presets, themes, tool definitions, slash commands, LLM server recipes, model registries, project templates, schemas, components, and providers. CI writes one formula for each item, with the name `kdeps-<set>-<name>`. The `tarball` is the YAML file of the item in `kdeps/packages`, not an archive. `kdeps update` installs and updates them, and the items under `requires:` of each item ([how](/agent/repl-updating)). The formula has those items as `dependencies`. The asset index records each item's [shelf](/reference/glossary#shelf) as `category`.

```yaml
# formulas/kdeps-components-web_search.yaml  (written by kdeps CI, not by hand)
name: kdeps-components-web_search
version: 1.0.1
type: kdeps-component    # harness | event | action | preset | theme | tool | command | recipe | model | template | ...
github: kdeps/packages
tarball: https://raw.githubusercontent.com/kdeps/packages/main/assets/components/web_search/1.0.1.yaml
sha256: 5a2844c4...      # SHA256 of that YAML file
description: 'kdeps kdeps-component: web_search'
tags: [kdeps, kdeps-component]
license: Apache-2.0
dependencies:            # the item's requires:, as package names; omitted when it has none
  kdeps-schemas-v1: ">=1.0.3"
```

### kdeps resource types

The registry also lists the built-in resource types (`httpClient:`, `chat:`, `exec:`, ...) as `type: resource`. As a result, you can browse them next to packages. They are part of the `kdeps` binary, and you do not install them. CI writes one formula for each type, with the name `kdeps-resource-<key>`. The `tarball` is the docs page of the type, with the same version as kdeps.

```yaml
# formulas/kdeps-resource-http-client.yaml  (written by kdeps CI, not by hand)
name: kdeps-resource-http-client
version: 2.58.2          # the kdeps release the page documents
type: resource
github: kdeps/packages
tarball: https://raw.githubusercontent.com/kdeps/packages/main/resources/httpClient/2.58.2.md
sha256: 25d97114...      # SHA256 of that page
description: The `httpClient:` resource makes an outbound HTTP request ...
tags: [kdeps, resource, httpClient]   # the last tag is the resource's YAML key
license: Apache-2.0
```

## Publishing workflow

### 1. Create a release

Tag your repo with a semantic version:

```bash
git tag v1.2.0 && git push --tags
```

The tag must match the `version` field in the formula.

### 2. Generate the formula

Run `kdeps submit` from your package directory:

```bash
cd my-package/
kdeps submit --tag v1.2.0
```

This downloads the tarball, computes the SHA256, and prints the formula YAML.

### 3. Submit a PR

Open a pull request to `github.com/kdeps/registry` adding the formula as `formulas/<name>.yaml`. One formula file per package per version. Reviewers check the PR for:

- Unique name (no collisions)
- Valid SHA256 (tarball matches)
- Manifest kind matches formula type
- Description is present and meaningful

### 4. Install

After the merge, all users can install it:

```bash
kdeps install my-agent
```

## Repository layout

The registry repo structure:

```
registry/
  formulas/
    scraper-native.yaml
    search-web-native.yaml
    summarizer.yaml
    cv-matcher.yaml
    kdeps-harness-safety.yaml   # kdeps assets, written by kdeps CI
    ...
  README.md
```

No nested directories. Flat `formulas/` folder. One file per package name. Version bumps update the existing file.

## See also

- [Registry commands](/registry/cli) - install, search, publish, verify
- [Components reference](/agencies/component-reference) - component.yaml reference
- [Packaging commands](/deploy/cli) - archive formats
