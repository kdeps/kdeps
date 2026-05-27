# Registry Commands

Search, install, and publish packages from the kdeps registry.

## `kdeps registry`

```bash
kdeps registry <subcommand> [flags]
```

**Subcommands:**

| Subcommand | Description |
|---|---|
| `search` | Search for packages in the kdeps registry |
| `info` | Show metadata and README for a package, component, agent, or GitHub repo |
| `install` | Install from the registry, a GitHub repo (`owner/repo`), or a local archive (`.kdeps` `.kagency` `.komponent`) |
| `uninstall` | Uninstall an agent or component installed from the registry |
| `update` | Update an installed agent or component to a newer version |
| `list` | List installed and local components |
| `submit` | Generate a registry formula YAML for submitting a package via GitHub PR |
| `verify` | Run LLM-agnostic verification on a package directory |

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
```

## Publishing a Package

The kdeps registry is GitHub-hosted. Packages live in the author's own GitHub repo; the registry indexes a formula file per package.

```bash
# 1. Tag a release in your repo
git tag v1.2.0 && git push --tags

# 2. Generate the formula YAML (downloads the tarball and computes SHA256)
kdeps registry submit --tag v1.2.0

# 3. Open a PR to https://github.com/kdeps-io/registry
#    adding the printed formula as formulas/<your-package-name>.yaml
```

### Formula File Format

```yaml
# workflow.yaml
name: my-agent
version: 1.2.0
type: agent
github: owner/my-agent-repo
tarball: https://github.com/owner/my-agent-repo/archive/refs/tags/v1.2.0.tar.gz
sha256: <computed-by-kdeps-registry-submit>
description: ...
tags: [llm, chat]
license: Apache-2.0
```

`kdeps registry install` downloads from the GitHub tarball URL and verifies the SHA256 locally.

## See Also

- [Registry Formula Specification](/reference/registry-formula-spec) -- formula file format and publishing workflow
- [CLI Overview](/reference/cli/) -- global flags, exit codes, env vars
- [Components Reference](/reference/components) -- component packaging and publishing
- [Packaging Commands](/reference/cli/packaging) -- bundle and build
