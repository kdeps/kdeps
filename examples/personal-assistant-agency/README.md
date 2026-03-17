# Personal Assistant Agency

An OpenClaw-style personal AI assistant built as a kdeps agency.  All five
OpenClaw components — Gateway, Brain, Memory, Skills, and Heartbeat — map
directly onto kdeps agents and resource types.

## Architecture

```
                    ┌──────────────────────────────────┐
                    │  pa-gateway  (port 17200)         │
                    │                                   │
POST /api/v1/chat ──►  normalise ──► call-brain ──────────────────────────┐
POST /webhook/telegram  (exprs)                                            │
POST /webhook/slack                                                        │
                    │               call-heartbeat ─────────────────────┐  │
POST /api/v1/heartbeat ─────────────►  (heartbeat route)                │  │
                    └──────────────────────────────────┘                │  │
                                                                         │  │
           ┌─────────────────────────────────────────────────────────────┘  │
           │                                                                 │
           ▼                                                                 │
  ┌─────────────────────────────────────────────────────────────────────┐    │
  │  pa-brain                                                           │    │
  │                                                                     │    │
  │  init-db ──► recall ──► think (LLM + tools) ──► save-user-turn    │    │
  │                │                                        │           │    │
  │                │   ┌────────────────────────────────────┘           │    │
  │                │   ▼                                                │    │
  │                │  save-assistant-turn ──► brain-response           │    │
  │                │                                                     │    │
  │                │  Skills (called by LLM via ReAct):                 │    │
  │                │    web-search-tool   → DuckDuckGo API              │    │
  │                │    send-email-tool   → SMTP                        │    │
  │                │    run-command-tool  → sh                          │    │
  │                │                                                     │    │
  │  Memory: SQLite /tmp/pa-memory/conversations.db                   │    │
  └─────────────────────────────────────────────────────────────────────┘    │
                                                                              │
           ┌───────────────────────────────────────────────────────────────────┘
           │
           ▼
  ┌─────────────────────────────────────────────────────────────────────┐
  │  pa-heartbeat                                                       │
  │                                                                     │
  │  load-tasks ──► system-context ──► assess (LLM) ──► act ──► resp  │
  │                                                                     │
  │  Task list: /tmp/pa-memory/heartbeat.md                           │
  └─────────────────────────────────────────────────────────────────────┘
```

## OpenClaw component mapping

| OpenClaw    | kdeps agent / resource                             |
|-------------|-----------------------------------------------------|
| **Gateway** | `pa-gateway` — multi-route API server normalising all channels |
| **Brain**   | `pa-brain` `think` resource — LLM with ReAct tool-calling |
| **Memory**  | `pa-brain` SQLite + `recall` / `save-*-turn` resources |
| **Skills**  | `web-search-tool`, `send-email-tool`, `run-command-tool` |
| **Heartbeat** | `pa-heartbeat` — triggered on demand or by cron |

## Setup

### 1. Clone or run locally

```bash
kdeps run examples/personal-assistant-agency/agency.yaml
```

### 2. Environment variables (for email skill)

```bash
export SMTP_HOST=smtp.gmail.com
export SMTP_USER=you@example.com
export SMTP_PASS=your-app-password
```

### 3. Customise heartbeat tasks

On first run, `/tmp/pa-memory/heartbeat.md` is created automatically.
Edit it to add or remove recurring tasks:

```markdown
# Heartbeat Tasks
- [ ] Summarise unread emails in INBOX
- [ ] Report disk usage
- [ ] Check for package updates
```

### 4. Swap the model

Change `llama3.2:3b` to any Ollama-compatible model in all three
`workflow.yaml` files, or switch `backend: ollama` to `backend: openai`
and set `OPENAI_API_KEY` to use GPT-4o / Claude.

## Usage

### Direct API (REST)

```bash
# Chat with the assistant
curl -X POST http://localhost:17200/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"message": "What is the disk usage on this machine?", "session_id": "alice"}'

# Continue the conversation (memory is per session_id)
curl -X POST http://localhost:17200/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"message": "Now search the web for the latest Python release.", "session_id": "alice"}'

# Send an email via the assistant
curl -X POST http://localhost:17200/api/v1/chat \
  -H 'Content-Type: application/json' \
  -d '{"message": "Email bob@example.com and tell him the meeting is at 3pm.", "session_id": "alice"}'
```

### Telegram webhook

Register your bot webhook at:
```
https://api.telegram.org/bot<TOKEN>/setWebhook?url=https://<your-host>/webhook/telegram
```

The gateway normalises Telegram's `message.text` and `message.from.id`
automatically — no extra configuration needed.

### Slack Events API

Point your Slack app's **Event Subscriptions** Request URL to:
```
https://<your-host>/webhook/slack
```

Subscribe to `message.im` or `message.channels` events.  The gateway
extracts `event.text` and `event.user` automatically.

### Heartbeat (proactive checks)

Trigger a heartbeat manually:
```bash
curl -X POST http://localhost:17200/api/v1/heartbeat
```

Or wire it to a cron job / cloud scheduler to run every 30 minutes:
```cron
*/30 * * * * curl -s -X POST http://localhost:17200/api/v1/heartbeat
```

## Example responses

### Chat response
```json
{
  "success": true,
  "data": "The disk at / shows 14 GB used of 50 GB (28%). Plenty of space available."
}
```

### Heartbeat response
```json
{
  "success": true,
  "data": {
    "assessment": {
      "needs_action": [
        { "task": "Report disk usage", "reason": "Routine check", "action": "df -h /" }
      ],
      "deferred": ["Summarise unread emails"],
      "summary": "Disk check completed; email summary deferred until IMAP is configured."
    },
    "actions_taken": "=== Heartbeat Actions — 2026-03-17T10:30:00Z ===\n..."
  }
}
```

## Agents

| Agent | Role | API server |
|---|---|---|
| `pa-gateway` | Multi-channel normaliser, dispatches to brain or heartbeat | ✅ `portNum: 17200` |
| `pa-brain` | ReAct LLM + SQLite memory + web/email/shell skills | ❌ internal |
| `pa-heartbeat` | Proactive task reviewer and executor | ❌ internal |

## Packaging

```bash
# Package as a portable .kagency archive
kdeps package examples/personal-assistant-agency/

# Run the packed agency
kdeps run personal-assistant-1.0.0.kagency

# Build a Docker image
kdeps build personal-assistant-1.0.0.kagency

# Export as a self-booting ISO
kdeps export iso personal-assistant-1.0.0.kagency
```
