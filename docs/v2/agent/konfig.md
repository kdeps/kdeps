# konfig

konfig is one YAML file that is a total, self-contained, declarative description of a kdeps agent's behavior - tuning, harness, themes, events, actions, presets, and skills. Export it from the current effective state (even a completely default, never-customized setup) and hand the file to another machine to fully configure an agent there, no other setup needed.

## Exporting

```bash
kdeps konfig export [path]     # default ./konfig.yaml, from a bare CLI invocation
```

```
/konfig export [path]          # inside the REPL, exports the LIVE session's config
```

The CLI form reads whatever is persisted in `~/.kdeps/agent-loop-settings.yaml` (or built-in defaults if nothing was ever customized). The REPL form captures the running session's exact state instead, including anything changed with `/model tool set`, `/theme`, `/efficiency verbose on`, etc. that has not necessarily been touched via a command that persists it elsewhere.

## Importing

```bash
kdeps konfig import [path]     # default ./konfig.yaml
kdeps --konfig <path> [path]   # import, then start the agent loop
```

```
/konfig import [path]          # inside the REPL, applies immediately
```

Import writes each section to its usual place, replacing any built-in or user entry with the same name:

| Section | Written to |
|---|---|
| harness | `~/.kdeps/harness/<name>.yaml` |
| themes | `~/.kdeps/themes/<name>.yaml` |
| events | `~/.kdeps/events/<name>.yaml` |
| actions | `~/.kdeps/actions/<name>.yaml` |
| presets | `~/.kdeps/presets/<name>.yaml` |
| skills | `~/.kdeps/skills/<name>/SKILL.md` |
| tuning, registry, active theme | `~/.kdeps/agent-loop-settings.yaml` |

`kdeps konfig import` and `/konfig import` reload the harness, theme, event, action and preset registries and apply the theme immediately. `--konfig <path>` applies before startup reads settings, so the values take effect for that run. Imported skills load on the next process start.

## What is in the file

| Section | Contents |
|---|---|
| `tuning` | Every `/model tool set` knob - rounds, retries, compaction/fold thresholds, memory-graph leaf limits, web/bash/file/code call limits, turo settings, refine/handshake toggles |
| `harness` | Every tool-use/behavior-prompt section - all built-in sections plus any `~/.kdeps/harness/*.yaml` overrides, already merged by name |
| `themes` | Every REPL theme - built-in plus any `~/.kdeps/themes/*.yaml` overrides, merged by name, with every palette color fully resolved (never left blank to inherit from `normal` on import) |
| `events` | Every reactive LLM event (`auto-compact`, `fold`, and the round-count guards) - built-in plus any `~/.kdeps/events/*.yaml` overrides, merged by name. See [events](./events.md) |
| `actions` | Every registered action name (the fixed vocabulary events' `run:` fields draw from) - built-in plus any `~/.kdeps/actions/*.yaml` overrides, merged by name. See [events](./events.md) |
| `presets` | Every named harness/event bundle (`frugal`, `balanced`, `thorough`, ...) - built-in plus any `~/.kdeps/presets/*.yaml` overrides, merged by name. See [presets](./events.md#presets) |
| `activeTheme` | The currently selected theme's name |
| `skills` | Every loaded skill, with its full `SKILL.md` content inlined - skills travel with the file, not by path |
| `registry` | Enabled workflow/agency/component/skill lists, default model, model-name display mode, favorite models, custom OpenAI-compatible endpoints |

Harness and themes export as the full effective set, not a diff against built-ins, so the file alone determines behavior on any machine or kdeps version.

## Status

Export and import are both implemented, including the `--konfig <path>` startup flag.
