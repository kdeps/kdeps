# Registry commands

Search, install, and publish packages from the kdeps registry, and install, update and remove kdeps assets (harness prompts, tool definitions, commands, themes, templates, schemas, model registries). An argument naming an asset set (`tools`) or item (`tools/bash_exec`) is an asset; anything else is a package.

## `kdeps registry`

```bash
kdeps registry <subcommand> [flags]
```

**Subcommands:**

| Subcommand | Description |
|---|---|
| `search` | Search for packages in the kdeps registry |
| `info` | Show metadata and README for a package, component, agent, or GitHub repo |
| `install` | Install assets (`set`, `set/name[@version]`, with what they require), or a package from the registry, a GitHub repo (`owner/repo`), or a local archive (`.kdeps` `.kagency` `.komponent`) |
| `uninstall` | Remove assets, or an agent or component installed from the registry |
| `update` | Update assets (no args: the profile and every installed asset; `--check` lists updates), or an installed agent or component |
| `list` | List installed and local components; `--assets` lists every asset, its version and state |
| `submit` | Generate a registry formula YAML for submitting a package via GitHub PR |
| `verify` | Run LLM-agnostic verification on a package directory |
| `profile` | Choose what assets kdeps installs: `minimum`, `workflows`, `agent`, `deploy`, `full`, `custom`, or sets and items; no name opens the picker |

## Examples

```bash
kdeps registry search scraper
kdeps registry install scraper
kdeps registry install scraper@2.1.0
kdeps registry install jjuliano/kdeps-component-scraper
kdeps registry install ./scraper-1.0.0.komponent
kdeps registry list
kdeps registry info scraper
kdeps registry uninstall scraper
kdeps registry update scraper
kdeps registry submit --tag v1.2.0
kdeps registry verify .

# kdeps assets
kdeps registry install tools/bash_exec      # one asset, with what it requires
kdeps registry update                       # every installed asset to its newest version
kdeps registry update --check               # what would change
kdeps registry uninstall themes/vim
kdeps registry list --assets
kdeps registry profile agent                # install the agent profile
```

## Publishing a package

The kdeps registry is GitHub-hosted. Packages live in the author's own GitHub repo; the registry indexes a formula file per package.

```bash
# 1. Tag a release in your repo
git tag v1.2.0 && git push --tags

# 2. Generate the formula YAML (downloads the tarball and computes SHA256)
kdeps registry submit --tag v1.2.0

# 3. Open a PR to https://github.com/kdeps/registry
#    adding the printed formula as formulas/<your-package-name>.yaml
```

### Formula file format

```yaml
# formulas/my-agent.yaml
name: my-agent
version: 1.2.0
type: workflow
github: owner/my-agent-repo
tarball: https://github.com/owner/my-agent-repo/archive/refs/tags/v1.2.0.tar.gz
sha256: <computed-by-kdeps-registry-submit>
description: ...
tags: [llm, chat]
license: Apache-2.0
```

`kdeps registry install` downloads from the GitHub tarball URL and verifies the SHA256 locally.

## See also

- [Registry formula specification](/registry/formula-spec) - formula file format and publishing workflow
- [CLI overview](/reference/cli) - global flags, exit codes, env vars
- [Components reference](/agencies/component-reference) - component packaging and publishing
- [Packaging commands](/deploy/cli) - bundle and build
