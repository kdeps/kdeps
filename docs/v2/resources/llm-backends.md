# LLM Backends Reference

KDeps supports multiple LLM backends, both local and cloud-based. This reference covers all available backends and their configuration.

## Backend Overview

### Local Backend

| Backend | Name | Default URL | Description |
|---------|------|-------------|-------------|
| Ollama | `ollama` | `http://localhost:11434` | Local model serving (default) |

### Cloud Backends

| Backend | Name | API URL | Env Variable |
|---------|------|---------|--------------|
| OpenAI | `openai` | `https://api.openai.com` | `OPENAI_API_KEY` |
| Anthropic | `anthropic` | `https://api.anthropic.com` | `ANTHROPIC_API_KEY` |
| Google | `google` | `https://generativelanguage.googleapis.com` | `GOOGLE_API_KEY` |
| Cohere | `cohere` | `https://api.cohere.ai` | `COHERE_API_KEY` |
| Mistral | `mistral` | `https://api.mistral.ai` | `MISTRAL_API_KEY` |
| Together | `together` | `https://api.together.xyz` | `TOGETHER_API_KEY` |
| Perplexity | `perplexity` | `https://api.perplexity.ai` | `PERPLEXITY_API_KEY` |
| Groq | `groq` | `https://api.groq.com` | `GROQ_API_KEY` |
| DeepSeek | `deepseek` | `https://api.deepseek.com` | `DEEPSEEK_API_KEY` |

## Local Backend

### Ollama (Default)

Ollama is the default backend for local model serving.

```yaml
run:
  chat:
    backend: ollama
    model: llama3.2:1b
    prompt: "Hello, world!"
```

**Configuration:**

<div v-pre>

```yaml
# Custom Ollama URL
run:
  chat:
    backend: ollama
    baseUrl: "http://custom-ollama:11434"
    model: llama3.2:1b
    prompt: "{{ get('q') }}"
```

</div>

**Workflow-level Ollama configuration:**

```yaml
settings:
  agentSettings:
    models:
      - llama3.2:1b
      - nomic-embed-text
    ollamaUrl: "http://ollama:11434"
    installOllama: true  # Explicitly install Ollama in Docker image
```

**Docker Build:**

When building Docker images, Ollama is automatically installed if:
- A Chat resource uses the `ollama` backend (or no backend specified)
- Models are configured in `agentSettings.models`
- `installOllama: true` is explicitly set

You can also disable Ollama installation by setting `installOllama: false`.

## Cloud Backends

### OpenAI

<div v-pre>

```yaml
run:
  chat:
    backend: openai
    model: gpt-4o
    prompt: "{{ get('q') }}"
```

</div>

**With explicit API key:**

<div v-pre>

```yaml
run:
  chat:
    backend: openai
    apiKey: "{{ get('OPENAI_API_KEY', 'env') }}"
    model: gpt-4o
    prompt: "{{ get('q') }}"
```

</div>

**Available models:**

| Model | Description |
|-------|-------------|
| `gpt-4o` | Latest GPT-4 Omni |
| `gpt-4o-mini` | Smaller, faster GPT-4 |
| `gpt-4-turbo` | GPT-4 Turbo |
| `gpt-3.5-turbo` | Fast, cost-effective |

### Anthropic (Claude)

<div v-pre>

```yaml
run:
  chat:
    backend: anthropic
    model: claude-3-5-sonnet-20241022
    prompt: "{{ get('q') }}"
    contextLength: 4096
```

</div>

**Available models:**

| Model | Description |
|-------|-------------|
| `claude-3-5-sonnet-20241022` | Latest Claude 3.5 Sonnet |
| `claude-3-opus-20240229` | Most capable |
| `claude-3-sonnet-20240229` | Balanced |
| `claude-3-haiku-20240307` | Fast, efficient |

### Google (Gemini)

<div v-pre>

```yaml
run:
  chat:
    backend: google
    model: gemini-1.5-pro
    prompt: "{{ get('q') }}"
```

</div>

**Available models:**

| Model | Description |
|-------|-------------|
| `gemini-1.5-pro` | Latest Gemini Pro |
| `gemini-1.5-flash` | Fast inference |
| `gemini-pro` | Standard Gemini |

### Mistral

<div v-pre>

```yaml
run:
  chat:
    backend: mistral
    model: mistral-large-latest
    prompt: "{{ get('q') }}"
```

</div>

**Available models:**

| Model | Description |
|-------|-------------|
| `mistral-large-latest` | Most capable |
| `mistral-medium-latest` | Balanced |
| `mistral-small-latest` | Fast, efficient |
| `open-mistral-7b` | Open-source 7B |
| `open-mixtral-8x7b` | MoE model |

### Together AI

Access to many open-source models.

<div v-pre>

```yaml
run:
  chat:
    backend: together
    model: meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo
    prompt: "{{ get('q') }}"
```

</div>

**Popular models:**

| Model | Description |
|-------|-------------|
| `meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo` | Llama 3.1 70B |
| `meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo` | Llama 3.1 8B |
| `mistralai/Mixtral-8x7B-Instruct-v0.1` | Mixtral 8x7B |
| `Qwen/Qwen2-72B-Instruct` | Qwen2 72B |

### Groq

Ultra-fast inference with Groq hardware.

<div v-pre>

```yaml
run:
  chat:
    backend: groq
    model: llama-3.1-70b-versatile
    prompt: "{{ get('q') }}"
```

</div>

**Available models:**

| Model | Description |
|-------|-------------|
| `llama-3.1-70b-versatile` | Llama 3.1 70B |
| `llama-3.1-8b-instant` | Llama 3.1 8B (fastest) |
| `mixtral-8x7b-32768` | Mixtral with 32K context |
| `gemma2-9b-it` | Google Gemma 2 9B |

### Perplexity

Search-augmented LLM responses.

<div v-pre>

```yaml
run:
  chat:
    backend: perplexity
    model: llama-3.1-sonar-large-128k-online
    prompt: "{{ get('q') }}"
```

</div>

**Available models:**

| Model | Description |
|-------|-------------|
| `llama-3.1-sonar-large-128k-online` | Large with web search |
| `llama-3.1-sonar-small-128k-online` | Small with web search |
| `llama-3.1-sonar-large-128k-chat` | Large chat only |

### Cohere

<div v-pre>

```yaml
run:
  chat:
    backend: cohere
    model: command-r-plus
    prompt: "{{ get('q') }}"
```

</div>

**Available models:**

| Model | Description |
|-------|-------------|
| `command-r-plus` | Most capable |
| `command-r` | Fast and capable |
| `command` | Standard |
| `command-light` | Fast, efficient |

### DeepSeek

<div v-pre>

```yaml
run:
  chat:
    backend: deepseek
    model: deepseek-chat
    prompt: "{{ get('q') }}"
```

</div>

**Available models:**

| Model | Description |
|-------|-------------|
| `deepseek-chat` | General chat |
| `deepseek-coder` | Code generation |

## Backend Configuration

### Common Options

All backends support these options:

<div v-pre>

```yaml
run:
  chat:
    backend: openai          # Backend name
    baseUrl: "https://..."   # Custom base URL (optional)
    apiKey: "sk-..."         # API key (optional, falls back to env)
    model: gpt-4o            # Model name
    prompt: "{{ get('q') }}" # User prompt

    # Optional settings
    contextLength: 4096      # Max tokens
    jsonResponse: true       # Request JSON output
    jsonResponseKeys:        # Expected JSON keys
      - answer
      - confidence
```

</div>

### Custom Base URL

Override the default API URL:

<div v-pre>

```yaml
# Use Azure OpenAI
run:
  chat:
    backend: openai
    baseUrl: "https://my-resource.openai.azure.com/openai/deployments/my-deployment"
    apiKey: "{{ get('AZURE_OPENAI_KEY', 'env') }}"
    model: gpt-4o
    prompt: "{{ get('q') }}"

# Use OpenAI-compatible proxy
run:
  chat:
    backend: openai
    baseUrl: "https://my-proxy.example.com"
    model: gpt-4o
    prompt: "{{ get('q') }}"
```

</div>

### API Key Configuration

API keys can be provided in multiple ways:

**1. Environment variable (recommended):**

```bash
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."
```

**2. In resource configuration:**

<div v-pre>

```yaml
run:
  chat:
    backend: openai
    apiKey: "{{ get('OPENAI_API_KEY', 'env') }}"
    model: gpt-4o
```

</div>

**3. From session/request:**

<div v-pre>

```yaml
run:
  chat:
    backend: openai
    apiKey: "{{ get('apiKey', 'session') }}"
    model: gpt-4o
```

</div>

## Mixing Backends

Use different backends in the same workflow:

<div v-pre>

```yaml
# resources/fast-summary.yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: fastSummary
run:
  chat:
    backend: groq
    model: llama-3.1-8b-instant
    prompt: "Summarize: {{ get('q') }}"

---
# resources/deep-analysis.yaml
apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: deepAnalysis
  requires:
    - fastSummary
run:
  chat:
    backend: anthropic
    model: claude-3-5-sonnet-20241022
    prompt: |
      Based on this summary: {{ get('fastSummary') }}
      Provide detailed analysis.
```

</div>

## Feature Support

| Feature | Ollama | OpenAI | Anthropic | Google | Mistral | Groq |
|---------|--------|--------|-----------|--------|---------|------|
| JSON Response | Yes | Yes | Partial | Yes | Yes | Yes |
| Tools/Functions | Yes | Yes | No | Yes | Yes | Yes |
| Vision | Yes* | Yes | Yes | Yes | Yes | Yes |
| Streaming | No** | No** | No** | No** | No** | No** |

*Requires vision-capable model (e.g., `llama3.2-vision`)
**KDeps uses non-streaming for reliability

## Troubleshooting

### Connection Issues

**Local backend not responding:**

```yaml
# Verify the backend is running
run:
  httpClient:
    url: "http://localhost:11434/api/tags"
    method: GET
```

**API key errors:**

<div v-pre>

```yaml
# Debug: check if API key is set
run:
  expr:
    - set('hasKey', get('OPENAI_API_KEY', 'env') != nil)
    - set('keyLength', len(default(get('OPENAI_API_KEY', 'env'), '')))
```

</div>

### Model Not Found

Ensure the model is available:

**Ollama:**
```bash
ollama list  # See available models
ollama pull llama3.2:1b  # Download model
```

**Cloud backends:**
Check the provider's documentation for current model names.

### Rate Limiting

Handle rate limits with retry configuration:

<div v-pre>

```yaml
run:
  chat:
    backend: openai
    model: gpt-4o
    prompt: "{{ get('q') }}"
    retry:
      maxAttempts: 3
      initialDelay: "1s"
      maxDelay: "30s"
      backoffMultiplier: 2
```

</div>

## See Also

- [LLM Resource](llm) - Complete LLM resource documentation
- [Tools](../concepts/tools) - LLM function calling
- [Docker Deployment](../deployment/docker) - Deploying with local models
