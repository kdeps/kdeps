# Desktop app

*Applies to agent mode.*

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

The app is standalone: the `kdeps` CLI does not need to be installed. You only need a model (a cloud API key, or a local model it downloads on first use).

```bash
brew install --cask kdeps/tap/kdeps-desktop   # macOS, picks Apple Silicon or Intel; clears the quarantine flag
```

After `brew tap kdeps/tap` the short form `brew install --cask kdeps-desktop` works too. Update with `brew upgrade --cask kdeps-desktop`. The cask is rewritten in `kdeps/homebrew-tap` by the `Release Desktop` workflow on every release.

### Linux

```bash
# Debian/Ubuntu runtime libraries (Fedora: sudo dnf install gtk3 webkit2gtk4.1)
sudo apt install -y libgtk-3-0 libwebkit2gtk-4.1-0

VERSION=2.53.2   # latest: https://github.com/kdeps/kdeps/releases/latest
curl -fLO "https://github.com/kdeps/kdeps/releases/download/v${VERSION}/kdeps-desktop_${VERSION}_linux_amd64.tar.gz"

tmp="$(mktemp -d)" && tar -xzf "kdeps-desktop_${VERSION}_linux_amd64.tar.gz" -C "$tmp"
mkdir -p ~/.local/bin ~/.local/share/applications ~/.local/share/icons
install -m 755 "$tmp/kdeps-desktop" ~/.local/bin/                 # binary (~/.local/bin must be on PATH)
cp "$tmp/kdeps-desktop.desktop" ~/.local/share/applications/      # launcher entry
cp "$tmp/kdeps.png" ~/.local/share/icons/                         # launcher icon

kdeps-desktop   # or find "kdeps" in your app launcher
```

### Windows (PowerShell)

```powershell
$v = "2.53.2"   # latest: https://github.com/kdeps/kdeps/releases/latest
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
| Search | The sidebar search box matches the chat name, first prompt and every message (case-insensitive) and shows the matching snippet. |
| Streaming | Tokens render as they arrive. Narration lines ("Reading config.yaml...") and tool cards show each call, its arguments and result. |
| Approvals | In the default `ask` permission mode a modal shows each tool call or out-of-workspace path. Choose allow once, allow always, or deny. |
| Files | Drag files onto the window or use the attach button. Text files up to 1 MiB are inlined into the prompt; other files are passed as paths. |
| Stop | The send button becomes stop while a turn runs; cancelling is not an error. |

```text
drop file -> attach chip -> Send -> prompt + file text -> agent loop
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

Both modes run on the agent loop: this page is **agent mode** only.

## Settings modal

Open it with the Settings button at the bottom of the sidebar. Six tabs:

| Tab | What it edits |
|-----|---------------|
| Appearance | Theme (light, dark, system), accent color and density. Stored in the WebView's local storage, not in `config.yaml`. |
| All settings | Every scalar in `~/.kdeps/config.yaml` (`llm.backend`, `llm.ctx_size`, `resource_defaults.chat.temperature`, ...), generated from the config schema so new fields appear automatically. Secret fields (keys, tokens) show only whether one is set and are write-only. Fields with a fixed set of values render as dropdowns (`llm.backend`, `llm.strategy`, `resource_defaults.onError.action`), numeric ranges as sliders with a reset link (`temperature`, `top_p`, penalties), and free-form fields (timeouts, context sizes, Python version, timezone, hosts) offer a suggestion list while still accepting any value. A one-line help text sits under each. |
| Prompts and harness | System-prompt harness sections (toggle each, and whether it is repeated as a per-turn reminder) plus the preset list, same as `/harness` in the REPL. |
| Custom instructions | Free text saved to `KDEPS.md` in the workspace, which the agent loads into every session's system prompt for that folder. |
| Memories | The workspace's persistent memory: list, add or overwrite (same key) and delete facts. |
| Work profile | Export or import tuning, harness settings and skills as one konfig file, to carry your setup between machines. |

Saving a setting writes `config.yaml` (preserving comments and other keys) and applies the matching environment variable to the running process. Changing instructions or harness sections rebuilds the system prompt before the next turn.

## Offline

The front end is plain HTML, CSS and JavaScript embedded in the binary. It loads nothing from a CDN, so the app works air-gapped with a local model - the data sovereignty story is the same as the CLI's.
