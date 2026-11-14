# Judge panel

The judge panel is the eval. Workflow chats take the same panel as a [`judges:` block](/workflow/resources/llm#judges). It reviews a turn's answer on its own. Roster generation and each judge use the [subagent channel](/agent/subagent). That is the session model unless you point it at another model.

Goal enforcement checks that the loop continues to move. It does not check if
the final answer is correct. The panel is an independent review of that answer,
after the turn makes it. One or more reviewer personas, each with real tool
access, check the output. They can send it back for revision before it gets to
you.

```text
answer -> [judge 1] -> [judge 2] -> [judge 3] -> all approved? -> answer
              |             |             |
              v (tools)     v (tools)     v (tools)
          read files    run tests    search the web
                    any rejected -> revise -> re-judge
```

**A judge is a full agent, not an automatic approval.** Each judge runs its own
temporary tool-calling loop (the same round loop as the main turn). Then the
judge can read a file, run a test, or search the web to verify a claim before
its decision. It does not only read the answer again and guess. A judge
completes its review with one call to `judge_verdict{approved, feedback}`. A
judge cannot approve or reject with only prose.

**The panel, not one judge.** All judges in the roster review the *same* output
at the same time. If a judge rejects it, kdeps combines the feedback of the
judges that reject it into a revision directive. The loop makes a new answer,
and the full panel reviews it again. This occurs a maximum of
`JudgeMaxIterations` times (default 2).

When the budget is empty, kdeps returns the last answer. A judge panel that
never approves must not stop the turn permanently. If a revision is empty (or
has the "model produced nothing" notice), kdeps ignores it and keeps the
previous answer. For this reason, a failed revision never replaces a real answer with a
blank or a standard message.

**kdeps makes the roster at the same time as the turn.** Auto-judges make one
subagent call (the same synthetic JSON call that the loop uses for compaction).
It starts with the turn, on the prompt as you typed it. It runs at the same
time as prompt refinement, the planning council, and the turn. The REPL prints
the panel when the answer is ready, immediately before the judges review it:

```
[judges] panel generated - 2 reviewer(s)
  - correctness: every factual claim must be verifiable from the codebase
  - security: no secrets, credentials, or unsafe shell input
```

The roster call can return nothing, because the model is not available or kdeps
cannot parse its reply. In this condition, the REPL tells you. It does not skip
the review silently:

```
[judges] no panel this turn - roster generation returned nothing
```

`/judges` (or `/judges list`) shows that last auto-generated panel. An
explicit roster from `/judges add` still wins at review time and is not
re-printed every turn.

**You can see the review as it occurs.** A judge is a full tool-calling loop.
For this reason, a review can take as long as a real turn. The REPL prints `[judge]` status
lines, so the prompt is never silent:

```
[judge] reviewing output - correctness, security
[judge] approved by correctness, security
```

or, on rejection:

```
[judge] reviewing output - correctness, security
[judge] revision requested (1/2): security: uses a shell command with unescaped input
[judge] reviewing output - correctness, security
[judge] approved by correctness, security
```

**Configuring the roster.**

```go
cfg := agent.Config{
    // Explicit roster - always wins when set.
    Judges: []agent.JudgeSpec{
        {Name: "correctness", Criteria: "every factual claim must be verifiable from the codebase"},
        {Name: "security", Criteria: "no secrets, credentials, or unsafe shell input"},
    },
    // Or: generate a roster (1-3 judges) per turn instead of hand-configuring one.
    AutoJudges: true,
    JudgeMaxIterations: 2, // revise-and-rejudge budget
}
```

The `allowedTools:` list of a judge sets the registered tools that it can call.
An empty list means only `judge_verdict`. The judge reads the candidate text
and gives one decision. If the judge must check claims in the workspace, give
the tools.

With `AutoJudges`, one LLM call for each turn makes the roster. This call uses
the same synthetic-workflow JSON-mode pattern as compaction. kdeps skips it
for simple chat, as compaction does. A failure in the panel becomes an
approval, so it does not stop the turn. Examples of failures are a judge
that gives an error, has a timeout, or never calls `judge_verdict`.

**On until you turn it off.** The REPL and the desktop agent loop start with the panel on. If a settings file does not set `auto_judges`, the panel stays on. `/judges off` (or `/judges clear`) removes the roster and writes `auto_judges: false`. `/judges auto on` turns it on again. A roster that you configure (`/judges add` / `remove`) is only for the session and has priority over the generated roster.

Library callers get no panel unless they set `Config.Judges` or `Config.AutoJudges`. When the agent calls a workflow or an agency as a tool, the panel reviews that turn's final answer. An agency started as its own agent loop gets the same default. `kdeps run` does not.

**Graders** are part of a judge. They run on the answer first, in the process, with no model call. If a grader fails, it rejects the answer, and that judge does not start an LLM review. If the graders pass and the judge also has criteria, the judge uses `judge_verdict`. A judge with graders and no criteria never calls the model.

The auto-generated roster stays criteria-only. Kinds: `exact`, `contains`, `not_contains`, `regex`, `json` (valid JSON, or equal to a JSON pattern), `expr` (`output` and `answer` are the candidate text). `/judges` prints them under that judge. `/judges off` drops them with the roster.

**From the REPL**, use `/judges` to manage the same roster. You do not need a
restart:

```
/judges                                        # show the current roster / auto-judges state
/judges auto off                               # turn the panel off and store it
/judges add correctness every claim must cite a source   # or: hand-configure an explicit roster instead
/judges grade correctness contains workflow.yaml         # attach a grader to that judge
/judges remove correctness                     # drop it again
/judges clear                                  # disable the panel entirely
```

## Judge-panel presets

A preset can carry a panel. Applying it replaces the session's judges and changes no limits. `design-review` ships with four reviewers for a design document or plan: `scope`, `clarity`, `failure-modes` and `readiness`, each with `read_file`, `list_files` and `search_local`.

```text
/harness preset design-review    # Applied preset "design-review": ..., judge panel of 4 for this session
```

Write your own as `~/.kdeps/presets/<name>.yaml`:

```yaml
name: api-review
description: Two reviewers for an API change
judges:
  - name: compatibility
    criteria: >-                  # the rubric folded into the judge's system prompt
      No existing route, field or status code changes meaning.
    allowedTools: [read_file, search_local]   # tools it may call to check the answer; empty = judge_verdict only
    maxRounds: 10                 # its tool rounds; 0 = the judge-max-rounds event
  - name: errors
    criteria: Every new failure returns a documented status and message.
    graders:                      # checks without a model call; a failure rejects
      - kind: not_contains
        pattern: TODO
```

The panel lasts for the session, like `/judges add`.

## See also

- [Agent mode](/agent/) - overview and starting the REPL
- Goal-directed execution - the task cursor a judge's revision request feeds back into
- [REPL slash commands](/agent/commands) - full command reference
