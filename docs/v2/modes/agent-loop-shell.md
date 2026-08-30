# Shell execution

`bash_exec`, one of the [built-in tools](/modes/agent-loop-tools), runs any shell command and streams output to the terminal. Two keyboard shortcuts change its behavior mid-run:

| Key | Effect |
|-----|--------|
| `Ctrl+C` | Cancel the running tool. Partial output is returned to the LLM as a result so it can decide what to do next. Works for any built-in tool, not only `bash_exec`. |
| `Ctrl+Z` | Detach the process as a background job. `bash_exec` immediately returns `{"status":"backgrounded","job_id":N}` to the LLM. |

Ctrl+C is read directly from the terminal while a tool runs, so it cancels even long-running tools (e.g. a slow `search_local` or `web_scraper`) - the REPL does not rely on the terminal delivering a signal.

`Ctrl+Z` at the REPL prompt (no tool running) suspends kdeps normally (`fg` to resume).

Background jobs are managed with two companion tools:

| Tool | Description |
|------|-------------|
| `bash_job_list` | Show all background jobs with status (`running`/`done`/`failed`), elapsed time, and command |
| `bash_job_wait` | Block until a job completes and return its full output. Pass `job_id` from the backgrounded result. |

Set `KDEPS_ALLOW_BASH=false` to disable all three `bash_*` tools.

## Token savings with rtk (optional)

[rtk](https://github.com/rtk-ai/rtk) is a CLI proxy that compresses command output before it reaches the LLM. `git status` costs ~300 tokens; `rtk git status` costs ~60 for the same information. If rtk is installed, `bash_exec` uses it automatically - nothing to configure.

```text
LLM calls bash_exec("go test ./...")
  -> kdeps asks: rtk rewrite "go test ./..."
  -> rtk answers: rtk go test ./...
  -> kdeps runs the rewritten command
  -> LLM sees filtered output (up to 90% fewer tokens)
```

Install it with `brew install rtk`, or skip it - kdeps runs your commands unchanged when rtk is absent.

| Env var | Effect |
|---------|--------|
| _(none)_ | Auto-detect. rtk is used when it is on `PATH` and passes verification. |
| `KDEPS_RTK=off` | Never use rtk, even if installed. |
| `RTK_DISABLED=1` | Also honored. rtk's own escape hatch, so one variable turns it off everywhere. |

What this does **not** change:

- **Your commands still run.** If rtk is missing, too old, wedged, or has no compression for a command, kdeps runs the original. rtk can never block execution.
- **Permissions are unaffected.** kdeps gates shell commands itself. rtk is only a compressor here, so its own permission verdicts are ignored rather than double-gating you.
- **Workflow mode is untouched.** Only agent loop `bash_exec` uses rtk. Workflow `exec` resources keep raw output, because pipelines parse it downstream.

::: tip Verifying the right rtk
An unrelated crate on crates.io is also named `rtk`. kdeps does not trust the name - it verifies the binary by behavior, so an impostor on your `PATH` is ignored rather than producing broken commands. Check yours with `rtk gain`: it works on the real one.
:::

## See also

- [Built-in Tools](/modes/agent-loop-tools) -- the full tool catalog
- [Tool Execution Monitoring](/modes/agent-loop-monitoring) -- live status lines and stall detection
