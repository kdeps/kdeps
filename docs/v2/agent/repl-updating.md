# Updating kdeps

`/upgrade` installs a new kdeps binary. `/update` (and `kdeps update`) updates the [versioned assets](#updating-assets): harness, tool definitions, slash commands, themes, recipes, model registries and templates. Other REPL behaviors are on [REPL features](/agent/repl).

At startup kdeps checks GitHub for a newer stable release and prints a notice under the banner. The check runs a maximum of one time in 24 hours. kdeps keeps the result in `~/.kdeps/update-check.json`. The check stops after 3 seconds:

```
Update available: v3.0.0 -> v3.1.0. Run /upgrade to update.
```

Development builds skip the notice. `make build` stamps `v2.0.0-dev`, which is older than every release. `/upgrade` still checks when you run it.

`/upgrade` (or `kdeps --upgrade` outside the REPL) checks now and installs. The result depends on the install method:

- **Homebrew** (`brew install kdeps/tap/kdeps`): prints `brew upgrade kdeps`. If kdeps replaces the binary, Homebrew does not know the correct version.
- **.deb / .apk package**: prints the package-manager upgrade command.
- **Standalone** (`curl | sh` installer or a downloaded binary): asks `[Y/n]` (`KDEPS_YES=1` skips it), downloads the release archive, verifies its SHA256 against `checksums.txt`, and atomically replaces the binary. Restart kdeps afterward.

## Nightly builds

kdeps builds a nightly from `main` most days. `/upgrade nightly` (or `kdeps --upgrade --nightly`) installs the latest nightly for that one check.

Nightly works only for standalone installs. Homebrew, .deb and .apk track stable only, so `/upgrade nightly` prints standalone-install instructions there. A nightly keeps the current stable version number until the next stable release. "Already up to date" on the nightly channel means you run that exact nightly tag.

## Installing a specific version (including a downgrade)

`/upgrade <version>` (or `kdeps --upgrade --target-version <version>`) installs exactly that version. It skips the update check, so it upgrades, downgrades or reinstalls:

```
/upgrade 2.35.0
Downgrade to v2.35.0 now? [Y/n]
```

This is standalone-only, like nightly. Homebrew, .deb and .apk print a link to the release page.

## Updating assets

kdeps behavior files are versioned assets. They update separately from the binary. Harness sections, events, actions, presets, themes, tool definitions and slash commands drive agent mode. LLM server recipes, `kdeps new` templates, workflow schemas, executor defaults, components and Docker/Kubernetes templates drive workflow mode.

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
| `models` | A model registry: llamafile and GGUF aliases (harvested from HuggingFace), the cloud catalog `models/cloud`, or the OpenRouter free models `models/openrouter` (harvested daily) | `models/gguf`, `models/openrouter` |
| `templates` | A `kdeps new` project template (all its files) | `templates/api-service` |
| `schemas` | The JSON schemas `kdeps validate`, `run` and `package` check workflows, resources, components and agencies against | `schemas/v1` |
| `defaults` | Executor defaults (timeouts, SQL pool sizes, search limits) used when neither the resource nor `config.yaml` sets a value | `defaults/executors` |
| `docker` | The Dockerfile, entrypoint and supervisord templates `kdeps build` renders | `docker/image` |
| `k8s` | The Deployment, Service and NetworkPolicy templates `kdeps export k8s` renders | `k8s/manifests` |
| `components` | A YAML kdeps component a workflow calls with `component:`, e.g. [ocr_image](/workflow/resources/ocr) | `components/ocr_image` |
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

The binary contains none of these files. kdeps loads them only from `~/.kdeps/assets/`. The first run asks for a profile and installs it:

```text
$ kdeps
What will you use kdeps for? (change it later with kdeps profile)
  1) minimum                kdeps run and validate: chat, httpClient, python, exec, apiResponse
  2) workflows  ~3.0k tok    minimum, kdeps new templates and the components (sql, ...)
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
| `agent` | base + `harness`, `commands`, `themes`, `events`, `actions`, `presets`, `memory`, the loop's required tools and the file, shell, web and memory tools |
| `deploy` | base + `templates`, `components`, `docker`, `k8s`, `recipes` |
| `full` | every set |
| `custom` | a checklist of every set and item |

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
Toggle items (e.g. 3 5-9), a = all, n = none, Enter to go back:
```

In `custom`, a set number selects or clears the full set, and `o<N>` opens it. In a set, numbers and ranges select or clear items. `a` selects all, and `n` selects none. The loader of the set needs each item with the mark `*`. This item stays selected when a different item in the set is selected. A base set keeps its necessary items (or one item) as a minimum.

kdeps records a fully selected set by name, so `kdeps update` installs items that are published to it later. kdeps records a partly selected set item by item.

`kdeps info <set>/<name>` prints what an asset is for and where kdeps runs it. Its kdeps.io page shows the same flow:

```text
$ kdeps info tools/read_file
# tools/read_file
Read a file from the local filesystem. ...
- Set: tools, used in workflow and agent mode
- Loaded by: kdeps (agent loop), kdeps run, /reload, /settings
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

The flow is one `//kdeps:flow` line on the function that uses the set. The bracketed step is the one the set drives:

```go
//kdeps:flow tools: the agent loop builds the request > [the tool definitions are sent to the model] > the model calls a tool > kdeps runs it > the result goes back to the model
func definedAs(id string, t *kdepstools.Tool) *kdepstools.Tool {
```

`tools/assetflow` derives the rest from the call graph: "Loaded by", the mode, and "In the code". It fails if a set has no flow line or two flow lines. It also fails if the flow line is on a function that does not get to a use of the set. The error gives the function where you must move the line. `make generate` writes `pkg/assets/setflow_gen.go` again. CI fails until this file is current.

The asset publish job makes the file again, so kdeps.io matches main. `pkg/assets/setinfo.go` gives the purpose of each set. The `description:` of an item gives its purpose.

`~tok` estimates what an item adds to the model's context (about 4 bytes per token). Harness sections are system-prompt text; tool definitions and components are tool schemas sent every turn. Themes, events, schemas, templates and other sets never reach the model and show no count. The estimate appears in the asset index, on kdeps.io, in `kdeps install` and `kdeps update`, and as a total in `kdeps list --assets`:

```text
$ kdeps install tools/bash_exec
  tools/bash_exec                      install   - -> 1.0.0  ~159 tok
LLM context: these added about 159 tokens (prompts, tool definitions) when loaded.
```

### Profiles without a terminal

CI, Docker and scripts get no question. `KDEPS_PROFILE` names the profile (`KDEPS_PROFILE=agent`) or a list of sets and items (`KDEPS_PROFILE=docker,k8s,tools/bash_exec`). Without it, kdeps installs only the sets the command loads. kdeps records the profile in `~/.kdeps/assets/lock.json`.

### What a profile installs later

A command that loads a set outside the profile installs only that set's required items, never the whole set. Under `minimum`, `kdeps` (the agent loop) adds the `normal` theme, the loop's required tools and harness, and 23 core commands, which `/help` lists: `help`, `exit`, `clear`, `model`, `compact`, `session`, `settings`, `update`, `fold`, `memory`, `skills`, `prompts`, `reload`, `judges`, `harness`, `efficiency`, `recall`, `knowledge`, `prompt`, `search`, `context`, `konfig`, `kartographer`.

```text
$ KDEPS_PROFILE=minimum kdeps
Installing the minimum profile...
minimum profile  downloaded 7/19 providers/openai
Installing harness/compaction-system, ..., tools/note_get, ..., commands/help, ..., themes/normal (needed by this command)...
```

When files download, a terminal writes one line again and again, as in `minimum profile  downloaded 7/19 providers/openai`. The profile name stays on that line. A log prints one line for each file. Items that the registry does not have now show as one line, `Skipped 3 items that are no longer published.`

kdeps gives the name of an item that fails for a different reason. If other items installed, kdeps continues and shows the profile again (`Profile: full (every asset set)`). If no items installed, kdeps stops. It tells you to run `kdeps profile` when you are online.

- Sets with no necessary items (`events`, `actions`, `presets`) stay empty. kdeps records them as empty in `lock.json`, so the next start makes no network call.
- If a necessary item is not installed when code reads it, kdeps installs it one time, on first use. The terminal shows its name: `Installed knowledge/vault 1.0.0 (first use).` (the first `kdeps knowledge build` or local search).
- A Go built-in command (`builtin: true`) runs without its file; the file only adds it to `/help`.
- A YAML command (`run:`, `setting:` or `subcommands:`, such as `/refine` or `/review`) exists only when installed: `kdeps install commands/refine` adds `/refine`, `kdeps install commands` adds all.
- You cannot uninstall a necessary item.

A switch to a smaller profile prunes items. `kdeps profile minimum` after `agent` or `full` deletes all items that are not in the new profile (`commands/efficiency  remove  not in the profile`). A set that is not in the profile also loses its necessary items. The base sets and the sets of the agent loop are exceptions. If you install an item by name later, `kdeps update` keeps it.

With no profile chosen (no terminal, no `KDEPS_PROFILE`), a command installs each set it loads in full on first need (`kdeps export` installs `docker` and `k8s`). Either way, a workflow that calls a missing component downloads it on that call:

```text
$ kdeps run workflow.yaml
Installed components/sql 1.0.2 (first use).
```

Offline, a missing set stops the command and names the fix: `kdeps assets docker are not installed; connect to the internet and run kdeps update docker`. `kdeps update` and `kdeps registry` run without assets. After installation, kdeps runs offline. `kdeps build` images carry the build host's assets in `/app/.kdeps/assets`, so containers download nothing.

### Update notices and commands

A startup notice shows when there are new published updates. kdeps checks each day and keeps the result in `~/.kdeps/assets/check.json`. The desktop app shows an **Updates available** button in the top bar.

```text
2 asset updates available (harness/safety 1.2.0, tools/read_file 1.1.0). Run kdeps update.
```

```bash
kdeps update                                # profile + everything installed since, to newest (skips pinned and removed items)
kdeps profile agent                         # install a profile (or sets and items: docker,tools/bash_exec) and keep it current
kdeps profile                               # choose the profile again, or tick sets and items (custom)
kdeps update tools                          # every item in one set
kdeps update harness/safety                 # newest version of one item; unpins and restores it
kdeps update harness/safety@1.2.0           # exactly that version, pinned until you update it by name
kdeps update --check                        # what would change
kdeps list --assets                         # every item: version, state, shelf, ~tokens; total at the end
kdeps list rag                              # one shelf (see Shelves below)
kdeps info tools/read_file                  # an item's purpose, where it runs, installed version, ~tokens
kdeps info events                           # a set: purpose, where it runs, items
kdeps uninstall themes/vim,presets/frugal   # remove items
```

In the REPL, `/update`, `/update check`, `/update list` and `/update remove <item>...` do the same and apply immediately. Changed tool descriptions reach the model from the next start.

### How a version is chosen

- Each file starts with `version:` (semver). It can declare `kdeps: ">=X.Y.Z"`, the oldest kdeps that loads it. kdeps skips versions that need a newer kdeps (run `kdeps --upgrade` first).
- `requires:` lists other items and the versions accepted. Installing the file first installs each missing, removed or too-old requirement at its newest version in range, recursively:

  ```yaml
  # components/web_search/template.yaml (a bundle's header; a single-file item has these at its top)
  version: 1.0.1
  requires:
    schemas/v1: ">=1.0.3"   # its component.yaml uses connection:, added in schemas/v1 1.0.3
  ```

  ```text
  $ kdeps update components/web_search
    schemas/v1                           install   - -> 1.0.3  (required by components/web_search)
    components/web_search                install   - -> 1.0.1
  ```

  Install fails if a requirement is not published for your kdeps. It also fails if a requirement is pinned outside the range. To remove the pin, update the pinned item by name. Events require their action, presets their events and actions, `/frugal` and `/thorough` their preset, components their schema version. Publishing rejects a requirement no published version satisfies.
- Before a download replaces a file, kdeps checks it against the published sha256 and the parser of the set. If you edit a file in `~/.kdeps/assets/` manually, its checksum fails and kdeps ignores it. `kdeps update <item>` restores it.
- kdeps skips a file that it cannot read (corrupt, or written for a newer kdeps) with a warning. As a result, `kdeps update` can always repair it.

### Removing and overriding

A removed item stays hidden until you update it by name. The model does not see a removed tool. kdeps does not send a removed harness section. kdeps does not list a removed theme or template. You cannot remove these items:

- `themes/normal`
- the internal harness prompts: compaction, judges, prompt refinement, session handshake
- the loop's control tools: `judge_verdict`, `session_handshake`, `session_ops`
- the model registries: `models/llamafile`, `models/gguf`, `models/cloud`, `models/openrouter`

`/harness disable <name>` turns a harness section off without removing it.

Files in `~/.kdeps/harness/`, `~/.kdeps/themes/`, `~/.kdeps/llm-servers/` and similar directories layer on top of the installed version.

`KDEPS_ASSETS_URL` points downloads at a mirror laid out like `assets/` in kdeps/packages (`index.json` plus `<set>/<name>/<version>.yaml`). `KDEPS_ASSETS_URL=off` turns downloads off.

`KDEPS_ASSETS_SRC` loads assets from a directory laid out like `assets/` in the kdeps repo (`<set>/<name>.yaml`, one directory per template). kdeps uses an edited file when its `version:` is higher than the installed version:

```bash
KDEPS_ASSETS_SRC=~/src/kdeps/assets ./kdeps   # run on a checkout's assets
```

## Shelves

A [shelf](/reference/glossary#shelf) is the engineering category of an asset. The set stays the kind of file, and the path stays the id (`harness/safety`, `tools/bash_exec`): `kdeps update` installs that id, and the formula is `kdeps-<set>-<name>`. `kdeps list` prints each item's shelf, `kdeps list <shelf>` shows one shelf, and kdeps.io filters by the same names.

| Shelf | What is filed here |
|---|---|
| model | `models/`, `providers/`, `recipes/`, and `/model` |
| context | `presets/`. History-rewrite prompts, window-trim events, actions `compact`, `truncate`, `drop_oldest`, `cap` |
| harness | Preamble sections, nudges, praise, the skills preamble, the tools reminder |
| tools | Tool definitions other than memory, rag, guardrails, or agent. Web search and scrapers stay here |
| memory | `memory_*`, `knowledge_*`, the memory harness section, memory events |
| rag | `embedding_*`, `search_local`, `retrieve_context`, `load_document`, the rerank tools |
| agent | Slash commands other than `/model`. Goals, tasks, teams, `load_skill`, `instruct`, `identity_get`, `themes/` |
| guardrails | Approval tools, the handshake, `session_handshake`, the safety harness. Events and actions that block, reject, or fail |
| evals | Judge prompts, `judge_verdict`, `judge-iterations`, `judge-max-rounds`. The items keep the name judge. The panel is the eval |
| workflow | `schemas/`, `templates/`, `components/`, `defaults/` |
| deploy | `docker/`, `k8s/` |

`events/` and `actions/` stay their own sets; each file takes the shelf of the behavior it serves.

Tokens, structured output, embeddings, session, trace, and fine-tune are not shelves.

Tokens are counts. Structured output is `harness.output.json`. Embeddings are the rag items. Session resume and `kdeps --instrument` are code. Fine-tune is not in kdeps. Install profiles (`minimum`, `workflows`, `agent`, `deploy`) are a separate axis.
