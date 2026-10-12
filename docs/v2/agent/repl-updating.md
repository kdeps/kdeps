# Updating kdeps

`/upgrade` checks GitHub and, for a standalone install, downloads and verifies the new binary. `/update` (and `kdeps update`) updates the [versioned assets](#updating-assets) - harness, tool definitions, slash commands, themes, recipes, model registries and templates - without a new binary. The other REPL behaviors stay on [REPL features](/agent/repl).

kdeps checks GitHub for a newer stable release at startup (throttled to once every 24 hours, cached at `~/.kdeps/update-check.json`, bounded to 3 seconds) and, if one exists, prints a one-line notice under the banner:

```
Update available: v2.8.0 -> v2.9.0. Run /upgrade to update.
```

Run `/upgrade` (or `kdeps --upgrade` outside the REPL) any time to check immediately and install. What happens next depends on how kdeps was installed:

- **Homebrew** (`brew install kdeps/tap/kdeps`): prints `brew upgrade kdeps` instead of touching the binary - self-replacing it would desync Homebrew's own bookkeeping.
- **.deb / .apk package**: prints the matching package-manager upgrade command.
- **Standalone** (the `curl | sh` installer, or a manually downloaded binary): after a `[Y/n]` confirmation (skippable with `KDEPS_YES=1`), downloads the release archive for your platform, verifies its SHA256 against `checksums.txt`, and atomically replaces the running binary. Restart kdeps afterward.

## Nightly builds

kdeps also cuts a nightly build from `main` most days. `/upgrade nightly` (or `kdeps --upgrade --nightly`) switches the channel for that one check: it installs the latest nightly instead of the latest stable.

Nightly opt-in only works for a standalone install - Homebrew/.deb/.apk only ever track stable, so on those `/upgrade nightly` prints standalone-install instructions instead of a package-manager command. "Already up to date" for the nightly channel means you are running that exact nightly tag: a nightly reuses the current stable version number until the next stable release ships, so it is always offered until you are actually on it.

## Installing a specific version (including a downgrade)

`/upgrade <version>` (or `kdeps --upgrade --target-version <version>` outside the REPL) installs exactly that version, skipping the "is an update available" check entirely - the version was named explicitly, so kdeps installs it whether it is newer than the running build (upgrade), older (downgrade), or the same (reinstall):

```
/upgrade 2.35.0
Downgrade to v2.35.0 now? [Y/n]
```

Same standalone-only restriction as nightly: Homebrew/.deb/.apk cannot be told to install a specific (especially older) version, so those print a link to the release page instead.

## Updating assets

kdeps' behavior files are versioned assets that update separately from the binary. They apply to both modes: harness sections, events, actions, presets, themes, tool definitions and slash commands shape agent mode; LLM server recipes and the `kdeps new` project templates are used in workflow mode.

| Set | What an item is | Example id |
|---|---|---|
| `harness` | A system-prompt section or internal prompt | `harness/safety` |
| `events` | An agent-loop event and its limits | `events/file-read-limit` |
| `actions` | An action an event can fire | `actions/block` |
| `presets` | A named bundle of harness/event settings | `presets/frugal` |
| `themes` | A REPL color theme | `themes/vim` |
| `tools` | A built-in tool's description and parameter schema (the code stays in the binary) | `tools/read_file` |
| `commands` | A REPL slash command: a built-in's help and aliases, or a [declared command](/agent/commands#declared-commands) run entirely from its steps | `commands/review` |
| `recipes` | An LLM server recipe | `recipes/vllm` |
| `models` | A model registry: every llamafile or GGUF alias (harvested nightly from HuggingFace) | `models/gguf` |
| `templates` | A `kdeps new` project template (all its files) | `templates/api-service` |

```d2
direction: right
main: "kdeps repo, main\nassets/<set>/" {shape: oval}
branch: "kdeps/packages assets/\n<set>/<name>/<version>.yaml\n+ index.json"
formulas: "kdeps/registry\nformulas/kdeps-<set>-<name>.yaml"
site: "kdeps.io\none catalog: packages,\nresource types, assets"
update: "kdeps update\n(first run: automatic, required)"
store: "~/.kdeps/assets/\nfiles + lock.json"
loader: "loaders\n(user overrides on top)" {shape: oval}
main -> branch: "publish on merge"
main -> formulas: "one formula per item"
formulas -> site: "indexed like any package"
branch -> update: "download + sha256 check"
update -> store
store -> loader
```

The `kdeps` binary contains none of these files: kdeps loads them only from `~/.kdeps/assets/`. On the first run (and after an upgrade that adds a set), any command that needs them downloads the newest version of every item first:

```text
$ kdeps
Downloading kdeps assets (harness, tools, commands, themes, templates, models)...
```

Offline with nothing installed, kdeps stops and tells you to connect and run `kdeps update` (`kdeps update`, `kdeps validate` and `kdeps registry` run without assets). Once installed, kdeps runs offline. Images built with `kdeps build` carry the build host's assets in `/app/.kdeps/assets`, so a container needs no download.

After that, a startup notice appears when updates are published (checked once a day, cached at `~/.kdeps/assets/check.json`):

```text
2 asset updates available (harness/safety 1.2.0, tools/read_file 1.1.0). Run kdeps update.
```

The desktop app shows the same as an **Updates available** button in the top bar.

```bash
kdeps update                          # every item to its newest version (skips pinned and removed items)
kdeps update tools                    # every item in one set
kdeps update harness/safety           # newest version of one item; unpins and restores it
kdeps update harness/safety@1.2.0     # exactly that version, pinned until you update it by name
kdeps update --check                  # what would change
kdeps update --list                   # every item: version, downloaded | pinned | removed
kdeps update --remove themes/vim,presets/frugal   # remove items
```

`/update`, `/update check`, `/update list` and `/update remove <item>...` do the same inside the REPL and apply right away; changed tool descriptions reach the model from the next start.

How a version is chosen:

- Each file starts with `version:` (semver) and may declare `kdeps: ">=X.Y.Z"`, the oldest kdeps that can load it. Versions that need a newer kdeps are skipped (`kdeps --upgrade` first).
- Every download is checked against the published sha256, and against the set's own parser before it replaces anything. A file edited by hand in `~/.kdeps/assets/` fails its checksum and is ignored; `kdeps update <item>` restores it.
- A file kdeps cannot read (a corrupt one, or one written for a newer kdeps) is skipped with a warning instead of stopping kdeps, so `kdeps update` can always fix it.

Removing an item hides it from kdeps until you update it by name again: a removed tool is not offered to the model, a removed harness section is not sent, a removed theme or template is not listed. A few items cannot be removed because kdeps needs them to work: `themes/normal`, the internal harness prompts (compaction, goals, judges, prompt refinement, the session handshake) the loop's control tools (`goal_task_complete`, `goal_task_fail`, `judge_verdict`, `session_handshake`) and the two model registries (`models/llamafile`, `models/gguf`). Turn a harness section off without removing it with `/harness disable <name>`.

Your own overrides still win: files in `~/.kdeps/harness/`, `~/.kdeps/themes/`, `~/.kdeps/llm-servers/` and the like are layered on top of whatever version is installed.

`KDEPS_ASSETS_URL` points downloads at a mirror laid out like `assets/` in kdeps/packages (`index.json` plus `<set>/<name>/<version>.yaml`); `KDEPS_ASSETS_URL=off` turns downloads off.

To work on assets, point `KDEPS_ASSETS_SRC` at a directory laid out like `assets/` in the kdeps repo (`<set>/<name>.yaml`, one directory per template). kdeps loads it under the store, so an edited file is used as soon as it has a higher `version:` than the installed one:

```bash
KDEPS_ASSETS_SRC=~/src/kdeps/assets ./kdeps   # run on a checkout's assets
```
