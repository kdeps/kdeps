# Registry formula specification

A formula is the submission format for publishing a package to the kdeps registry. It is a YAML document that tells `kdeps registry install` where to download the package and how to verify it.

Works in workflow and agent mode.

**Formulas are the submission layer.** You PR a formula file to get listed. The registry at [kdeps.io](https://kdeps.io) runs on a database - formulas are ingested on merge and served from the DB for browsing, search, and install metadata.

Formulas live in `github.com/kdeps/registry` under `formulas/<name>.yaml`. To publish a package, open a PR adding your formula.

## Formula fields

```yaml
# formulas/my-agent.yaml
name: my-agent                         # required - unique package name in the registry
version: 1.2.0                         # required - semantic version
type: agent                            # required - component | workflow | agency
github: owner/repo                     # required - GitHub owner/repo containing the release
tarball: https://github.com/owner/repo/archive/refs/tags/v1.2.0.tar.gz  # required
sha256: abc123...                      # required - SHA256 of the tarball; computed by kdeps registry submit
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
| `sha256` | yes | Hex-encoded SHA256 of the tarball. Computed by `kdeps registry submit --tag`. |
| `description` | yes | One sentence describing what the package does. |
| `tags` | no | List of search keywords. Use lowercase, no spaces. |
| `license` | no | SPDX license identifier (`MIT`, `Apache-2.0`, etc.). |

## Package types

### Component

A reusable resource bundle invoked via `component:` in any workflow. Manifest: `kind: Component`, archive: `.komponent`.

```yaml
type: component
```

Example: `scraper`, `search`, `browser`, `tts`, `email`

### Workflow

A complete DAG pipeline that runs standalone via `kdeps run`. Manifest: `kind: Workflow`, archive: `.kdeps`.

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

kdeps publishes its own versioned assets (harness sections, events, actions, presets, themes, tool definitions, LLM server recipes, model registries, project templates) through the same registry. CI writes one formula per item, named `kdeps-<set>-<name>`; the `tarball` is the item's YAML file in `kdeps/packages`, not an archive. These are installed and updated with `kdeps update`, not `kdeps registry install`.

```yaml
# formulas/kdeps-harness-safety.yaml  (written by kdeps CI, not by hand)
name: kdeps-harness-safety
version: 1.0.0
type: harness            # harness | event | action | preset | theme | tool | recipe | model | template
github: kdeps/packages
tarball: https://raw.githubusercontent.com/kdeps/packages/main/assets/harness/safety/1.0.0.yaml
sha256: 5a2844c4...      # SHA256 of that YAML file
description: 'kdeps harness: safety'
tags: [kdeps, harness]
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

Run `kdeps registry submit` from your package directory:

```bash
cd my-package/
kdeps registry submit --tag v1.2.0
```

This downloads the tarball, computes the SHA256, and prints the formula YAML.

### 3. Submit a PR

Open a pull request to `github.com/kdeps/registry` adding the formula as `formulas/<name>.yaml`. One formula file per package per version. The PR is reviewed for:

- Unique name (no collisions)
- Valid SHA256 (tarball matches)
- Manifest kind matches formula type
- Description is present and meaningful

### 4. Install

Once merged, anyone can install:

```bash
kdeps registry install my-agent
```

## Repository layout

The registry repo structure:

```
registry/
  formulas/
    scraper.yaml
    search.yaml
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
