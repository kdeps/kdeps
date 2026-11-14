# konfig

konfig is one YAML file that fully describes the behavior of a kdeps agent. It is self-contained and declarative. It includes tuning, harness, themes, events, actions, presets, and skills. Export it from the current effective state (also from a default setup that you did not change). Then give the file to another machine. The file fully configures an agent on that machine, with no other setup.

## Exporting

```bash
kdeps konfig export [path]     # default ./konfig.yaml, from a bare CLI invocation
```

```
/konfig export [path]          # inside the REPL, exports the LIVE session's config
```

The CLI form reads the settings that kdeps keeps in `~/.kdeps/agent-loop-settings.yaml`. If you did not change a setting, it reads the built-in default. The REPL form records the exact state of the current session. This state includes changes from `/model tool set`, `/theme`, `/efficiency verbose on`, and other commands that do not save to a file.

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

kdeps exports the harness and themes as the full effective set, not as a diff from the built-ins. For this reason, the file alone sets the behavior on all machines and kdeps versions.

## Status

Export and import are both implemented, including the `--konfig <path>` startup flag.
