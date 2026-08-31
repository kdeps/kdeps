# Tool execution monitoring

How the [agent loop REPL](/modes/agent-loop-mode) shows a running tool's progress and handles one that stalls - applies to every [built-in tool](/modes/agent-loop-tools), not just a specific category.

While a tool runs, the REPL shows a live monitor line - `⠴ bash_exec running (12m34s) · <latest output line>` - refreshed every second, so a long command (a full test suite, a large download) is visibly alive instead of silent. The line is replaced by the usual `... done (elapsed)` summary when the tool finishes.

Every tool gets a meaningful monitor line, not just `bash_exec`: the line is seeded with what the tool is acting on, derived from its arguments - the URL for `web_scraper`/`http_request`, the query for `web_search`/`sql_query`, the path for `search_local`, and so on (`⠴ web_scraper running (3s) · https://example.com`). Tools that stream output (like `bash_exec`) then replace the seed with their latest output line as it flows.

The same status line covers `! <cmd>` / `!! <cmd>` shell commands and `@file` ref expansion: while a bang command is silent the line shows `⠴ ! make lint running (57s)`, and any real output erases the status line first so the two never collide.

The monitor also detects hung tools. Staleness is measured by *silence*, not wall-clock time - a long build that keeps printing never trips it. After 2 minutes without output the line warns (`no output for 3m20s`); after the stall timeout (default 10 minutes of silence, tune with `/model tool set stall-timeout 5m`, `0` disables) the tool is killed and the model receives a structured error explaining the hang so it can retry with a narrower or more verbose command, or run it in the background.

When a tool stalls, the default is **auto-increase**: the stall timeout is bumped by the increment (default 5m) and the bump is announced (`[Auto-stall allocation: stall timeout increased by 5m. New timeout: 15m.]`), so a long silent-but-alive command keeps running without a prompt. This is on by default.

Two other modes are available via `/model tool set autokill <on|off>` (autokill and auto-increase are mutually exclusive - enabling one disables the other):

- `autokill on` - a stalled tool is **killed** at the stall timeout (no increase, no prompt), and the model gets a structured error so it can retry differently.
- `autokill off` - the default auto-increase-and-announce behavior.

To be prompted interactively instead, turn both off in config; the REPL then offers `(i)ncrease` / `(k)ill` when a tool stalls.

Tools marked "cached" in [Built-in Tools](/modes/agent-loop-tools) memoize successful results for the lifetime of the agent process: repeating the same query or URL returns the cached copy instantly instead of refetching. Failed and empty lookups are not cached, so they are retried on the next call. `wolfram_alpha` results are cached the same way.

## See also

- [Built-in tools](/modes/agent-loop-tools) - the full tool catalog
- [Shell execution](/modes/agent-loop-shell) - Ctrl+C/Ctrl+Z, background jobs, rtk
