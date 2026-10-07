# Desktop app

This is the reference for installing, building and configuring the app. For the product overview, see [kdeps desktop](/desktop/).

The window uses the same model config as the [coding CLI](/agent/). A local model stays on this machine. To use a server you run, set `llm.base_url` in Settings or in `~/.kdeps/config.yaml`. The steps are [Desktop app to a private LLM server](/start/data-sovereignty#desktop-app-to-a-private-llm-server). Build that server with [kdeps LLM server](/llm-server/).

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: openai
  base_url: http://192.168.1.50:8000/v1   # your LLM server, your network
  models:
    - llama3.2                            # one model, or several
```

The desktop app is the agent loop in a window: a chat UI with history, search, drag-and-drop files and a settings modal. It is the same `pkg/agent` loop the terminal REPL runs, so tools, memory, sessions, goals and judges behave identically - only the front end differs. One codebase builds for macOS, Linux and Windows (a [Wails](https://wails.io) shell around the system WebView, no Electron).

```d2
direction: right
window: "Desktop window\n(HTML/JS)"
app: "Wails shell\n(desktop/)"
svc: "pkg/desktop.Service"
loop: "pkg/agent Loop"
tools: "Built-in tools\nmemory, sessions"
window -> app: "bound methods"
app -> window: "kdeps:event stream"
app -> svc
svc -> loop
loop -> tools
```

It is an agent-mode front end. Workflows, components and agencies still show up only as tools the model may call; the window never runs a workflow by itself.

## Install

The app is standalone: the `kdeps` CLI does not need to be installed. You need a model: one this app downloads on first use, an LLM server you run, or a cloud API key.

```bash
brew install --cask kdeps/tap/kdeps-desktop   # macOS, picks Apple Silicon or Intel; clears the quarantine flag
```

After `brew tap kdeps/tap` the short form `brew install --cask kdeps-desktop` works too. Update with `brew upgrade --cask kdeps-desktop`. The cask and the Scoop manifest are rewritten by the `Release Desktop` workflow on every release.

## Linux (packages)

Releases after v2.53.2 attach a `.deb`, `.rpm` and Arch package. They install the app, launcher entry and icon, and pull in the GTK and WebKit libraries as dependencies:

```bash
sudo apt install ./kdeps-desktop_<version>_linux_amd64.deb        # Debian, Ubuntu
sudo dnf install ./kdeps-desktop_<version>_linux_amd64.rpm        # Fedora, RHEL
sudo pacman -U kdeps-desktop_<version>_linux_amd64.pkg.tar.zst    # Arch
```

## Linux (tarball)

```bash
# Debian/Ubuntu runtime libraries (Fedora: sudo dnf install gtk3 webkit2gtk4.1)
sudo apt install -y libgtk-3-0 libwebkit2gtk-4.1-0

VERSION=$(curl -fsSLI -o /dev/null -w "%{url_effective}" https://github.com/kdeps/kdeps/releases/latest | sed "s#.*/v##")   # latest release, e.g. 2.55.0
curl -fLO "https://github.com/kdeps/kdeps/releases/download/v${VERSION}/kdeps-desktop_${VERSION}_linux_amd64.tar.gz"

tmp="$(mktemp -d)" && tar -xzf "kdeps-desktop_${VERSION}_linux_amd64.tar.gz" -C "$tmp"
mkdir -p ~/.local/bin ~/.local/share/applications ~/.local/share/icons
install -m 755 "$tmp/kdeps-desktop" ~/.local/bin/                 # binary (~/.local/bin must be on PATH)
cp "$tmp/kdeps-desktop.desktop" ~/.local/share/applications/      # launcher entry
cp "$tmp/kdeps.png" ~/.local/share/icons/                         # launcher icon

kdeps-desktop   # or find "kdeps" in your app launcher
```

## Windows (Scoop)

```powershell
scoop bucket add kdeps https://github.com/kdeps/scoop-bucket
scoop install kdeps-desktop   # adds a Start menu shortcut; scoop update kdeps-desktop upgrades it
```

## Windows (PowerShell)

```powershell
$v = (Invoke-RestMethod https://api.github.com/repos/kdeps/kdeps/releases/latest).tag_name.TrimStart("v")   # latest release
$dir = "$env:LOCALAPPDATA\kdeps-desktop"
Invoke-WebRequest "https://github.com/kdeps/kdeps/releases/download/v$v/kdeps-desktop_${v}_windows_amd64.zip" -OutFile kdeps-desktop.zip
Expand-Archive kdeps-desktop.zip -DestinationPath $dir -Force

# Start menu shortcut
$s = (New-Object -ComObject WScript.Shell).CreateShortcut("$env:APPDATA\Microsoft\Windows\Start Menu\Programs\kdeps.lnk")
$s.TargetPath = "$dir\kdeps-desktop.exe"; $s.Save()

& "$dir\kdeps-desktop.exe"
```

The WebView2 runtime ships with Windows 11; on Windows 10 install it from Microsoft first. The exe is not code-signed, so SmartScreen may warn on first launch: More info, then Run anyway.

## Download

Every tagged release attaches the desktop app next to the CLI archives, built on native runners by the `Release Desktop` workflow (it runs when the release is published; `gh workflow run release-desktop.yml -f tag=vX.Y.Z` re-attaches to an existing tag):

| OS | File | Notes |
|----|------|-------|
| macOS (Apple Silicon) | `kdeps-desktop_<version>_darwin_arm64.dmg` | Drag `kdeps.app` to Applications. Ad-hoc signed, so the first launch needs right-click, Open. |
| macOS (Intel, x86_64) | `kdeps-desktop_<version>_darwin_amd64.dmg` | Same as above, for Intel Macs. |
| Linux | `kdeps-desktop_<version>_linux_amd64.tar.gz` | Needs `libgtk-3` and `libwebkit2gtk-4.1` installed. Includes `kdeps-desktop.desktop` and `kdeps.png`: copy them to `~/.local/share/applications/` and `~/.local/share/icons/` for a launcher icon. |
| Windows | `kdeps-desktop_<version>_windows_amd64.zip` | Needs the WebView2 runtime (ships with Windows 11). |

Each file has a `.sha256` beside it.

## Build from source

From the repo root, the Makefile picks the right tags and linker flags for your OS:

```bash
make desktop-build     # dist/desktop/kdeps-desktop
make desktop-package   # macOS: kdeps.app + .dmg | Linux: .tar.gz | Windows: .zip (all in dist/desktop/)
```

Set `APPLE_SIGN_IDENTITY="Developer ID Application: ..."` before `make desktop-package` on macOS to sign with a real identity instead of ad-hoc.

## Build by hand

The desktop app is a separate Go module under `desktop/` so the CLI keeps building without a C toolchain or a WebView.

```bash
cd desktop

# macOS (needs the UniformTypeIdentifiers framework for file drops)
CGO_LDFLAGS="-framework UniformTypeIdentifiers" \
  go build -tags desktop,production -o kdeps-desktop .

# Linux (Debian/Ubuntu: apt install libgtk-3-dev libwebkit2gtk-4.1-dev)
go build -tags desktop,production,webkit2_41 -o kdeps-desktop .

# Windows (PowerShell; WebView2 runtime ships with Windows 11)
go build -tags desktop,production -o kdeps-desktop.exe .
```

Run the binary. It opens on the last workspace you used, or your home folder on first launch. The `desktop` GitHub Actions workflow builds and tests all three platforms.

## Workspaces

A workspace is a folder. Chat history and persistent memory are stored per folder, and the agent's file tools are confined to it - the same boundary `KDEPS_WORKSPACE_ROOT` sets for the terminal agent. Use the workspace button at the bottom of the sidebar to open another folder or pick one of the ten most recent. Switching starts a fresh chat and re-scopes history and memory; it is refused while a turn is running.

Window state lives in `kdeps-desktop/state.json` under your OS config directory (`~/Library/Application Support`, `~/.config`, `%AppData%`).

## Chat

| Feature | Behavior |
|---------|----------|
| History | Every session in the workspace is listed in the sidebar, newest first. Click to reload; hover a row and click Delete twice to remove it. |
| All folders | The **This folder / All folders** switch under the search box. All folders lists the chats of every workspace (terminal `kdeps` chats included), grouped by folder, newest first. Opening a chat from another folder switches the workspace to that folder. Search covers chat names, messages and folder names. The choice is remembered. |
| Search | The sidebar search box matches the chat name, first prompt and every message (case-insensitive) and shows the matching snippet. |
| Streaming | Tokens render as they arrive. Narration lines ("Reading config.yaml...") and tool cards show each call, its arguments and result. |
| Approvals | In the default `ask` permission mode a modal shows each tool call or out-of-workspace path. Choose allow once, allow always, or deny. |
| Files | Drag files or folders onto the window, or use the attach button. Dropping onto an empty composer sends them at once and the agent analyzes them: what each file is, what it contains, and what you can do with it. Type a message first (or attach with the button) to give your own instruction instead. Images, audio, video and PDFs go to the model as multimodal parts; UTF-8 text up to 64 KiB is inlined; everything else (DOCX, CSV, large logs, binaries, folders) is listed by path and size, and the model opens it with the right tool (`read_file`, `load_document`, `list_files`). |
| Stop | The send button becomes stop while a turn runs; cancelling is not an error. |

```d2
direction: right
drop: "drop files" {shape: oval}
empty: "composer empty?" {shape: diamond}
chip: "attach chip\n(wait for your message)"
send: "send turn"
split: "sort each file" {shape: diamond}
loop: "agent loop" {shape: oval}
drop -> empty
empty -> send: "yes: analyze them"
empty -> chip: "no"
chip -> send: "Send"
send -> split
split -> loop: "media: multimodal part"
split -> loop: "small text: inlined"
split -> loop: "other: path + size,\nmodel reads with a tool"
```

## Models, slash commands and autocomplete

The window is a client of the same REPL the terminal runs, so it behaves the same: every slash command goes through the CLI's own dispatcher and its output streams into the chat live (terminal colors stripped).

```text
composer ──/cmd──> REPL dispatcher ──stdout──> live output block
   │                    │
   └─plain text─> agent loop ─tokens─> assistant message
```

| Feature | Behavior |
|---------|----------|
| Model picker | The model name at the top of the chat opens the picker (so does a bare `/model`). It lists everything `/model list` shows: the harvested llamafile and GGUF registries, Ollama models and the cloud catalog, filterable by type and searchable. Badges show current, ready, not downloaded and no API key. |
| Switching | Picking a ready model runs `/model <name>`. A model that is not downloaded asks first, then downloads it with progress in the chat. A cloud model without a key points you to Settings. The choice is saved as the default for the next launch. |
| Start model | A saved default wins, then the same auto-pick the terminal uses (installed model, available key, or catalog). |
| Slash commands | Type `/` for the command list with one-line descriptions. Skills and prompts appear too. Terminal-only commands (`/exit`, `/editor`, `/login`, `/upgrade`) reply with the window equivalent; `/settings` and `/theme` open Settings; `/clear` clears the view. |
| Autocomplete | Tab, Up and Down complete slash commands, subcommands, model names and `@file` paths, with the same candidates as terminal tab completion. |
| Failed turns | An unreachable backend (for example no network to the auto-picked cloud model) shows the model name and a Switch model button. The unsaved chat stays visible in the sidebar until a turn succeeds. |

Chat runs on the agent loop. The Projects window below also builds and runs workflow-mode projects.

## Settings modal

Open it with the Settings button at the bottom of the sidebar. Seven tabs:

| Tab | What it edits |
|-----|---------------|
| Appearance | Theme (light, dark, system), accent color and density. Stored in the WebView's local storage, not in `config.yaml`. |
| All settings | Every scalar in `~/.kdeps/config.yaml` (`llm.backend`, `llm.ctx_size`, `resource_defaults.chat.temperature`, ...), generated from the config schema so new fields appear automatically. Secret fields (keys, tokens) show only whether one is set and are write-only. Fields with a fixed set of values render as dropdowns (`llm.backend`, `llm.strategy`, `resource_defaults.onError.action`), numeric ranges as sliders with a reset link (`temperature`, `top_p`, penalties), and free-form fields (timeouts, context sizes, Python version, timezone, hosts) offer a suggestion list while still accepting any value. A one-line help text sits under each. |
| Config file | The whole `~/.kdeps/config.yaml` as text (created from the template if missing), for what the generated fields do not cover: `sql_connections`, `http_connections` and the other connection lists, `agents` profiles, the LLM `router`. Save writes it as typed, comments included. YAML that does not parse is refused with the parser error; unknown keys and bad values are saved and listed as warnings. Values apply to the running app at once, except variables you exported before starting it, which still win. |
| Prompts and harness | System-prompt harness sections (toggle each, and whether it is repeated as a per-turn reminder) plus the preset list, same as `/harness` in the REPL. |
| Custom instructions | Free text saved to `KDEPS.md` in the workspace, which the agent loads into every session's system prompt for that folder. |
| Memories | The workspace's persistent memory: list, add or overwrite (same key) and delete facts. |
| Work profile | Export or import tuning, harness settings and skills as one konfig file, to carry your setup between machines. |

Saving a setting writes `config.yaml` (preserving comments and other keys) and applies the matching environment variable to the running process. Changing instructions or harness sections rebuilds the system prompt before the next turn.

## Projects: workflows, agencies and components

The Projects window lists every workflow, agency and component in the workspace, plus the packages `kdeps registry install` put in `~/.kdeps/agents` and `~/.kdeps/components`. From there you build a project with forms, run it, or hand it to the chat agent as a tool. It is `kdeps new`, `kdeps run` and a YAML editor in one window. Open it with the Projects button at the bottom of the sidebar.

This section covers both modes: you build and run workflow-mode projects, and "Use in chat" turns one into a tool for agent mode.

```d2
direction: right
list: "Projects list\n(workspace + installed)" {shape: oval}
build: "Build tab\n(forms)"
files: "workflow.yaml\nresources/*.yaml"
env: ".env + app variables"
run: "Run tab\n(kdeps run <dir>)"
chat: "Overview: Use in chat\n(agent tool)"
engine: "kdeps engine\n(DAG, all executors)" {shape: oval}
list -> build: "edit"
build -> files: "writes YAML"
files -> run
env -> run
files -> chat
run -> engine: "server: each request\nelse: once"
chat -> engine: "when the agent calls it"
```

| Tab | What it does |
|-----|--------------|
| Overview | Name, version, path and parse errors. **Use in chat** registers the project as a tool the chat agent may call; it never runs unless the agent calls it. **Validate**, **Open folder** and **Delete** (workspace projects only, asks twice). |
| Run | One **Run** button. For a workflow or agency it runs `kdeps run <absolute folder path>` from the workspace with the project's environment (below): an API or web server keeps running and the button becomes **Stop**, with the URL, a copyable `curl` per route and the live output, plus an **Open** button that opens the web interface in your browser when the project has a `webServer`; any other workflow (single run, file input, bot) behaves as on the command line. For a component, Run takes one field per declared input, like a `with:` block, and shows each resource's progress and the response. |
| Build | The manifest form (name, description, version, target, and the `settings` or `interface` YAML) and one card per resource. A resource card has `actionId`, `name`, `description`, `requires` (tick the resources it depends on), the fields of its action type, and an **advanced** box for everything else (`validations`, `onError`, `loop`, `before`, `after`, `items`). |

**New** creates a project folder in the workspace from a template (`api-service`, `sql-agent`, `agency`), the same templates as `kdeps new`, and opens it in Build.

### How the builder writes YAML

Every resource type the engine has (chat, httpClient, sql, python, exec, apiResponse, browser, email, scraper, ...) is offered, with its fields read from the kdeps types, so new executors appear without an app update. A card shows the fields that have a value; **add field** lists the rest with their descriptions. Lists and objects (`tools`, `headers`, `response`) are edited as YAML text.

Saving a card with type `chat` and three fields writes this file:

```yaml
# resources/ask.yaml - written by the Build tab
actionId: ask                       # what other resources list in requires and read with output('ask')
name: Ask the model
requires:
  - fetch                           # ticked in the requires checkboxes
chat:
  model: llama3.2:1b
  prompt: Say hi to {{ get('who') }}  # multi-line fields are text areas
  temperature: 0.3                  # numbers are checked: "warm" is refused as "chat.temperature: must be a number"
```

| Rule | Behavior |
|------|----------|
| Where new resources go | A workflow gets `resources/<actionId>.yaml`. A manifest that already lists `resources:` inline (every component does) gets the new item appended there. |
| Editing existing files | Only the parts you changed are rewritten. Untouched keys keep their comments and order. |
| Checks before writing | `actionId` must be unique and use letters, digits, `-` or `_`; the resource must pass the resource schema. After a save the whole project is validated and any problem (for example a `requires` that names a missing resource) is shown with the save. |
| Read-only | Installed packages, and Jinja2 files (`*.j2`), are shown but not edited. Use Open folder for those. |

### Environment variables

The Run tab sets the environment of a workflow or agency run. Running `house-finder` with these settings is the same as typing `HOUSE_FINDER_ROOT=$PWD/house-finder kdeps run $PWD/house-finder` in the workspace folder; the app always passes the project folder as an absolute path:

```bash
# house-finder/.env - loaded automatically on every Run
HOUSE_FINDER_ROOT=$PWD/house-finder   # $PWD is the workspace folder, where kdeps run starts
CITY=Amsterdam
```

```bash
# Run tab > variables - saved in the app, override .env on the same name
CITY=Utrecht
```

| Source | Behavior |
|--------|----------|
| `.env` in the project folder | Read on every Run; the tab lists the names it found. A line that is not `NAME=value` stops the run with that line. |
| **variables** box | `NAME=value` per line, `#` comments, optional `export ` prefix and quotes. Saved per project in `~/.kdeps/desktop-project-env.json`. Wins over `.env`. |
| **Add** | Appends one `NAME=value` from the name and value fields. |
| **Import from file** | Appends the `NAME=value` lines of any text file you pick; it does not need to be named `.env`. |

`$NAME` references expand, with earlier lines visible to later ones. The output log prints the command with names only (`$ HOUSE_FINDER_ROOT=... kdeps run /Users/you/Projects/house-finder`), never the values.

### Servers need an API token

Every kdeps API server requires a token, from the app as from the CLI. Set it once:

```yaml
# ~/.kdeps/config.yaml
api_auth_token: "change-me"   # or export KDEPS_API_AUTH_TOKEN; requests send it as a Bearer token or X-API-Key
```

```bash
curl -H "Authorization: Bearer $KDEPS_API_AUTH_TOKEN" http://127.0.0.1:16395/api/v1/hello
```

If the configured port is taken, the next free one is used and shown. A running project stops when you click Stop, delete the project, or quit the app. The app runs projects with its own binary, so the `kdeps` CLI does not need to be installed.

## Offline

The front end is plain HTML, CSS and JavaScript embedded in the binary. It loads nothing from a CDN, so the app works air-gapped with a local model. Pointed at an LLM server you run, prompts go to that server and stop there. The rule is the same as the CLI's: [Data sovereignty](/start/data-sovereignty).
