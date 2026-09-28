# Built-in tools

The [agent loop](/agent/) has access to a set of built-in tools that the LLM can call without any YAML configuration. Tools that require credentials are only registered when the relevant environment variable is set.

*Applies to agent mode.*

## Fenced tools only, and narration

Two instructions go into the system preamble for **every** model whenever tools are registered:

- **Use the fenced kdeps tools, not a built-in sandbox.** Every capability the model has here is a kdeps tool - including `bash_exec` and the file tools. A model trained with a code-interpreter / "run code" / "analysis" habit will otherwise act against its own empty `/mnt/data`-style environment and conclude it "cannot access" anything. The rule is restated in one line on every turn after the first. M365 Copilot gets an extra, stronger version of this on top (its habit is the most persistent). The guidance also tells the model that kdeps is a parser and an interpreter: it parses a matched `<invoke>` block out of the message at runtime and interprets it. That is not the model's built-in code interpreter. Writing the block is the call, not an example. On a backend with no native tool channel, that block is a LITERAL invoke block: the characters `<invoke>` and `</invoke>` written as text in the message. One matched block does it -

  ```
  <invoke name="read_file">
  <parameter name="file_path">cmd/serve.go</parameter>
  </invoke>
  ```

  and the runtime hands back the real output. The same preamble lists every registered tool in an `<available_tools>` block: name, parameters, and an `<invoke>` skeleton for each. That block is what the model copies a name from. Later turns repeat the list next to the one-line reminder. Backends with a native tool channel (Anthropic, OpenAI) also receive the tools as a native schema; kdeps recovers an `<invoke>` / `<tool_call>` block written as text if a native model emits one anyway.
- **Simulated sandbox sessions are caught, and escalated.** When a round makes no tool call but the text reads like a failed code-interpreter session - `NO CONTENT AVAILABLE`, `/mnt/data`, an "expired" or "reset" session, "cannot access the filesystem" - the loop treats it as a hallucination (kdeps never emits those strings and nothing ran). A model that repeats the same hallucination a second time within one turn gets a second, sharper nudge instead of the turn silently accepting the repeat as a real answer - each corrective nudge (sandbox, fabricated `<tool_response>`, silent round) fires up to twice per turn, not once. If the model still hasn't made a real call after both nudges, the turn does not settle on the fake transcript unflagged: in goal mode the active task is recorded failed, not done; otherwise the returned text is prefixed with a clear "this describes a sandbox that doesn't exist here, treat it as unverified" banner (the model's words are kept, just flagged). Separately, once a session has produced even one such hallucination, every later turn resends the full fenced-tools guidance (with the worked `<invoke>` examples) instead of the one-line reminder - a model that has shown this failure mode gets more reinforcement, not less.
- **A premature "sorry, I can't" is pushed back on when there was real progress.** When a round ends with no tool call and the text reads as declining to continue ("I cannot complete this," "unable to," "no access," ...), but a tool call already succeeded earlier this turn and nothing is currently failing, the loop does not accept the refusal as final - it tells the model tool calls just worked and to try a different approach, or state exactly what is blocking it, bounded to the same two-nudges-per-turn cap as every other corrective nudge here. A give-up reply with no progress behind it (nothing succeeded yet this turn) is never second-guessed this way - there is nothing to push back with. This applies with or without goal mode: in goal mode it fires *before* the active task would otherwise be recorded failed, so a premature refusal never gets logged as a settled failure while the nudge budget remains.
- **Good tool-calling behavior is praised, not just bad behavior flagged.** Three cases, each a `[GOOD]` acknowledgment landing in the tool result message itself - part of conversation history, not just the terminal, so it carries forward into later tasks in the session:
  - A failed call that then succeeds on retry - the positive counterpart to `[TOOL FAILED]`.
  - A real tool call right after a fake sandbox session was detected - specifically calling out that this one reached the real filesystem, not the simulated one.
  - Any of the first 3 successful tool calls in a session, even with no prior failure - building the habit early. After that, an ordinary success gets no banner; constant praise for routine calls would just be noise.
  The mandatory handshake's own tool result carries the same kind of line on a correct call, and a retry nudge there reshows the exact `<invoke>` syntax with a running miss count, framed as help rather than a rebuke.
- **Narrate before each tool call.** The model is asked to say in one present-tense sentence what it is about to do ("Reading config.yaml to check the timeout.") before every call. That line is printed to the terminal - without it the loop is silent between actions, because the streamer writes tool-round output to an internal buffer. Suppressed when `StreamFinalOnly` is set.

## Session-integrity handshake

Off by default; enable with `/handshake on`, which verifies the current model immediately (before your very next prompt), not just after a future model change. Its purpose is to enforce that the model actually uses real kdeps tool calls, not fabricated text - right after a model switch, a resumed session, or a compaction/fold that rewrites the context, the model's tool-calling path against that new state is unproven, and it could fabricate a plausible "I called X, it returned Y" in plain text instead of making a real call. Before the next prompt is sent, kdeps runs a challenge/response round: it issues a random 4-digit code and instructs the model, as a normal user-turn message (not a system prompt a backend might deprioritize as background context), to call the internal `session_handshake` tool with it. kdeps verifies a real, correctly-valued tool call actually reached the registry - not text that merely looks like one. A miss where the model is at least engaging - calling the tool, just with the wrong code - is retried with a corrective nudge for as long as it takes; there is no attempt cap for that case, since giving up and proceeding unverified is the exact failure this exists to prevent. But a model that ignores the directive entirely, attempting no tool call at all across 5 consecutive tries, is not going to get there no matter how long kdeps waits - that case prints a `[handshake]` warning that the check is unverified and lets the turn proceed, rather than bricking the session on a model that can't do this at all. Each retry reshows the exact `<invoke>` syntax and states how many attempts have missed so far, framed as help toward getting it right rather than a rebuke - the same challenge code the whole way through (see below), never a moving target. After 2 consecutive misses the nudge escalates: live testing against a small local model showed it repeatedly writing sentences ABOUT calling the tool ("I will call...", "Here is the block:") wrapped around an empty code fence, instead of the literal tag text itself - the escalated nudge drops the friendly framing, anchors on the model's own earlier real tool call (the bash_exec evidence step) as proof it can already do this, and explicitly forbids narration, code fences, or any text before/after the block. The synthetic conversation itself (warm-up plus every retry) is also capped to the first exchange plus the 3 most recent - unbounded, it grows across warm-up, evidence, and every retry, and live testing showed that ballooning a small local model's prompt from ~3k to over 50k tokens after a few retries, making its answers worse rather than better as the context grew. The first exchange is kept rather than dropped for a second reason: the m365 backend decides whether a request continues an existing conversation or starts a new one by fingerprinting the first message in the array, so a plain sliding window that eventually evicts it would make m365 silently start a brand-new conversation - with a fresh generic system prompt and no memory of the evidence step - on every retry past the cap. Every handshake directive (the challenge, the evidence step, and both retry nudges) now says explicitly that the block goes outside any markdown code fence, not just "nothing else around it" - fencing was the actual live failure mode, so it is named directly rather than left implied. Separately, as a safety net for when a model fences it anyway: a genuine, correctly-coded `<invoke>` block wrapped in a markdown fence out of habit is still recognized - the normal text-salvage path deliberately protects fenced code from being parsed as a real call (so a documentation example for a different tool is never misread as one), but that protection is narrowed for `session_handshake` specifically in this one round, where the risk it guards against does not apply. The grounding message also names `session_handshake` as a real, registered tool (a compact one-line note, not the full tool catalog - a session with many registered tools sending its whole catalog on every handshake round was measured ballooning a small local model's prompt to tens of thousands of tokens for what should be a two-line request), so the model isn't asked to call a tool with nothing else confirming it's real. The exchange itself never touches session history, so it leaves no trace in `/prompt` or a saved transcript. Set `KDEPS_DEBUG=1` to log each attempt. `/handshake` alone reports the current state; the setting persists across sessions.

The directive tells the model that kdeps is a parser and an interpreter and will parse a matched `<invoke>` block out of the message at runtime and interpret it. Not the model's code interpreter. It shows the literal syntax to copy, with the actual code already filled in:

```
<invoke name="session_handshake">
<parameter name="code">4821</parameter>
</invoke>
```

The exchange also carries a system message grounding the request: the same tool-use guidance every normal turn already includes, not a one-off line invented just for this. A bare user prompt plus a raw tool schema and nothing else reads as unusually sparse next to a normal turn's full preamble - even to models that make real tool calls fine on ordinary turns - leading them to reason the tool "wasn't really available" and refuse to call it.

Before the invoke request itself, kdeps also asks two warm-up questions - "how do you invoke a kdeps tool?" and "how many tools do you have?" - answers not checked, but establishing an ongoing conversation about kdeps tools before asking the model to actually call one, rather than a cold open. Each attempt (warm-up included) stays in the same growing conversation, so a retry sees its own prior miss as context instead of a repeat cold start.

After the warm-up and before the actual challenge, kdeps runs one more evidence step (skipped only if `bash_exec` isn't registered): it asks the model to call `bash_exec` with `pwd` and see the real result. This is deliberately evidence, not another assertion - a model that has reasoned itself into believing it is running in a code-interpreter sandbox and "can't do anything" is more likely to be moved by a genuine result it just saw than by more text telling it otherwise. On success, the response is followed by a short reinforcement line ("that came from a real shell, not a sandbox") that stays in the conversation the challenge round builds on. This step is best-effort - it retries up to 3 times with a nudge that spells out the difference explicitly (a normal assistant session may trigger a tool call through its own native channel outside the visible text; kdeps gives no such channel, so the `<invoke>` block itself has to be the literal text sent) and then gives up quietly rather than blocking the turn, since the actual security gate is the challenge itself. For the m365 backend specifically - the one with the most aggressive observed "run code" / "Coding and executing" habit - the handshake's grounding message also carries the same sandbox-specific reinforcement normal turns get, not a bare version of it.

## Parameter-name synonyms

A model sometimes names a tool's argument something plausible but not exact - `grep`'s `pattern` instead of `search_local`'s `query`, `cat`'s `path` instead of `read_file`'s `file_path`. These synonym keys are normalized to the key the tool actually expects before dispatch, without clobbering a real key already present. This is scoped to parameter names only: kdeps does not maintain a separate table of alternate *tool* names (e.g. routing a call to `grep` as if it were `search_local`) - the tool list already tells the model the real name to call, and a second, silently-redirecting name for the same tool added maintenance cost without fixing the cases where a model loses track of what tools it has (see "Tool-list amnesia" below).

## Memory tools

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `memory_save` | Save a fact to persistent memory. Injected into every LLM call automatically. |
| `memory_search` | Search memory entries by key or value. At most 20 matches, best first (key hit, then more query words, then newer); each value cut at 500 characters. |
| `memory_delete` | Remove a memory entry by key. |
| `memory_query` | Run an expr-lang relational query over agent state: `memory` (persistent entries), `tool_calls` (recent tool call history), `tasks` (active goal's task list). Supports `filter()`, `map()`, `join()`, `union()`. |

Memory is stored per-project at `~/.kdeps/memory/<encoded-cwd>/memory.bolt`. Facts persist across sessions and are auto-extracted from every turn - the agent can write `[MEMORY: key] value` on its own line to persist a fact without calling `memory_save`. See [Persistent memory](/agent/memory) for details.

The `memory_*` tools are how the *model* reads and writes memory during a turn. To inspect the store yourself from the REPL, use `/memory` (overview), `/memory list` (every entry), and `/memory search <query>` - see [REPL slash commands](/agent/commands).

## Identity tool

Always available. `identity_get` returns the agent's configured name, email, and address - see [Agent Identity](/reference/advanced-config#agent-identity) for how to set one. Returns "No identity configured for this agent." when unset. Never returns account credentials, even if configured; a model that can read a password can leak it in its own output.

## Shell execution

`bash_exec` runs any shell command and streams output to the terminal, with Ctrl+C to cancel, Ctrl+Z to background it, and companion `bash_job_list`/`bash_job_wait` tools. If [rtk](https://github.com/rtk-ai/rtk) is installed, output is compressed automatically before it reaches the LLM (up to 90% fewer tokens).

A command is HTML-unescaped before it's validated and run - some models emit `&amp;&amp;` for `&&`, `&quot;` for `"`, or `&lt;`/`&gt;` for `<`/`>` (e.g. when the text passed through an HTML-rendering step upstream). `sql_query` gets the same treatment, since an escaped comparison operator (`WHERE x &lt; 5`) is the same kind of syntax corruption there.

See [Shell Execution](/agent/shell) for the full keyboard-shortcut and rtk reference.

## File operations

Always available. No environment variables required.

| Tool | Description |
|------|-------------|
| `read_file` | Read file contents with a 1-based line number on every line (plain text, plus PDF/DOCX/EPUB/RTF/ODT extraction) |
| `write_file` | Write or overwrite a file |
| `edit_file` | `command`-dispatched editor: `view`, `str_replace`, `insert`, `patch`, `replace_symbol`, `undo_edit` |
| `list_files` | List directory contents |
| `md5_file` | Compute a file's MD5 hash - cheap way to check whether content actually changed |
| `tail_file` | Read the last N lines of a file without loading the whole thing |

Every line `read_file` returns is prefixed with its 1-based line number, and the line itself is rendered `cat -A` style: tabs show as `^I`, other control characters as `^X`, and a `$` marks the true end of the line (`  42⇥code$`) - so indentation, trailing whitespace, and stray control characters are visible instead of silently hidden. Those line numbers are what `edit_file`'s `insert` and `view` commands point at.

`read_file`, `tail_file`, and `md5_file` treat `file_path` as optional: omit it and the tool operates on the file most recently read, edited, or written this session. This covers the common slip where the model means "the file I was just looking at" and calls `read_file` with only `offset`/`limit`. `write_file` and `edit_file` always require an explicit path.

`write_file` and `edit_file` print a **colored diff** of what changed under the tool call - removed lines in red, added lines in green, with a couple of context lines - so you can see every change the agent makes at a glance. Large diffs (e.g. writing a whole new file) are capped. The diff is shown in the terminal only; the model receives a concise result, not the ANSI-colored text.

### Jumping from search_local to read_file with match_id

Every `search_local` result that finds its query in the file carries `match_id`, `line`, and `revision` alongside the usual `path`/`snippet` fields. Pass `match_id` to `read_file` instead of `file_path` to jump straight to that hit - `context_before`/`context_after` (default 20 each) control how much surrounding code comes back, the same defaults `edit_file`'s `anchor` view uses:

```xml
<invoke name="search_local">
  <parameter name="path">/app</parameter>
  <parameter name="query">func handleLogin</parameter>
</invoke>
```

```json
{"results": [{"path": "/app/server.go", "match_id": "match-4a9f21bc", "line": 112, "revision": "sha256:...", "snippet": "..."}]}
```

```xml
<invoke name="read_file">
  <parameter name="match_id">match-4a9f21bc</parameter>
  <parameter name="context_after">40</parameter>
</invoke>
```

A `match_id` is process-local and capped in count (oldest evicted first) - it's for chaining a search into a read within the same session, not a durable reference. An unresolvable one (evicted, or from a different process) is a clear error naming it; `file_path` given explicitly always takes priority over `match_id`.

### edit_file - six commands

`edit_file` takes a `command`. The reliable primitive is still an exact,
byte-verified string swap - there is no fuzzy matching - but `str_replace` can
now resolve that exact text from an anchor pair instead of requiring you to
retype the whole block, and `occurrence` picks a specific match when the text
repeats.

**`view`** - `edit_file` with `command: view` and a `file_path` prints the file
with a 1-based line number on every line (plain text, not `read_file`'s
visible-whitespace rendering), followed by the file's `[revision
sha256:...]`. Pass `view_range: [start, end]` (1-based
inclusive, `end` `-1` = to end of file) for a slice, or `anchor` (a unique
string) to show the region around it instead - `context_before`/
`context_after` control how many lines of context (default 20 each):

```xml
<invoke name="edit_file">
  <parameter name="command">view</parameter>
  <parameter name="file_path">/app/server.go</parameter>
  <parameter name="anchor">func handleLogin</parameter>
  <parameter name="context_after">40</parameter>
</invoke>
```

A `view` also satisfies the read-before-edit requirement below.

**`str_replace`** - pass `old_str` (the exact current text) and `new_str`.
`old_str` must match the file **byte-for-byte** - indentation and all:

- 0 matches -> `old_str did not appear verbatim` (copy it from a `view`).
- 2+ matches -> the error lists every line the text starts on; either add
  surrounding lines to make it unique, or pass `occurrence: N` (1-based) to
  pick one of the listed matches directly.
- 1 match (or a resolved `occurrence`) -> the swap is written and the result
  is a **numbered snippet of the changed region** plus the file's new
  `[revision ...]`.

Instead of `old_str`, pass `start_anchor`/`end_anchor` to replace everything
from one unique string through another (inclusive) - useful for a large block
you don't want to retype:

```xml
<invoke name="edit_file">
  <parameter name="command">str_replace</parameter>
  <parameter name="file_path">/app/server.go</parameter>
  <parameter name="start_anchor">func handleLogin</parameter>
  <parameter name="end_anchor">\n}\n</parameter>
  <parameter name="new_str">func handleLogin(w http.ResponseWriter, r *http.Request) {\n\t// ...\n}</parameter>
</invoke>
```

**`insert`** - pass `insert_line` (0 = before the first line, N = after line N)
and `new_str`. Returns the same numbered snippet. Instead of `insert_line`,
pass `insert_before_anchor`/`insert_after_anchor` (a unique string) to
position by content instead of a line number you'd otherwise have to look up.

**`patch`** - pass `patch`, a standard unified diff with one or more `@@` hunks.
Each hunk's context and removed lines must match the file byte-for-byte and
appear **exactly once**; hunks are resolved against the file's original
content and applied atomically - if any hunk fails to resolve, nothing is
written:

```xml
<invoke name="edit_file">
  <parameter name="command">patch</parameter>
  <parameter name="file_path">/app/server.go</parameter>
  <parameter name="patch">@@ -10,3 +10,3 @@
 func main() {
-	log.Println("starting")
+	log.Println("starting v2")
 }
</parameter>
</invoke>
```

**`replace_symbol`** - pass `symbol` (a function/type/class/etc. name) and
`new_str` to replace its whole declaration. The symbol must be declared
**exactly once** in the file - same "ambiguous, add specificity" rule as
`str_replace`, with the error listing every declaration line. `symbol_kind`
(e.g. `"function"` or `"type"`) narrows which declaration keywords count,
useful only when a function and a type share a name:

```xml
<invoke name="edit_file">
  <parameter name="command">replace_symbol</parameter>
  <parameter name="file_path">/app/server.go</parameter>
  <parameter name="symbol">handleLogin</parameter>
  <parameter name="new_str">func handleLogin(w http.ResponseWriter, r *http.Request) {
	// ...
}</parameter>
</invoke>
```

Where the symbol's block *ends* is found **lexically, not by a language
parser**: a brace-depth scan for C-like bodies (Go, JS/TS, Java, C/C++/C#,
Rust - braces inside string/comment text are correctly ignored), or an
indentation scan when there's no opening brace nearby (Python `def`/`class`).
This covers ordinary top-level declarations reliably without requiring
gopls/pyright/tree-sitter or any other external parser to be installed, but
it is not full-language-semantics: it does not parse JavaScript embedded
inside an HTML `<script>` tag, and it cannot distinguish two symbols that are
only disambiguated by real scoping (that surfaces as the same "ambiguous"
error `str_replace` gives for a repeated string). `view` accepts the same
`symbol`/`symbol_kind` (plus `context_before`/`context_after`) to show a
declaration without knowing its line range first.

**`undo_edit`** - reverts the last mutation on that file (a per-file history
kept for the session).

**Read before you edit.** `str_replace`, `insert`, `patch`, and
`replace_symbol` refuse to touch a file that was not read this turn
(`read_file`, or `edit_file command: view`) - editing a file you have not
looked at is how a change lands in the wrong place. A file you just wrote
with `write_file`, or just edited, counts as read. The file's line-ending
style is preserved on write.

**Revision checks.** Every `view` and every successful mutation reports a
`[revision sha256:...]` token. Pass it back as `revision` on a later
`str_replace`/`insert`/`patch`/`replace_symbol` to reject the edit if the file
changed since you read it, instead of silently overwriting someone else's
change:

```json
{"error": "edit_file: /app/server.go changed since revision sha256:1a2b3c4d5e6f was read (it is now sha256:9f8e7d6c5b4a) - view the file again and retry"}
```

Omitting `revision` skips the check entirely - it's an extra safeguard on top
of the read-this-turn gate above, not a replacement for it.

**Preview first with `dry_run`.** `str_replace`, `insert`, `patch`, and
`replace_symbol` all accept `dry_run: true` - the same diff and would-be
`[revision ...]` are returned, but nothing is written to disk and nothing is
pushed to the undo history.

**Reject a broken edit with `validate_syntax`.** `str_replace`, `insert`,
`patch`, and `replace_symbol` accept `validate_syntax: true` - the resulting
file is checked *before* anything is written, and the edit is refused
outright on failure, so a malformed change is never written and never needs
rolling back. Combine with `dry_run` to check without applying. What "check"
means depends on the file extension:

| Extension | Check |
|-----------|-------|
| `.go` | Real parse via Go's standard `go/parser` |
| `.json` | Real parse via `encoding/json` |
| `.yaml`/`.yml` | Real parse via `yaml.v3` |
| `.js`/`.ts`/`.java`/`.c`/`.cpp`/`.cs`/`.rs`/`.php`/`.swift`/`.kt`/etc. | Balanced `()[]{}` scan (string/comment-aware) - lexical, not a real parse |
| `.py`/`.rb`/`.sh`/etc. | Same balanced-delimiter scan, `#` treated as a comment |
| anything else | No-op - `validate_syntax` never blocks a file type it can't check |

### Failed tool calls are fed back to the model

Every tool failure is returned to the model as `{"error": ...}` **plus a
`[TOOL FAILED]` banner** ("nothing changed - fix the cause and retry, or say it
failed"), so a skimmed error is hard to miss. The turn will not end on a "done"
claim made right after a work tool failed: the loop pushes back for a real
success or an honest "it failed" - up to twice, with the second push-back
reading as a repeat, same bound as the other corrective nudges. If the model
still hasn't acknowledged the failure after both, the turn does not settle on
the bald claim unflagged: the response is followed by a notice naming the
tool and its error, so the claim is kept but clearly marked unresolved. An
honest admission at any point (in whatever words) is accepted immediately,
no nudge needed. With a goal active, `task_complete` on such a task is
refused, and a prose "done" records the task **failed**, not done.

Loop-generated notices - goal transitions, budget changes, forced task failures,
context-window trims - are injected into the model's context as `[kdeps] ...`
messages, since the human at the REPL cannot act on them but the model can
adjust.

## Web and search

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `web_search` | (none - uses DuckDuckGo) | Search the web (30s timeout, cached) |
| `wikipedia` | (none) | Fetch a Wikipedia article (30s timeout, cached) |
| `web_scraper` | (none) | Fetch and extract text from any URL (60s timeout, cached) |
| `serpapi_search` | `SERPAPI_API_KEY` | Google search via SerpAPI (30s timeout, cached) |
| `exa_search` | `EXA_API_KEY` or `METAPHOR_API_KEY` | Neural search via Exa (cached) |
| `perplexity_search` | `PERPLEXITY_API_KEY` | Search via Perplexity (30s timeout, cached) |

Web and search tools carry a hard timeout so a hung remote endpoint cannot stall the turn. Ctrl+C during any tool call cancels the in-flight request immediately and skips the round's remaining tools. Tools marked "cached" memoize successful results for the process lifetime; failed/empty lookups are retried.

While any tool runs, the REPL shows a live status line and detects hangs via a stall timeout - see [Tool Execution Monitoring](/agent/monitoring) for the full mechanics.

## Permission modes

Set `KDEPS_PERMISSION_MODE` to restrict which tools the agent may call:

```bash
KDEPS_PERMISSION_MODE=read-only ./kdeps          # reads, searches, lookups only
KDEPS_PERMISSION_MODE=workspace-write ./kdeps    # adds file writes and bash_exec
KDEPS_PERMISSION_MODE=danger-full-access ./kdeps # no restrictions (default)
```

Blocked calls return a `permission denied` tool error to the model, which explains the restriction instead of executing. Tools not in the built-in policy - including workflow, component, and agency tools - require `workspace-write`, so `read-only` blocks anything that could mutate state.

## Git commit attribution

Commits the agent creates carry a co-author trailer naming kdeps and the model that wrote them. Switching models mid-session with `/model` is normal, so the trailer records which one was actually driving:

```text
Co-Authored-By: kdeps (deepseek/deepseek-reasoner) <noreply@kdeps.com>
```

How the model is named depends on where it runs:

| Model | Trailer |
|-------|---------|
| Cloud provider | `kdeps (deepseek/deepseek-reasoner) <noreply@kdeps.com>` |
| Cloud provider | `kdeps (openai/gpt-4o-mini) <noreply@kdeps.com>` |
| Ollama | `kdeps (ollama/llama3.2) <noreply@kdeps.com>` |
| llamafile | `kdeps (hfuser/gemma4-2-9b llamafile) <noreply@kdeps.com>` |
| GGUF | `kdeps (hfuser/gemma4-2-9b gguf) <noreply@kdeps.com>` |

Cloud and Ollama models are namespaced by their provider (`provider/model`). Local llamafile and GGUF models already carry their HuggingFace namespace in the name, so the runtime is appended instead - the same repo is often published as both, and the name alone cannot tell them apart.

With no model configured, the trailer falls back to `Co-Authored-By: kdeps <noreply@kdeps.com>`.

A configured [identity](/reference/advanced-config#agent-identity) takes priority over all of the above: with `identity.name`/`identity.email` set, the trailer becomes `Co-Authored-By: Sales Bot <sales-bot@example.com>` instead of naming the model.

## Lean mode

The full tool catalog (~55 tools - `bash_exec`, `web_search`, `web_scraper`, `wikipedia`, `http_request`, external API tools, plus the lean set below) is **on by default for every session**. Trim it down when you want less prompt weight (each tool costs tokens twice - once as a native tool schema, once as prose in the tool-use guidance) or a restricted surface for CI/automation:

```
/tools lean     # this session only, switch back any time
/tools full     # switch back
/tools          # show current mode and tool count
```

The choice persists automatically across sessions - no flag needed - the same way `/model tool set` settings do. Lean mode keeps `read_file`, `write_file`, `edit_file`, `list_files`, `code_search`, `code_definition`, `code_references`, `code_symbols`, `code_hover`, `code_diagnostics`, `search_local`, `load_document`, `calculator`, `embedding_vectorize`, `embedding_search`, `transcribe_audio` (~16 tools).

`KDEPS_LEAN_MODE`/`KDEPS_AGENT_PRESET` (below) start a session already in lean mode and take priority over the persisted `/tools` choice.

## Agent presets

`KDEPS_AGENT_PRESET` combines lean mode with a permission mode in one flag for common workflows:

```bash
KDEPS_AGENT_PRESET=audit       # read-only, lean tools
KDEPS_AGENT_PRESET=explain     # read-only, lean tools
KDEPS_AGENT_PRESET=implement   # workspace-write, lean tools
```

| Preset | Permission mode | Tool set |
|--------|----------------|----------|
| `audit` | ReadOnly | Lean (no bash, no network) |
| `explain` | ReadOnly | Lean (no bash, no network) |
| `implement` | WorkspaceWrite | Lean + file writes |

`IsLeanOrPreseted()` returns `true` when either `KDEPS_LEAN_MODE` or `KDEPS_AGENT_PRESET` is set.

## Computation

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `calculator` | (none) | Evaluate math expressions |
| `wolfram_alpha` | `WOLFRAM_APP_ID` | Wolfram Alpha queries |

## Data and SQL

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `sql_list_tables` | `KDEPS_SQL_DB_PATH` or connection config | List tables in a database |
| `sql_describe_table` | same | Describe a table's columns and types |
| `sql_query` | same | Execute a SELECT query |

## Embeddings and reranking

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `embedding_vectorize` | (none) | Convert text to embeddings and index it in the local embedding DB |
| `embedding_search` | (none) | Semantic search over the local embedding DB |
| `retrieve_context` | `KDEPS_RAG_BASE_URL` | Retrieve chunks from a remote RAG endpoint (only registered when the URL is set) |
| `cohere_rerank` | `COHERE_API_KEY` | Rerank results using Cohere |
| `voyageai_rerank` | `VOYAGEAI_API_KEY` | Rerank results using VoyageAI |
| `jina_rerank` | `JINA_API_KEY` | Rerank results using Jina |

## Actions and integrations

| Tool | Required env var | Description |
|------|-----------------|-------------|
| `zapier_list_actions` | `ZAPIER_NLA_API_KEY` | List available Zapier NLA actions |
| `zapier_run_action` | `ZAPIER_NLA_API_KEY` | Execute a Zapier NLA action |
| `google_cache_create` | (Google credentials) | Create a Google AI cached content object |
| `google_cache_list` | (Google credentials) | List Google AI cached content objects |
| `google_cache_delete` | (Google credentials) | Delete a Google AI cached content object |

## Resource-backed tools

These always-on tools invoke the corresponding kdeps executor directly:

| Tool | Description |
|------|-------------|
| `http_request` | Make an HTTP request (GET/POST/PUT/DELETE/PATCH) |
| `search_local` | Search the local document index |
| `transcribe_audio` | Transcribe an audio file (OpenAI, Groq, a local HTTP server, or offline via whisper-cpp) |
| `ocr_image` | Extract text from an image via tesseract (local, no API key) |
| `load_document` | Load and extract text from a document |

## See also

- [Agent mode](/agent/) - overview and starting the REPL
- [REPL slash commands](/agent/commands) - full command reference
- [Shell execution](/agent/shell) - bash_exec keyboard shortcuts and rtk
- [Tool execution monitoring](/agent/monitoring) - status lines and stall detection
- [Agent registries](/agent/registries) - task_*/team_*/cron_* tools for multi-agent coordination
