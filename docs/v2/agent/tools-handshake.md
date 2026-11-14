# Session-integrity handshake

The handshake checks that the model makes a real tool call after the context changes. Permission modes are on [Built-in tools](/agent/tools).

It is off by default. `/handshake on` enables it and verifies the current model before your next prompt. `/handshake` alone reports the state; the setting persists across sessions.

After a model switch, a resumed session, or a compaction/fold that rewrites the context, kdeps does not know if the model can call tools. The model can write "I called X, it returned Y" as plain text, and not make a call. The handshake finds this problem:

```d2
direction: right
change: "model switch, resume,\ncompaction or fold" {shape: oval}
warm: "warm-up questions"
evidence: "bash_exec pwd\n(evidence)"
challenge: "call session_handshake\nwith a 4-digit code"
ok: "real call,\nright code?" {shape: diamond}
prompt: "your prompt is sent" {shape: oval}
change -> warm -> evidence -> challenge -> ok
ok -> prompt: yes
ok -> challenge: "no: nudge, retry"
```

kdeps sends the challenge as a user-turn message. It does not use a system prompt, because a backend can treat a system prompt as background. kdeps checks that a real `session_handshake` call with the right code reached the tool registry, not text that looks like one.

## The block to copy

The directive says kdeps parses a matched `<invoke>` block out of the message at runtime; it is not the model's code interpreter. It shows the syntax with the code filled in:

```
<invoke name="session_handshake">
<parameter name="code">4821</parameter>
</invoke>
```

A `session_handshake` call without `code` is a miss and returns an error, not the success ack.

## Warm-up

Before the challenge, kdeps asks two unchecked warm-up questions: "how do you invoke a kdeps tool?" and "how many tools do you have?". The warm-up system message appends a names-only `<available_tools>` catalog, which the shared preamble refers to. Every attempt stays in one conversation, so a retry sees its earlier miss.

## Evidence step

After the warm-up, kdeps asks the model to call `bash_exec` with `pwd` (skipped if `bash_exec` is not registered). A real result moves a model that believes it is in a sandbox better than more text does. On success, kdeps adds "that came from a real shell, not a sandbox".

The step retries up to 3 times. The nudge explains that kdeps has no native tool channel, so the `<invoke>` block must be the literal text. Then it gives up quietly; the challenge is the real gate.

## Grounding message

The exchange carries the same tool-use guidance as every normal turn. A bare prompt plus a tool schema reads as sparse, and models then decide the tool "isn't really available". For m365, it also carries the sandbox reinforcement normal turns get.

The grounding names `session_handshake` as a registered tool in one line, not the full catalog. A full catalog per round inflated a small local model's prompt to tens of thousands of tokens.

## Retries

- **Wrong code:** retried with a corrective nudge, with no attempt cap. Proceeding unverified is the failure the handshake prevents.
- **No tool call in 5 sequential tries:** prints a `[handshake]` warning that the check did not complete, and lets the turn continue.

## The nudge

Each retry reshows the `<invoke>` syntax, says how many attempts missed, and keeps the same code. After 2 sequential misses, it escalates. It shows the earlier real `bash_exec` call of the model as proof that the model can call tools. It also forbids narration, code fences, and all text around the block. Some small local models wrote "I will call..." around an empty code fence, and not the tag.

## Code fences

Every directive says the block goes outside any markdown code fence. A correct `<invoke>` block inside a fence is still accepted for `session_handshake` in this round. Elsewhere, fenced code is never parsed as a call, so documentation examples are not misread.

## What the retry keeps

The handshake conversation keeps the first exchange plus the 3 most recent. With no limit, the prompt of a small local model increased from about 3k to more than 50k tokens, and the answers became worse. The first exchange stays because m365 uses a fingerprint of the first message. With it, m365 finds if a request continues a conversation. Without the first exchange, each retry starts a new conversation.

## No session trace

The exchange never touches session history, so nothing appears in `/prompt` or a saved transcript. `KDEPS_DEBUG=1` logs each attempt.
