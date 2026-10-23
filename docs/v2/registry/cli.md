# Registry commands

Search, install, and publish packages from the kdeps registry, and install, update and remove kdeps assets (harness prompts, tool definitions, commands, themes, templates, schemas, model registries). An argument naming an asset set (`tools`) or item (`tools/bash_exec`) is an asset; anything else is a package.

## Commands

Each is a top-level `kdeps` command; `--registry <url>` (or `KDEPS_REGISTRY_URL`) points it at another registry.

| Command | Description |
|---|---|
| `kdeps search` | Search for packages in the kdeps registry |
| `kdeps info` | Show what a kdeps asset is for and where it runs (`kdeps info tools/read_file`, `kdeps info events`), or the metadata and README of a package, component, agent, or GitHub repo |
| `kdeps install` | Install assets (`set`, `set/name[@version]`, with what they require), or a package from the registry, a GitHub repo (`owner/repo`; `github:owner/repo` when the owner is named like an asset set), or a local archive (`.kdeps` `.kagency` `.komponent`) |
| `kdeps uninstall` | Remove assets, or an agent or component installed from the registry |
| `kdeps update` | Update assets (no args: the profile and every installed asset; `--check` lists updates), or an installed agent or component |
| `kdeps list` | List installed and local components and agents, each with the tokens its agent-mode tool definition adds to the LLM context; `--assets` lists every asset with its version, state, shelf and context tokens; `kdeps list <shelf>` shows one shelf |
| `kdeps submit` | Generate a registry formula YAML for submitting a package via GitHub PR |
| `kdeps verify` | Run LLM-agnostic verification on a package directory |
| `kdeps profile` | Choose what assets kdeps installs: `minimum`, `workflows`, `agent`, `deploy`, `full`, `custom`, or sets and items; no name opens the picker. Items the new profile leaves out are removed |

## LLM context tokens

`kdeps install`, `kdeps update` and `kdeps list` estimate what each item adds to the model's context (about 4 bytes per token). A package installed from the registry is a tool in agent mode, so its name, description and input schema count:

```text
$ kdeps install scraper-native
✓ Installed scraper-native (workflow) @1.0.0
  Web scraping + AI summarization using the native scraper executor.
  LLM context: about 96 tokens as an agent-mode tool (its tool definition)
```

## Examples

```bash
kdeps search scraper
kdeps install scraper-native
kdeps install scraper-native@1.0.0
kdeps install ./my-component-1.0.0.komponent
kdeps list
kdeps info scraper-native
kdeps info tools/read_file           # an asset: its purpose, where kdeps runs it, its state
kdeps uninstall scraper-native
kdeps update scraper-native
kdeps submit --tag v1.2.0
kdeps verify .

# kdeps assets
kdeps install tools/bash_exec      # one asset, with what it requires; prints ~tokens it adds to the LLM context
kdeps update                       # every installed asset to its newest version
kdeps update --check               # what would change
kdeps uninstall themes/vim
kdeps list --assets                # version, state, shelf, ~context tokens
kdeps list rag                     # one shelf; the id stays tools/embedding_search
kdeps profile agent                # install the agent profile
```

## Publishing a package

The kdeps registry is GitHub-hosted. Packages live in the author's own GitHub repo; the registry indexes a formula file per package.

```bash
# 1. Tag a release in your repo
git tag v1.2.0 && git push --tags

# 2. Generate the formula YAML (downloads the tarball and computes SHA256)
kdeps submit --tag v1.2.0

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

`kdeps install` downloads from the GitHub tarball URL and verifies the SHA256 locally.

## See also

- [Registry formula specification](/registry/formula-spec) - formula file format and publishing workflow
- [CLI overview](/reference/cli) - global flags, exit codes, env vars
- [Components reference](/agencies/component-reference) - component packaging and publishing
- [Packaging commands](/deploy/cli) - bundle and build
