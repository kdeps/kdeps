# Updating kdeps

`/upgrade` checks GitHub and, for a standalone install, downloads and verifies the new binary. `/update` (and `kdeps update`) updates the [versioned assets](#updating-assets) - harness, tool definitions, themes, recipes and templates - without a new binary. The other REPL behaviors stay on [REPL features](/agent/repl).

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

kdeps' behavior files are versioned assets that update separately from the binary. They apply to both modes: harness sections, events, actions, presets, themes and tool definitions shape agent mode; LLM server recipes and the `kdeps new` project templates are used in workflow mode.

| Set | What an item is | Example id |
|---|---|---|
| `harness` | A system-prompt section or internal prompt | `harness/safety` |
| `events` | An agent-loop event and its limits | `events/file-read-limit` |
| `actions` | An action an event can fire | `actions/block` |
| `presets` | A named bundle of harness/event settings | `presets/frugal` |
| `themes` | A REPL color theme | `themes/vim` |
| `tools` | A built-in tool's description and parameter schema (the code stays in the binary) | `tools/read_file` |
| `recipes` | An LLM server recipe | `recipes/vllm` |
| `templates` | A `kdeps new` project template (all its files) | `templates/api-service` |

```d2
direction: right
main: "kdeps repo, main" {shape: oval}
branch: "kdeps/packages assets/\n<set>/<name>/<version>.yaml\n+ index.json"
formulas: "kdeps/registry\nformulas/kdeps-<set>-<name>.yaml"
site: "kdeps.io\npackages + asset catalog"
update: "kdeps update\n(first run: automatic)"
store: "~/.kdeps/assets/\nfiles + lock.json"
loader: "loaders\n(user overrides on top)" {shape: oval}
main -> branch: "publish on merge"
main -> formulas: "one formula per item"
formulas -> site: "indexed like any package"
branch -> update: "download + sha256 check"
update -> store
store -> loader: "newer than built-in,\nor pinned"
```

Every item ships compiled into the binary, so kdeps works offline. On the first run, kdeps downloads the newest version of every item (bounded to 20 seconds; offline it starts on the built-in versions and tries again next start). After that, a startup notice appears when updates are published (checked once a day, cached at `~/.kdeps/assets/check.json`):

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
kdeps update --list                   # every item: version, built-in | downloaded | pinned | removed
kdeps update --remove themes/vim,presets/frugal   # remove items
```

`/update`, `/update check`, `/update list` and `/update remove <item>...` do the same inside the REPL and apply right away; changed tool descriptions reach the model from the next start.

How a version is chosen:

- Each file starts with `version:` (semver) and may declare `kdeps: ">=X.Y.Z"`, the oldest kdeps that can load it. Versions that need a newer kdeps are skipped (`kdeps --upgrade` first).
- Every download is checked against the published sha256, and against the set's own parser before it replaces anything. A file edited by hand in `~/.kdeps/assets/` fails its checksum and the built-in version is used instead.
- A download is used only while it is newer than the built-in version or pinned, so upgrading the binary never leaves you on older assets.

Removing an item hides it from kdeps until you update it by name again: a removed tool is not offered to the model, a removed harness section is not sent, a removed theme or template is not listed. A few items cannot be removed because kdeps needs them to work: `themes/normal`, the internal harness prompts (compaction, goals, judges, prompt refinement, the session handshake) and the loop's control tools (`goal_task_complete`, `goal_task_fail`, `judge_verdict`, `session_handshake`). Turn a harness section off without removing it with `/harness disable <name>`.

Your own overrides still win: files in `~/.kdeps/harness/`, `~/.kdeps/themes/`, `~/.kdeps/llm-servers/` and the like are layered on top of whatever version is installed.

`KDEPS_ASSETS_URL` points downloads at a mirror laid out like `assets/` in kdeps/packages (`index.json` plus `<set>/<name>/<version>.yaml`); `KDEPS_ASSETS_URL=off` turns downloads off.
