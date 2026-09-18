# konfig

*Applies to agent mode.*

konfig is one YAML file that is a total, self-contained, declarative description of a kdeps agent's behavior - tuning, harness, themes, and skills. Export it from the current effective state (even a completely default, never-customized setup) and hand the file to another machine to fully configure an agent there, no other setup needed.

## Exporting

```bash
kdeps konfig export [path]     # default ./konfig.yaml, from a bare CLI invocation
```

```
/konfig export [path]          # inside the REPL, exports the LIVE session's config
```

The CLI form reads whatever is persisted in `~/.kdeps/agent-loop-settings.yaml` (or built-in defaults if nothing was ever customized). The REPL form captures the running session's exact state instead, including anything changed with `/model tool set`, `/theme`, `/goal on`, etc. that hasn't necessarily been touched via a command that persists it elsewhere.

## What's in the file

| Section | Contents |
|---|---|
| `tuning` | Every `/model tool set` knob - rounds, retries, compaction/fold thresholds, memory-graph leaf limits, web/bash/file/code call limits, turo settings, goal/refine/handshake toggles |
| `harness` | Every tool-use/behavior-prompt section - all built-in sections plus any `~/.kdeps/harness/*.yaml` overrides, already merged by name |
| `themes` | Every REPL theme - built-in plus any `~/.kdeps/themes/*.yaml` overrides, merged by name, with every palette color fully resolved (never left blank to inherit from `normal` on import) |
| `activeTheme` | The currently selected theme's name |
| `skills` | Every loaded skill, with its full `SKILL.md` content inlined - skills travel with the file, not by path |
| `registry` | Enabled workflow/agency/component/skill lists, default model, model-name display mode, favorite models, custom OpenAI-compatible endpoints |

Harness and themes are exported as the **full effective set**, not a diff against the binary's built-ins - the file alone fully determines behavior on any machine or kdeps version, independent of what that machine's own `~/.kdeps` happens to contain.

## Status

Export is implemented. `kdeps konfig import <path>` / `/konfig import <path>` (materializing a konfig file to `~/.kdeps/harness`, `~/.kdeps/themes`, `~/.kdeps/skills`, and `~/.kdeps/agent-loop-settings.yaml`, plus a `--konfig <path>` startup flag) is planned but not yet shipped.
