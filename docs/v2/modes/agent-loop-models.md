# Local model management

Managing which LLM the [agent loop REPL](/modes/agent-loop-mode) talks to - switching mid-session, auto-routing, and running local model servers.

## Switching models

`/model <name>` switches models mid-session. For local backends (`file`, `gguf`), the REPL downloads and starts the server if it isn't already running, then shows a progress display until the completions endpoint is accepting requests - the first prompt after the switch never gets a "network error" while weights load.

```
/model qwen3.5-4b                     # switch to a known alias
/model default qwen3.5-4b             # save as default startup model
/model default                        # show the current default
```

The default model is persisted to `~/.kdeps/agent-loop-settings.yaml` and loaded automatically at startup when `--model` is not passed.

## Same alias, different backends

llamafile, GGUF, and Ollama registries can each have their own, unrelated
entry under the same bare alias (e.g. `qwen3.5` might exist as a distinct
download in both the llamafile and GGUF catalogs, or as a locally-pulled
Ollama tag). When kdeps detects this, `/model list` and `/model <tab>` show
every colliding entry qualified by its backend instead of hiding one:

```
llamafile:qwen3.5
gguf:qwen3.5
```

Switch to a specific one with its qualified name:

```
/model gguf:qwen3.5
```

Typing the bare, still-ambiguous name auto-picks the same backend kdeps has
always preferred (`ollama` > `gguf` > `llamafile`) and prints a one-line
notice so you know which one you got:

```
/model qwen3.5
"qwen3.5" is ambiguous across backends (llamafile:qwen3.5, gguf:qwen3.5) -- using gguf:qwen3.5. Use the full name to pick a specific one.
```

Non-colliding names - the vast majority of aliases - are completely
unaffected; you only ever need the `backend:` prefix when kdeps tells you to.

## `--model auto`: route across your configured models

`--model auto` (or `KDEPS_AGENT_MODEL=auto`) picks the best hardware-fit model from your own `llm.models` config (`~/.kdeps/config.yaml`) via [`llmfit`](https://github.com/AlexsJones/llmfit) - the same `auto` strategy the [workflow-mode router](/resources/llm/routing) uses, so one config drives both modes:

```yaml
# ~/.kdeps/config.yaml
llm:
  strategy: auto
  models:
    - model: llama3.2:1b
      backend: file
    - model: qwen2.5:7b
      backend: gguf
```

```bash
kdeps --model auto
```

If nothing's configured (or none of it scores), `auto` falls through to the same installed-model pick described below, then the same fixed tiers - it's always at least as good as omitting `--model` entirely.

## `--model auto-router`: zero-config, fully automatic

`--model auto` still scores *your configured* `llm.models`. `--model auto-router` (or `KDEPS_AGENT_MODEL=auto-router`) skips `llm.models` entirely, every time, and always goes straight to discovery:

```bash
kdeps --model auto-router
```

1. **Best-fit installed local model** - every cached llamafile, loadable GGUF, and pulled Ollama tag scored via `llmfit`. Requires `llmfit` on `PATH`; skipped (no cost) when it isn't installed.
2. **Cloud fallback** - the first provider with both an API key env var set and a known representative model (`gpt-4o` for OpenAI, `claude-sonnet-4-6` for Anthropic, ...).
3. **Fixed tiers** - if neither finds anything, falls through to the same fixed-order pick described below.

Workflow mode has the same sentinel via `model: auto-router` on a chat resource - see [LLM backends](/resources/llm/routing).

## How a model is picked when none is configured

With no `--model` flag, no saved default, and no `model:` in `~/.kdeps/config.yaml`, kdeps first checks whether `llmfit` is installed (`brew install AlexsJones/llmfit/llmfit`): if it is, and at least one local model (llamafile/GGUF/Ollama) is already downloaded, kdeps starts with whichever downloaded model llmfit scores as the best hardware fit for this machine, skipping the fixed order below entirely.

Otherwise - no `llmfit`, or nothing downloaded yet - kdeps picks the first option that is actually usable, in this fixed order:

1. **llamafile** - the `llamafile` runner binary on `PATH`, or a cached `*.llamafile` in the models directory (a `.llamafile` is self-executing, so no runner is needed).
2. **GGUF** - the first `*.gguf` in the models directory that `llama-server` can load. Files with an unreadable header or a GGUFv1 container are skipped: current llama.cpp builds refuse them (`GGUFv1 is no longer supported`), so serving one would start a server that exits immediately and fail every request.
3. **Cloud** - the first known provider whose API key env var is set (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, ...).
4. **ollama** - the `ollama` binary on `PATH`.

The models directory is `$KDEPS_MODELS_DIR`, or `~/.kdeps/models` when that is unset. If nothing matches, kdeps starts with no model and `/model` lists what you can download.

Each local model server writes its stdout and stderr next to the model file as `<model>.server.log`. When a server fails to become healthy, the tail of that log is included in the error, so a bad model file reports its real cause instead of a bare connection failure.

## Registering a model by URL

`/model <url>` registers a custom model and switches to it. The URL kind is detected automatically:

```bash
# Direct GGUF or llamafile file - downloaded immediately, then served locally
/model https://huggingface.co/user/repo/resolve/main/Qwen2.5-7B-Q4_K_M.gguf
/model https://example.com/rocket-3b.Q4_K_M.llamafile

# Any other http(s) URL is treated as an OpenAI-compatible endpoint
/model http://localhost:1234/v1          # LM Studio / llama.cpp server
/model https://api.together.xyz/v1       # a hosted compat provider
```

Each registered model gets a memorable, kind-prefixed ID so it's easy to recall and retype next time:

- `.gguf` URL -> `gguf-<filename>` (e.g. `gguf-Qwen2.5-7B-Q4_K_M`)
- `.llamafile` URL -> `llamafile-<filename>`
- OpenAI-compatible endpoint -> `api-<host>` (e.g. `api-localhost-1234`)

A collision with an existing model gets the next free `-2`, `-3`, ... suffix, so re-registering never overwrites.

Registered models persist and keep appearing in `/model` and `/model <tab>`:

- `.gguf` / `.llamafile` URLs are added to `~/.kdeps/gguf_versions.yaml` / `llamafile_versions.yaml` (downloaded on registration).
- OpenAI-compatible endpoints are saved to `~/.kdeps/agent-loop-settings.yaml`. No API key is stored; if the endpoint needs one, set `OPENAI_API_KEY` (or `KDEPS_CUSTOM_API_KEY`) in your environment.

## Favorite models

Star models you use often so they lead the `/model` and `/model <tab>` lists and persist across sessions:

```bash
/model favorite gpt-4o          # star it (also: /model fav, /model star)
/model favorite gguf-my-model
/model unfavorite gpt-4o        # remove the star (also: /model unfav)
```

Favorites are saved to `~/.kdeps/agent-loop-settings.yaml`, shown first (marked `★`) with no text typed, and remain selectable even if the model is a cloud model or a not-yet-downloaded alias.

## Searching and downloading from HuggingFace

`/model hff` lets you discover and download GGUF models directly from within the REPL. Set `HF_TOKEN` in your environment to authenticate (required for gated models; increases rate limits for all requests).

```bash
# Search for GGUF repos by keyword (sorted by downloads)
/model hff search qwen3

# List GGUF files and sizes inside a repo
/model hff info unsloth/Qwen2.5-VL-7B-Instruct-GGUF

# Download a specific file - registers it as an alias in ~/.kdeps/gguf_versions.yaml
/model hff download unsloth/Qwen2.5-VL-7B-Instruct-GGUF Qwen2.5-VL-7B-Instruct-Q4_K_M.gguf

# Switch to it immediately after download
/model Qwen2.5-VL-7B-Instruct-Q4_K_M
```

`/model hff download <repo>` without a filename shows the available files (same as `/model hff info`). Downloaded files go to `~/.kdeps/models/` and the alias is the filename without the `.gguf` extension.

## Managing running servers

`/model ps` shows all llamafile and llama-server processes started in the current session:

```
PID      PORT   BACKEND      MODEL                                STATUS
12345    8080   gguf         Qwen2.5-VL-7B-Instruct-Q4_K_M       healthy
12346    8081   file         phi4                                  loading
```

```
/model ps kill phi4           # send SIGKILL, remove port file
/model ps switch phi4         # set active model to an already-running server
```

## See also

- [Agent loop mode](/modes/agent-loop-mode) - overview and starting the REPL
- [LLM backends & routing](/resources/llm/backends) - the workflow-mode equivalent of `auto`/`auto-router`
- [REPL slash commands](/modes/agent-loop-commands) - full command reference
