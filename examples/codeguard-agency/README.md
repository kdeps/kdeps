# CodeGuard Agency

AI-powered code review agency that runs security and quality analysis **in parallel**,
then synthesises a single prioritised report.

## Architecture

```
POST /api/v1/review
        │
        ▼
  ┌─────────────┐
  │ code-intake │  ← entry point (portNum 17100)
  └──────┬──────┘
         │ fans out in parallel
    ┌────┴────┐
    ▼         ▼
┌────────┐ ┌─────────┐
│security│ │ quality │  ← both run simultaneously
└────┬───┘ └────┬────┘
     └────┬─────┘
          ▼
     ┌────────┐
     │ report │  ← synthesises both findings
     └────────┘
          │
          ▼
    JSON review report
```

## Request

```bash
curl -X POST https://<your-deployment>.kdeps.io/api/v1/review \
  -H 'Content-Type: application/json' \
  -d '{
    "code": "import subprocess\ndef run(cmd):\n    subprocess.call(cmd, shell=True)",
    "language": "python"
  }'
```

## Response

```json
{
  "overall_score": 3,
  "risk_level": "critical",
  "executive_summary": "The code contains a critical shell-injection vulnerability ...",
  "top_priorities": [
    { "priority": 1, "title": "Remove shell=True", "rationale": "..." },
    { "priority": 2, "title": "Validate command input", "rationale": "..." },
    { "priority": 3, "title": "Add error handling", "rationale": "..." }
  ],
  "security": { "severity": "critical", "findings": [...], "summary": "..." },
  "quality":  { "score": 4, "suggestions": [...], "summary": "..." },
  "next_steps": ["Parameterise command args", "Add allow-list validation", "..."],
  "estimated_fix_effort": "small"
}
```

## Agents

| Agent | Role | API server |
|---|---|---|
| `code-intake` | Entry point — validates, fans out, collects results | ✅ `portNum: 17100` |
| `code-security` | OWASP Top 10, injection, secrets, crypto checks | ❌ internal only |
| `code-quality` | Complexity, DRY, naming, error handling, test coverage | ❌ internal only |
| `code-report` | Synthesises both reviews into a prioritised report | ❌ internal only |

## Model

All agents use `llama3.2:3b` (local Ollama).  Swap to any Ollama-compatible model
in the `agentSettings.models` list of each workflow.
