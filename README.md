# kdeps: AI agents in YAML.

Orchestrate LLMs, databases, and APIs without glue or legacy code.

## 1. Install
```bash
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/main/install.sh | sh
```

## 2. Run
```bash
kdeps new my-agent
kdeps run workflow.yaml --dev
```

## 3. Example
```yaml
run:
  chat:
    model: llama3.2:1b
    prompt: "Summarize: {{ get('text') }}"
```

[Documentation](https://kdeps.com) | [Visual Editor](https://kdeps.io)
