# Why kdeps?

## The AI Appliance Model

Chat AIs (Claude, Gemini, ChatGPT) and their CLI and MCP extensions are tools you operate. You prompt them, they respond, the session ends. They are powerful, but they are not something you ship.

kdeps is an **AI appliance builder**. You define what the agent does, bundle it, and deploy it as a self-contained unit. It exposes an HTTP API, runs on a schedule, responds to bot messages, or processes files - without a human in the loop, without a chat session, without anyone prompting it.

## Coordinated Multi-Agent Systems

Single-agent workflows are often insufficient for complex business logic. kdeps allows you to orchestrate **Agencies** — collections of specialized agents that coordinate and delegate tasks.

- **Specialization**: Each agent can be locked to specific models and tools optimized for its task.
- **Coordination**: Agents communicate through a fully defined control flow, ensuring predictable interactions.
- **Auditable**: Every step of the multi-agent coordination is visible, version-controlled, and testable.

## Chat AI vs. kdeps

| | Chat AI + MCP | kdeps (Appliance) |
|---|---|---|
| **Who drives it** | You | The system (autonomous/event-driven) |
| **Deployed as** | A chat session | Docker, Edge ISO, or Binary |
| **Logic lives in** | Prompts and MCP config | YAML code - versioned, reviewed, tested |
| **Orchestration** | Model-driven | Fully defined control flow |
| **Multi-Agent** | Sequential prompts | Coordinated, specialized Agencies |
| **Ships to production** | No | Yes |

## Defined Control Flow

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
