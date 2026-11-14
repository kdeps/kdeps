# Local model management

Managing which LLM the [agent loop REPL](/agent/) talks to - switching mid-session, auto-routing, and running local model servers.

## Switching models

`/model <name>` switches models during a session. For local backends (`file`, `gguf`), the REPL downloads and starts the server if necessary. It shows the progress until the completions endpoint accepts requests. The first prompt after a switch never gets a "network error" when weights load.

```
/model qwen3.5-4b                     # switch to a known alias
/model default qwen3.5-4b             # save as default startup model
/model default                        # show the current default
```

The model picker and `/model <tab>` tag each name with its backend: `[llamafile]`, `[gguf]`, `[ollama]`, `[openrouter]`, or `[openai enabled]` when you set the key of that provider. The name after the switch stays the model id.

kdeps saves the default model to `~/.kdeps/agent-loop-settings.yaml`. If you do not give `--model`, kdeps loads it automatically at startup.

## Refusal record

kdeps keeps a record of refusals the loop caught from each model: an invented limit ("I can't access the filesystem"), a sandbox story, or a claimed action with no tool run. It saves the model's first sentence, keyed by backend and model, in `~/.kdeps/model-records.json`. When the record has entries, each request to that model includes it as a `<your-record>` block in the system prompt (the `model-refusal-record` harness entry). The block gives each excuse word for word, and states that each excuse was false.

```text
/model record          # openai/fake: 2 caught refusals ... [invented limit] "I can't access the repository."
/model record clear    # wipe the current model's record
```

The record keeps the newest 20 distinct refusals (the `model-record-limit` event); repeats only raise the count.

## Cloud model catalog

The `models/cloud` asset gives the cloud models that `/model list` shows. It also gives the context window, output limit, and thinking and image support of each model. This data does not come from the binary. If a provider renames or adds a model, run `kdeps update models/cloud`.

```yaml
# models/cloud (as shipped, one entry)
version: 1.0.0
models:
  - id: "claude-opus-4-8"     # the API model ID
    backend: anthropic        # kdeps backend that serves it
    desc: "most capable"      # label in /model list
    envVar: ANTHROPIC_API_KEY # a key here marks the provider ready
    thinking: true            # supports extended thinking
    images: true              # accepts image input
    contextWindow: 1000000    # input tokens; drives compaction and request sizing
    maxOutputTokens: 128000   # per-call output cap sent to the provider
```

A model that is not in the catalog also works by name (`/model my-fine-tune`). kdeps uses the defaults of the provider for it.

OpenRouter is its own backend. `/model list` shows its free text models from `models/openrouter` (prompt and completion price both zero). GitHub Actions collects this list each day (`make harvest-openrouter`). To refresh it, run `kdeps update models/openrouter`. You can also give a model name yourself, for example a paid id or the `openrouter/free` router. OpenRouter has no single default model, so it is not in the cloud fallback of auto-router.

Free ids (ending `:free`) share one account limit for each minute (`openrouter-free-per-minute`, 20 by default). All free-model requests in the process wait in one queue, so the process stays in the limit. This includes requests from the REPL, judges, compaction, and workflow resources. The queue does not print a line for each request. If the agent loop waits for longer than the interval, it prints `openrouter: waiting <duration> before the next free-model request`.

The signatures that classify OpenRouter errors are event assets (`openrouter-refusal`, `openrouter-rate-limit`, `openrouter-provider-limit`, each a `patterns:` list). The free-model size ranking reads the `params` field in `models/openrouter`, which the harvester fills from the model id.

## Same alias, different backends

The llamafile, GGUF, and Ollama registries can each have a different entry with
the same bare alias. For example, `qwen3.5` can be a different download in the
llamafile and GGUF catalogs, or an Ollama tag that you pulled locally. When
kdeps finds this, `/model list` and `/model <tab>` show each of these entries
with its backend. They do not hide an entry:

```
llamafile:qwen3.5
gguf:qwen3.5
```

Switch to a specific one with its qualified name:

```
/model gguf:qwen3.5
```

If you type the bare name that is not clear, kdeps selects the backend in its
usual order (`ollama` > `gguf` > `llamafile`). It prints a one-line notice
that tells you the backend that it selected:

```
/model qwen3.5
"qwen3.5" is ambiguous across backends (llamafile:qwen3.5, gguf:qwen3.5) -- using gguf:qwen3.5. Use the full name to pick a specific one.
```

This does not change unique names, which are almost all aliases. You need the
`backend:` prefix only when kdeps tells you to use it.

## `--model auto`: route across your configured models

`--model auto` (or `KDEPS_AGENT_MODEL=auto`) uses [`llmfit`](https://github.com/AlexsJones/llmfit) to pick the best hardware fit from the `models:` list under `llm:` in `~/.kdeps/config.yaml`. The [workflow-mode router](/workflow/resources/llm-routing) uses the same `auto` strategy, so one config drives both modes:

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

With nothing configured, or nothing scoring, `auto` falls through to the installed-model pick and fixed tiers below, so it is never worse than omitting `--model`.

## `--model auto-router`: zero-config, fully automatic

`--model auto` scores *your* `models:` list under `llm:`. `--model auto-router` (or `KDEPS_AGENT_MODEL=auto-router`) skips that list entirely, every time, and always goes straight to discovery:

```bash
kdeps --model auto-router
```

1. **Best-fit installed local model** - `llmfit` gives a score to each cached llamafile, loadable GGUF, and pulled Ollama tag. This step needs `llmfit` on `PATH`. If `llmfit` is not installed, kdeps skips this step (no cost).
2. **Cloud fallback** - the first provider that has an API key env var and a known representative model (`gpt-4o` for OpenAI, `claude-sonnet-4-6` for Anthropic, ...).
3. **Fixed tiers** - if the two steps above find nothing, kdeps uses the fixed order below.

Workflow mode has the same sentinel with `model: auto-router` on a chat resource - see [LLM backends](/workflow/resources/llm-routing).

## How a model is picked when none is configured

With no `--model`, no saved default and no `model:` in `~/.kdeps/config.yaml`, kdeps checks for `llmfit` (`brew install AlexsJones/llmfit/llmfit`). If you have `llmfit` and one or more local models (llamafile/GGUF/Ollama), kdeps starts with the model that has the best llmfit score. It skips the order below.

Otherwise - no `llmfit`, or nothing downloaded yet - kdeps picks the first option that is actually usable, in this fixed order:

1. **llamafile** - the `llamafile` runner binary on `PATH`, or a cached `*.llamafile` in the models directory (a `.llamafile` is self-executing, so you do not need a runner).
2. **GGUF** - the first `*.gguf` in the models directory that `llama-server` can load. kdeps skips files with a header that it cannot read or with a GGUFv1 container. The current llama.cpp refuses them (`GGUFv1 is no longer supported`), and the server stops immediately.
3. **Cloud** - the first known provider that has its API key env var (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, ...).
4. **ollama** - the `ollama` binary on `PATH`.

The models directory is `$KDEPS_MODELS_DIR`, or `~/.kdeps/models` when that is unset. If nothing matches, kdeps starts with no model and `/model` lists what you can download.

Each local model server writes stdout and stderr to `<model>.server.log` next to the model file. When a server fails to become healthy, the error includes the log's tail, so a bad model file reports its real cause.

## Registering a model by URL

`/model <url>` registers a custom model and switches to it. kdeps finds the URL type automatically:

```bash
# Direct GGUF or llamafile file - downloaded immediately, then served locally
/model https://huggingface.co/user/repo/resolve/main/Qwen2.5-7B-Q4_K_M.gguf
/model https://example.com/rocket-3b.Q4_K_M.llamafile

# Any other http(s) URL is treated as an OpenAI-compatible endpoint
/model http://localhost:1234/v1          # LM Studio / llama.cpp server
/model https://api.together.xyz/v1       # a hosted compat provider
```

Each registered model gets a memorable, kind-prefixed ID so it is easy to recall and retype next time:

- `.gguf` URL -> `gguf-<filename>` (e.g. `gguf-Qwen2.5-7B-Q4_K_M`)
- `.llamafile` URL -> `llamafile-<filename>`
- OpenAI-compatible endpoint -> `api-<host>` (e.g. `api-localhost-1234`)

A collision with an existing model gets the next free `-2`, `-3`, ... suffix, so re-registering never overwrites.

Registered models persist and keep appearing in `/model` and `/model <tab>`:

- kdeps adds `.gguf` / `.llamafile` URLs to `~/.kdeps/gguf_versions.yaml` / `llamafile_versions.yaml`. It downloads the file when it registers it.
- kdeps saves OpenAI-compatible endpoints to `~/.kdeps/agent-loop-settings.yaml`. It does not store an API key. If the endpoint needs a key, set `OPENAI_API_KEY` (or `KDEPS_CUSTOM_API_KEY`) in your environment.

## Favorite models

Star models you use often so they lead the `/model` and `/model <tab>` lists and persist across sessions:

```bash
/model favorite gpt-4o          # star it (also: /model fav, /model star)
/model favorite gguf-my-model
/model unfavorite gpt-4o        # remove the star (also: /model unfav)
```

kdeps saves favorites to `~/.kdeps/agent-loop-settings.yaml`. When you type no text, favorites show first (with the mark `★`). You can always select them, also if the model is a cloud model or an alias that you did not download.

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

`/model hff download` needs both the repo and the file; `/model hff info <repo>` lists the files. Downloaded files go to `~/.kdeps/models/` and the alias is the filename without the `.gguf` extension.

`search` and `info` are the `hf_search` and `hf_repo_files` tools: `python:` blocks in their tool assets, which the model can also call. `KDEPS_HF_API_URL` points them at another HuggingFace API base.

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

- [Agent mode](/agent/) - overview and starting the REPL
- [LLM backends & routing](/workflow/resources/llm-backends) - the workflow-mode equivalent of `auto`/`auto-router`
- [REPL slash commands](/agent/commands) - full command reference
