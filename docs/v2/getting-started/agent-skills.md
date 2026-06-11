# Agent Skills

Coding agents (Claude Code, Cursor, Grok, and others) can scaffold kdeps
projects for you when you install the **kdeps skill**.

## Install the skill

Clone into your agent's skills directory (folder name `kdeps`):

```bash
# Claude Code
git clone https://github.com/kdeps/skill ~/.claude/skills/kdeps

# Cursor
git clone https://github.com/kdeps/skill ~/.cursor/skills/kdeps

# Grok
git clone https://github.com/kdeps/skill ~/.grok/skills/kdeps
```

Symlink instead of clone if you prefer. The skill activates when you ask the
agent to build a kdeps workflow, component, or agency.

## What it covers

- Choosing between a component, workflow (agent), or agency
- All 15 primary resource actions plus `apiResponse`
- Workflow input (`api`, `bot`, `file`), `webServer`, session, and `kdeps serve`
- Expressions, validation, and error handling
- **Registry-ready packaging** — every scaffold includes `kdeps.pkg.yaml` for
  [kdeps.io](https://kdeps.io) distribution

## Publishing to kdeps.io

Projects the skill creates are distributable by default:

```bash
kdeps validate .
kdeps registry verify .
kdeps bundle package .
git tag v1.0.0 && git push --tags
kdeps registry submit --tag v1.0.0
# Open a PR to https://github.com/kdeps/registry with the printed formula
```

See the skill's [registry reference](https://github.com/kdeps/skill/blob/main/references/registry.md)
for the full publish checklist.

## Skill vs registry packages

| | **kdeps skill** | **Your kdeps project** |
|---|---|---|
| What it is | Instructions for coding agents | Runnable workflow / component / agency |
| Install | `git clone` into skills dir | `kdeps registry install <name>` |
| Manifest | `SKILL.md` | `kdeps.pkg.yaml` |

The skill teaches agents how to write YAML that installs from
[kdeps.io](https://kdeps.io). It is not itself a registry package.

## See also

- [Registry](https://kdeps.io) — browse and install community packages
- [Components](/concepts/components) — reusable resource bundles
- [Agencies](/concepts/agency) — multi-agent orchestration
- [Skill repository](https://github.com/kdeps/skill) — source and test fixtures