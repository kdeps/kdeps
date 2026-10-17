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

kdeps' behavior files are versioned assets that update separately from the binary. They apply to both modes: harness sections, events, actions, presets, themes, tool definitions and slash commands shape agent mode; LLM server recipes, the `kdeps new` project templates, the workflow schemas, executor defaults, kdeps components and the Docker and Kubernetes templates are used in workflow mode.

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
| `models` | A model registry: every llamafile or GGUF alias (harvested nightly from HuggingFace), or the cloud model catalog `models/cloud` (each provider's model IDs with context window, output limit, thinking and image support) | `models/gguf`, `models/cloud` |
| `templates` | A `kdeps new` project template (all its files) | `templates/api-service` |
| `schemas` | The JSON schemas `kdeps validate`, `run` and `package` check workflows, resources, components and agencies against (all four files) | `schemas/v1` |
| `defaults` | Executor defaults (timeouts, SQL pool sizes, search limits) used when neither the resource nor `config.yaml` sets a value | `defaults/executors` |
| `docker` | The Dockerfile, entrypoint and supervisord templates `kdeps build` renders (all five files) | `docker/image` |
| `k8s` | The Deployment, Service and NetworkPolicy templates `kdeps export k8s` renders (all three files) | `k8s/manifests` |
| `components` | A kdeps component (YAML, no Go code) a workflow calls with `component:`, e.g. [ocr](/workflow/resources/ocr) | `components/ocr` |
| `providers` | An [OpenAI-compatible LLM provider](/workflow/resources/llm-backends#openai-compatible-providers): chat endpoint, key variable and config key | `providers/groq` |

```d2
direction: right
main: "kdeps repo, main\nassets/<set>/" {shape: oval}
branch: "kdeps/packages assets/\n<set>/<name>/<version>.yaml\n+ index.json"
formulas: "kdeps/registry\nformulas/kdeps-<set>-<name>.yaml"
site: "kdeps.io\none catalog: packages,\nresource types, assets"
update: "kdeps update\n(first run: the chosen profile)"
store: "~/.kdeps/assets/\nfiles + lock.json"
loader: "loaders\n(user overrides on top)" {shape: oval}
main -> branch: "publish on merge"
main -> formulas: "one formula per item"
formulas -> site: "indexed like any package"
branch -> update: "download + sha256 check"
update -> store
store -> loader
```

The `kdeps` binary contains none of these files: kdeps loads them only from `~/.kdeps/assets/`. The first run asks what you will use kdeps for and installs that profile:

```text
$ kdeps
What will you use kdeps for? (change it later with kdeps profile)
  1) minimum                kdeps run and validate: chat, httpClient, python, exec, apiResponse
  2) workflows  ~3.0k tok    minimum, kdeps new templates and the components (sql, browser, email, ...)
  3) agent      ~11.1k tok   minimum and the agent loop: prompts, slash commands, themes, file/shell/web tools
  4) deploy     ~3.0k tok    workflows, and kdeps build/export: Docker and Kubernetes templates, LLM server recipes
  5) full       ~19.4k tok   every asset set
  6) custom                 tick the asset sets, or single items, yourself
  (~tok: what the profile adds to the LLM context - prompts and tool definitions)
Choice [5]:
```

| Profile | Sets it installs |
|---|---|
| `minimum` | `schemas`, `defaults`, `models`, `providers` (the base: every other profile includes it) |
| `workflows` | base + `templates`, `components` |
| `agent` | base + `harness`, `commands`, `themes`, `events`, `actions`, `presets`, the loop's required tools and the file, shell, web and memory tools |
| `deploy` | base + `templates`, `components`, `docker`, `k8s`, `recipes` |
| `full` | every set |
| `custom` | a checklist of every set and item: tick whole sets, or open a set to tick single items |

```text
Asset sets ([x] all, [-] some; the base sets are always installed):
   1) [ ] harness        0/43  items
   ...
   6) [-] tools          6/78  items  ~1.0k tok
   8) [x] recipes       10/10  items
   9) [x] models         3/3   items (base)
LLM context of this selection: about 1.0k tokens (prompts, tool definitions).
Toggle sets (e.g. 5 7), o<N> to pick items in set N (e.g. o6), Enter to install: o6

tools (6/78 ticked; * required):
    1) [ ] approval_grant
    5) [x] bash_exec  ~159 tok
   ...
   29) [x] goal_task_complete *  ~172 tok
Toggle items (e.g. 3 5-9), a = all, n = none, Enter to go back:
```

Each asset says what it is for and where kdeps runs it. `kdeps info <set>/<name>` prints that, and the same flow appears on the asset's kdeps.io page:

```text
$ kdeps info tools/read_file
# tools/read_file
Read a file from the local filesystem. ...
- Set: tools, used in agent mode
- Loaded by: kdeps (agent loop), /reload, /settings
- Installed: 1.0.0
- LLM context: ~631 tokens when loaded

## Where it runs
     the agent loop builds the request
  -> [ the tool definitions are sent to the model ]   <- tools/read_file
  -> the model calls a tool
  -> kdeps runs it
  -> the result goes back to the model

In the code: kdeps (agent loop) -> agent.RegisterBuiltinTools -> agent.definedAs
```

The story is written once, as a `//kdeps:flow` line on the function where kdeps uses the set; the bracketed step is the one the set drives:

```go
//kdeps:flow tools: the agent loop builds the request > [the tool definitions are sent to the model] > the model calls a tool > kdeps runs it > the result goes back to the model
func definedAs(id string, t *kdepstools.Tool) *kdepstools.Tool {
```

Everything else is read from kdeps's call graph by `tools/assetflow`. It finds where each set is loaded, follows the value into the variables it is stored in and the functions that read them, and walks back to the commands and slash commands that reach them: that gives "Loaded by" (the agent loop and workflow commands, and the commands nearest the use), the mode, and "In the code" (the call path to the function the story is on). It also checks the story: a set without one, with two, or with one on a function that does not lead to a use of the set fails the run, naming the function to move it to. The asset publish job regenerates all of it from the code it publishes, so the index and kdeps.io match main; `make generate` rewrites `pkg/assets/setflow_gen.go` for `kdeps info`, and CI fails while it is out of date. A set's one-sentence purpose is in `pkg/assets/setinfo.go`, and each item's purpose is its own `description:`.

Every count marked `~tok` estimates what an item adds to the model's context when kdeps loads it (about 4 bytes per token): harness sections are system-prompt text, tool definitions and components are tool schemas the model sees on every turn. Themes, events, schemas, templates and the other sets never reach the model and show none. The same estimate is published with each version in the asset index, shown on kdeps.io, printed by `kdeps install` and `kdeps update`, and totalled by `kdeps list --assets`:

```text
$ kdeps install tools/bash_exec
  tools/bash_exec                      install   - -> 1.0.0  ~159 tok
LLM context: these added about 159 tokens (prompts, tool definitions) when loaded.
```

A set number toggles the whole set; `o<N>` opens it. Inside, numbers and ranges toggle items, `a` ticks all and `n` none. An item marked `*` is one its set's loader needs: it stays ticked while anything else in the set is. A base set keeps at least its required items (or one item). A fully ticked set is recorded by name, so items published to it later are installed by `kdeps update`; a partly ticked one is recorded item by item.

Without a terminal (CI, Docker, scripts) there is no question: `KDEPS_PROFILE` names the profile (`KDEPS_PROFILE=agent`, or a list of sets and items such as `KDEPS_PROFILE=docker,k8s,tools/bash_exec`), and without it kdeps installs only the sets the command loads. The profile is recorded in `~/.kdeps/assets/lock.json`.

The profile is respected afterwards. A command that loads a set the profile leaves out installs only the items that set's loader requires, never the whole set: `kdeps` (the agent loop) under `minimum` adds 8 core commands (`help`, `exit`, `clear`, `model`, `compact`, `session`, `settings`, `update`), the `normal` theme and the loop's required tools and harness, so `/help` lists those 8. Sets with nothing required (`events`, `actions`, `presets`) stay empty and are recorded as empty on purpose in `lock.json`, so the next start makes no network call. Every built-in command runs without its file; `kdeps install commands/goal` lists `/goal` in `/help`, `kdeps install commands` lists them all. A required item cannot be uninstalled.

```text
$ KDEPS_PROFILE=minimum kdeps
Installing the minimum profile...
Installing harness/compaction-system, ..., tools/goal_task_complete, ..., commands/help, ..., themes/normal (needed by this command)...
```

Switching to a smaller profile prunes: `kdeps profile minimum` after `agent` or `full` deletes every downloaded item the new profile does not cover (`commands/goal  remove  not in the profile`), and a set it leaves out entirely loses its required items too, except the base sets and the agent loop's. An item you install by name later is kept by `kdeps update`.

Without a chosen profile (no terminal and no `KDEPS_PROFILE`), a command installs each set it loads whole the first time it needs it (`kdeps export` installs `docker` and `k8s`). Either way, a workflow that calls a component not installed downloads it on that call:

```text
$ kdeps run workflow.yaml
Installed components/sql 1.0.2 (first use).
```

Offline, a missing set stops the command and names the update to run (`kdeps assets docker are not installed; connect to the internet and run kdeps update docker`). `kdeps update` and `kdeps registry` run without assets. Once installed, kdeps runs offline. Images built with `kdeps build` carry the build host's assets in `/app/.kdeps/assets`, so a container needs no download.

After that, a startup notice appears when updates are published (checked once a day, cached at `~/.kdeps/assets/check.json`):

```text
2 asset updates available (harness/safety 1.2.0, tools/read_file 1.1.0). Run kdeps update.
```

The desktop app shows the same as an **Updates available** button in the top bar.

```bash
kdeps update                          # the profile and everything installed since, to their newest versions (skips pinned and removed items)
kdeps profile agent          # install a profile (or sets and items: docker,tools/bash_exec) and keep it current
kdeps profile                 # choose the profile again, or tick sets and items (custom)
kdeps update tools                    # every item in one set
kdeps update harness/safety           # newest version of one item; unpins and restores it
kdeps update harness/safety@1.2.0     # exactly that version, pinned until you update it by name
kdeps update --check                  # what would change
kdeps list --assets                   # every item: version, downloaded | pinned | removed, ~context tokens; total at the end
kdeps info tools/read_file            # what an item is for, where kdeps runs it, its installed version and ~tokens
kdeps info events                     # a set: its purpose, where it runs, its items
kdeps uninstall themes/vim,presets/frugal   # remove items
```

`/update`, `/update check`, `/update list` and `/update remove <item>...` do the same inside the REPL and apply right away; changed tool descriptions reach the model from the next start.

How a version is chosen:

- Each file starts with `version:` (semver) and may declare `kdeps: ">=X.Y.Z"`, the oldest kdeps that can load it. Versions that need a newer kdeps are skipped (`kdeps --upgrade` first).
- A file may list other items it needs under `requires:`, each with the versions it accepts. Installing it installs those first, at their newest version in range, when they are missing, removed or too old - and theirs, recursively:

  ```yaml
  # components/searchWeb/template.yaml (a bundle's header; a single-file item has these at its top)
  version: 1.0.1
  requires:
    schemas/v1: ">=1.0.3"   # its component.yaml uses connection:, added in schemas/v1 1.0.3
  ```

  ```text
  $ kdeps update components/searchWeb
    schemas/v1                           install   - -> 1.0.3  (required by components/searchWeb)
    components/searchWeb                 install   - -> 1.0.1
  ```

  An item whose requirement is not published for your kdeps, or is pinned to a version outside the range, fails instead of installing; update the pinned item by name to unpin it. Events require the action they run, presets the events and actions they tune, the `/frugal` and `/thorough` commands their preset, components the schema version their `component.yaml` needs. Publishing rejects a requirement no published version satisfies.
- Every download is checked against the published sha256, and against the set's own parser before it replaces anything. A file edited by hand in `~/.kdeps/assets/` fails its checksum and is ignored; `kdeps update <item>` restores it.
- A file kdeps cannot read (a corrupt one, or one written for a newer kdeps) is skipped with a warning instead of stopping kdeps, so `kdeps update` can always fix it.

Removing an item hides it from kdeps until you update it by name again: a removed tool is not offered to the model, a removed harness section is not sent, a removed theme or template is not listed. A few items cannot be removed because kdeps needs them to work: `themes/normal`, the internal harness prompts (compaction, goals, judges, prompt refinement, the session handshake) the loop's control tools (`goal_task_complete`, `goal_task_fail`, `judge_verdict`, `session_handshake`) and the two model registries (`models/llamafile`, `models/gguf`). Turn a harness section off without removing it with `/harness disable <name>`.

Your own overrides still win: files in `~/.kdeps/harness/`, `~/.kdeps/themes/`, `~/.kdeps/llm-servers/` and the like are layered on top of whatever version is installed.

`KDEPS_ASSETS_URL` points downloads at a mirror laid out like `assets/` in kdeps/packages (`index.json` plus `<set>/<name>/<version>.yaml`); `KDEPS_ASSETS_URL=off` turns downloads off.

To work on assets, point `KDEPS_ASSETS_SRC` at a directory laid out like `assets/` in the kdeps repo (`<set>/<name>.yaml`, one directory per template). kdeps loads it under the store, so an edited file is used as soon as it has a higher `version:` than the installed one:

```bash
KDEPS_ASSETS_SRC=~/src/kdeps/assets ./kdeps   # run on a checkout's assets
```
