# Registry commands

Overview: [kdeps registry](/registry/).


Search, install, and publish packages from formula registries. Install, update, and remove kdeps assets (harness prompts, tool definitions, commands, templates, schemas, model registries). An argument that names an asset set (`tools`) or item (`tools/bash_exec`) is an asset. All other arguments are packages.

## Registries

A registry is a GitHub repository of formula files (`formulas/<name>.yaml`). kdeps reads `kdeps/registry` first, then each repository listed under `registries:`. It keeps a copy of each one under `~/.kdeps/registries/<owner>/<repo>`, so search, info and install need no server.

```yaml
# ~/.kdeps/config.yaml
registries:
  - acme/tools   # github.com/acme/tools, read after kdeps/registry
```

```text
search, info, install -> ~/.kdeps/registries/<owner>/<repo>/formulas -> tarball URL -> install
                          (downloaded from GitHub on first use)
```

| Name | Meaning |
|---|---|
| `scraper` | the formula `scraper`, first match in registry order |
| `acme/tools/scraper` | the formula `scraper` in registry `acme/tools` (downloaded on first use) |
| `owner/repo` | a GitHub repository itself, not a formula |

`KDEPS_REGISTRY_ARCHIVE_URL` replaces `https://codeload.github.com` as the source of registry archives, for a mirror or GitHub Enterprise.

## Commands

Each is a top-level `kdeps` command.

| Command | Description |
|---|---|
| `kdeps search` | Search the formulas of every registry: `TEXT` matches a name, description or tag without case; `/REGEX/` is a regular expression; `--type` and `--limit` narrow the list |
| `kdeps info` | Show what a kdeps asset is for and where it runs (`kdeps info tools/read_file`, `kdeps info events`), a formula and where it comes from, a local component or agent, or a GitHub repo's README |
| `kdeps install` | Install assets (`set`, `set/name[@version]`, with what they require), or a package from a formula (`name`, `owner/repo/name`; `@version` must be the formula's version), a GitHub repo (`owner/repo`; `github:owner/repo` when the owner is named like an asset set), or a local archive (`.kdeps` `.kagency` `.komponent`) |
| `kdeps uninstall` | Remove assets, or an installed agent or component |
| `kdeps update` | Download every registry again, then list the installed assets that have a newer version; installs nothing |
| `kdeps upgrade` | Upgrade assets (no args: the profile and every installed asset), or an installed agent or component to its formula's version |
| `kdeps list` | List installed and local components and agents, each with the tokens its agent-mode tool definition adds to the LLM context; `--assets` lists every asset with its version, state, shelf and context tokens; `kdeps list <shelf>` shows one shelf |
| `kdeps create URL` | Print a formula for the package archive at URL: kdeps downloads it, computes sha256 and reads `kdeps.pkg.yaml`; `--no-fetch` skips the download |
| `kdeps edit NAME...` | Open a formula in the local registry copy to test a change; prints the GitHub link to publish it |
| `kdeps verify` | Run LLM-agnostic verification on a package directory |
| `kdeps profile` | Choose what assets kdeps installs: `minimum`, `workflows`, `agent`, `deploy`, `full`, `custom`, or sets and items; no name opens the picker. Items the new profile leaves out are removed |

## LLM context tokens

`kdeps install`, `kdeps upgrade` and `kdeps list` estimate what each item adds to the model's context (about 4 bytes per token). A package installed from the registry is a tool of a chat loop, so its name, description, and input schema count:

```text
$ kdeps install scraper-native
✓ Installed scraper-native (workflow) @1.0.0
  Web scraping + AI summarization using the native scraper executor.
  LLM context: about 96 tokens as an agent-mode tool (its tool definition)
```

## Examples

```bash
kdeps search scraper
kdeps search '/^scraper-/'
kdeps install scraper-native
kdeps install acme/tools/scraper
kdeps install scraper-native@1.0.0
kdeps install ./my-component-1.0.0.komponent
kdeps list
kdeps info scraper-native
kdeps info tools/read_file           # an asset: its purpose, where kdeps runs it, its state
kdeps uninstall scraper-native
kdeps upgrade scraper-native
kdeps create https://github.com/acme/scraper/archive/refs/tags/v1.2.0.tar.gz
kdeps verify .

# kdeps assets
kdeps install tools/bash_exec      # one asset, with what it requires; prints ~tokens it adds to the LLM context
kdeps update                       # newest formulas; lists what kdeps upgrade would change
kdeps upgrade                      # every installed asset to its newest version
kdeps uninstall presets/frugal
kdeps list --assets                # version, state, shelf, ~context tokens
kdeps list rag                     # one shelf; the id stays tools/embedding_search
kdeps profile agent                # install the agent profile
```

## Publishing a package

A registry is a GitHub repository. Packages live in the author's own GitHub repo; the registry holds one formula file per package. Publish to `kdeps/registry`, or to a registry of your own that users list under `registries:`.

```bash
# 1. Tag a release in your repo
git tag v1.2.0 && git push --tags

# 2. Print the formula (downloads the archive, computes sha256, reads kdeps.pkg.yaml)
kdeps create https://github.com/<owner>/<repo>/archive/refs/tags/v1.2.0.tar.gz

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

- [Registry formula specification](/reference/registry/formula-spec) - formula file format and publishing workflow
- [CLI overview](/reference/cli) - global flags, exit codes, env vars
- [Components reference](/reference/agencies/component-reference) - component packaging and publishing
- [Packaging commands](/reference/deploy/cli) - bundle and build
