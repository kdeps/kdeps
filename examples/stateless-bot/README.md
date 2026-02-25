# Stateless Bot Example

A one-shot bot workflow that reads a message from stdin (or environment variables), processes it with an LLM, and writes the reply to stdout. No long-running process — just pipe in a message and get a response.

## Features

- Single execution: start, process one message, exit
- Input via stdin JSON or environment variables
- Output via stdout — easy to compose with shell scripts, cron, CI

## Usage

### JSON stdin

```bash
echo '{"message":"What is the capital of France?","chatId":"42","userId":"u1","platform":"cli"}' \
  | kdeps run workflow.yaml
```

Output:

```
Paris is the capital of France.
```

### Environment Variables

```bash
export KDEPS_BOT_MESSAGE="Summarize the theory of relativity in one sentence."
export KDEPS_BOT_PLATFORM="cli"
kdeps run workflow.yaml
```

### From a Shell Script

```bash
#!/bin/bash
REPLY=$(echo '{"message":"Hello!"}' | kdeps run /path/to/stateless-bot/workflow.yaml)
echo "Bot replied: $REPLY"
```

### Cron Job (daily greeting)

```cron
0 9 * * * KDEPS_BOT_MESSAGE="Good morning! What's a fun fact for today?" \
          kdeps run /path/to/stateless-bot/workflow.yaml >> /var/log/bot-greetings.log
```

## Input Format

| Method | Format |
|--------|--------|
| stdin JSON | `{"message":"...","chatId":"...","userId":"...","platform":"..."}` |
| Env vars | `KDEPS_BOT_MESSAGE`, `KDEPS_BOT_CHAT_ID`, `KDEPS_BOT_USER_ID`, `KDEPS_BOT_PLATFORM` |

Only `message` (or `KDEPS_BOT_MESSAGE`) is required. All other fields are optional.

## Structure

```
stateless-bot/
├── workflow.yaml          # Stateless bot source config
└── resources/
    ├── llm.yaml           # LLM chat — processes input('message')
    └── reply.yaml         # Writes LLM reply to stdout
```

## See Also

- [Telegram Bot Example](../telegram-bot/) — Long-running polling bot
- [Input Sources Documentation](../../docs/v2/concepts/input-sources.md)
- [Bot Tutorial](../../docs/v2/tutorials/bot.md)
