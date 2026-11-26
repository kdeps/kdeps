# Example: stateless bot (stdin/stdout)

A workflow-mode (`kdeps run`) one-shot bot: it reads a message from stdin (or an env var), calls an LLM, and writes the reply to stdout. Use it in cron jobs, CI pipelines or custom integrations.

Send one message on stdin:

```bash
echo '{"message":"What is 2+2?","chatId":"42","userId":"u1","platform":"custom"}' \
  | kdeps run workflow.yaml
```

## Read next

- [Full walkthrough](/reference/examples/stateless-bot/) - every step, file and command
- [Input sources](../../workflow/input-sources) - Full platform config reference
