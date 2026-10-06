# Session-integrity handshake

The handshake checks that the model makes a real tool call after the context changes. It is off until `/handshake on`. Permission modes stay on [Built-in tools](/agent/tools).

Off by default; enable with `/handshake on`, which verifies the current model immediately (before your very next prompt), not just after a future model change. Its purpose is to enforce that the model actually uses real kdeps tool calls, not fabricated text - right after a model switch, a resumed session, or a compaction/fold that rewrites the context, the model's tool-calling path against that new state is unproven, and it could fabricate a plausible "I called X, it returned Y" in plain text instead of making a real call. Before the next prompt is sent, kdeps runs a challenge/response round: it issues a random 4-digit code and instructs the model, as a normal user-turn message (not a system prompt a backend might deprioritize as background context), to call the internal `session_handshake` tool with it. kdeps verifies a real, correctly-valued tool call actually reached the registry - not text that merely looks like one.

## Retries

A miss where the model is at least engaging - calling the tool, just with the wrong code - is retried with a corrective nudge for as long as it takes; there is no attempt cap for that case, since giving up and proceeding unverified is the exact failure this exists to prevent. But a model that ignores the directive entirely, attempting no tool call at all across 5 consecutive tries, is not going to get there no matter how long kdeps waits - that case prints a `[handshake]` warning that the check is unverified and lets the turn proceed, rather than bricking the session on a model that cannot do this at all.

## The nudge

Each retry reshows the exact `<invoke>` syntax and states how many attempts have missed so far, framed as help toward getting it right rather than a rebuke - the same challenge code the whole way through (see below), never a moving target. After 2 consecutive misses the nudge escalates: live testing against a small local model showed it repeatedly writing sentences ABOUT calling the tool ("I will call...", "Here is the block:") wrapped around an empty code fence, instead of the literal tag text itself - the escalated nudge drops the friendly framing, anchors on the model's own earlier real tool call (the bash_exec evidence step) as proof it can already do this, and explicitly forbids narration, code fences, or any text before/after the block.

## What the retry keeps

The synthetic conversation itself (warm-up plus every retry) is also capped to the first exchange plus the 3 most recent - unbounded, it grows across warm-up, evidence, and every retry, and live testing showed that ballooning a small local model's prompt from ~3k to over 50k tokens after a few retries, making its answers worse rather than better as the context grew. The first exchange is kept rather than dropped for a second reason: the m365 backend decides whether a request continues an existing conversation or starts a new one by fingerprinting the first message in the array, so a plain sliding window that eventually evicts it would make m365 silently start a brand-new conversation - with a fresh generic system prompt and no memory of the evidence step - on every retry past the cap.

## Code fences

Every handshake directive (the challenge, the evidence step, and both retry nudges) now says explicitly that the block goes outside any markdown code fence, not just "nothing else around it" - fencing was the actual live failure mode, so it is named directly rather than left implied. Separately, as a safety net for when a model fences it anyway: a genuine, correctly-coded `<invoke>` block wrapped in a markdown fence out of habit is still recognized - the normal text-salvage path deliberately protects fenced code from being parsed as a real call (so a documentation example for a different tool is never misread as one), but that protection is narrowed for `session_handshake` specifically in this one round, where the risk it guards against does not apply.

## No session trace

The grounding message also names `session_handshake` as a real, registered tool (a compact one-line note, not the full tool catalog - a session with many registered tools sending its whole catalog on every handshake round was measured ballooning a small local model's prompt to tens of thousands of tokens for what should be a two-line request), so the model is not asked to call a tool with nothing else confirming it is real. The exchange itself never touches session history, so it leaves no trace in `/prompt` or a saved transcript. Set `KDEPS_DEBUG=1` to log each attempt. `/handshake` alone reports the current state; the setting persists across sessions.

## The block to copy

The directive tells the model that kdeps is a parser and an interpreter and will parse a matched `<invoke>` block out of the message at runtime and interpret it. Not the model's code interpreter. It shows the literal syntax to copy, with the actual code already filled in:

```
<invoke name="session_handshake">
<parameter name="code">4821</parameter>
</invoke>
```

## Grounding message

The exchange also carries a system message grounding the request: the same tool-use guidance every normal turn already includes, not a one-off line invented just for this. A bare user prompt plus a raw tool schema and nothing else reads as unusually sparse next to a normal turn's full preamble - even to models that make real tool calls fine on ordinary turns - leading them to reason the tool "wasn't really available" and refuse to call it.

## Warm-up

Before the invoke request itself, kdeps also asks two warm-up questions - "how do you invoke a kdeps tool?" and "how many tools do you have?" - answers not checked, but establishing an ongoing conversation about kdeps tools before asking the model to actually call one, rather than a cold open. Each attempt (warm-up included) stays in the same growing conversation, so a retry sees its own prior miss as context instead of a repeat cold start.

## Evidence step

After the warm-up and before the actual challenge, kdeps runs one more evidence step (skipped only if `bash_exec` is not registered): it asks the model to call `bash_exec` with `pwd` and see the real result. This is deliberately evidence, not another assertion - a model that has reasoned itself into believing it is running in a code-interpreter sandbox and "can't do anything" is more likely to be moved by a genuine result it just saw than by more text telling it otherwise. On success, the response is followed by a short reinforcement line ("that came from a real shell, not a sandbox") that stays in the conversation the challenge round builds on. This step is best-effort - it retries up to 3 times with a nudge that spells out the difference explicitly (a normal assistant session may trigger a tool call through its own native channel outside the visible text; kdeps gives no such channel, so the `<invoke>` block itself has to be the literal text sent) and then gives up quietly rather than blocking the turn, since the actual security gate is the challenge itself. For the m365 backend specifically - the one with the most aggressive observed "run code" / "Coding and executing" habit - the handshake's grounding message also carries the same sandbox-specific reinforcement normal turns get, not a bare version of it.
