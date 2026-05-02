# Why kdeps?

## The AI appliance model

Chat AIs (Claude, Gemini, ChatGPT) and their CLI and MCP extensions are tools you operate. You prompt them, they respond, the session ends. They are powerful, but they are not something you ship.

kdeps is an **AI appliance builder**. You define what the agent does, bundle it, and deploy it as a self-contained unit. It exposes an HTTP API, runs on a schedule, responds to bot messages, or processes files - without a human in the loop, without a chat session, without anyone prompting it.

## Chat AI vs. kdeps

| | Chat AI + MCP | kdeps |
|---|---|---|
| Who drives it | You | Nobody - it runs on its own |
| Deployed as | A chat session | A Docker image, edge ISO, or binary |
| Logic lives in | Prompts and MCP config | YAML code - versioned, reviewed, tested |
| Other systems call it | No | Yes - it is an HTTP API |
| Model | One provider | Any LLM, swappable per resource |
| Ships to production | No | Yes |

## What about MCP?

MCP (Model Context Protocol) is a good protocol for giving a chat AI access to your tools. It is convenient for interactive use. But it still requires a human driving the conversation, and it produces no deployable artifact.

kdeps is for when your product needs AI capabilities baked in - not bolted on through a chat interface.

## Strictness as a feature

Chat AIs are deliberately open-ended - you can ask anything, and the model decides what to do. That flexibility is great for exploration and terrible for production. kdeps is the opposite: inputs are declared, outputs are typed, dependencies are explicit, and validations are enforced before any LLM is called. If something is wrong, it fails fast with a clear error rather than hallucinating a plausible-looking response.

Think of it like duck-typing vs. static types. A chat AI will try to do something with whatever you give it. kdeps requires you to be precise about what goes in and what comes out - and that precision is what makes the behavior reproducible, auditable, and safe to run unattended.

## Who it is for

- Developers shipping AI features into products (APIs, bots, pipelines)
- Teams that need agent logic in version control - reviewable and reproducible
- Engineers deploying to edge, Docker, or air-gapped environments
- Anyone who needs to run LLM workflows on a schedule or in response to events

## Who it is not for

- Interactive coding assistance - use Claude Code or Copilot
- One-off research or Q&A - use a chat interface
- No-code AI assistants - kdeps is infrastructure, not an end-user app
