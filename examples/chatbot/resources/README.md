# LLM Backend Examples

This directory contains examples of different LLM backend configurations.

## Files

- **llm.yaml** - Basic Ollama configuration (default backend)
- **llm-backend-example.yaml** - Shows backend selection with context length
- **response.yaml** - API response resource example

## Docker Build Configuration

To use Ollama in Docker builds, set `installOllama: true` in your workflow:

```yaml
deploy:
  runtime:
    installOllama: true  # Install Ollama for local LLM support
    models:
      - llama3.2:1b      # Pre-download models
```

For cloud-only workflows:
```yaml
deploy:
  runtime:
    installOllama: false  # Skip Ollama installation
```

## Backend Options

### Local Backend

#### Ollama (Default)
```yaml
chat:
  baseUrl: http://localhost:11434  # Optional
  model:
    backend: ollama  # or omit for default
  context:
    contextLength: 4096  # 4k tokens (default)
```

### Cloud Backends

#### OpenAI
```yaml
chat:
  apiKey: "{{ get('OPENAI_API_KEY', 'env') }}"
  model:
    backend: openai
    name: gpt-4o
  context:
    contextLength: 8192  # 8k tokens
```

#### Anthropic (Claude)
```yaml
chat:
  apiKey: "{{ get('ANTHROPIC_API_KEY', 'env') }}"
  model:
    backend: anthropic
    name: claude-3-5-sonnet-20241022
  context:
    contextLength: 32768  # 32k tokens
```

#### Google (Gemini)
```yaml
chat:
  apiKey: "{{ get('GOOGLE_API_KEY', 'env') }}"
  model:
    backend: google
    name: gemini-1.5-pro
  context:
    contextLength: 65536  # 64k tokens
```

#### Groq
```yaml
chat:
  apiKey: "{{ get('GROQ_API_KEY', 'env') }}"
  model:
    backend: groq
    name: llama-3.1-70b-versatile
  context:
    contextLength: 131072  # 128k tokens
```

#### Together AI
```yaml
chat:
  apiKey: "{{ get('TOGETHER_API_KEY', 'env') }}"
  model:
    backend: together
    name: meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo
```

#### Cohere
```yaml
chat:
  apiKey: "{{ get('COHERE_API_KEY', 'env') }}"
  model:
    backend: cohere
    name: command-r-plus
```

#### Mistral AI
```yaml
chat:
  apiKey: "{{ get('MISTRAL_API_KEY', 'env') }}"
  model:
    backend: mistral
    name: mistral-large-latest
```

#### Perplexity
```yaml
chat:
  apiKey: "{{ get('PERPLEXITY_API_KEY', 'env') }}"
  model:
    backend: perplexity
    name: llama-3.1-sonar-large-128k-online
```

#### DeepSeek
```yaml
chat:
  apiKey: "{{ get('DEEPSEEK_API_KEY', 'env') }}"
  model:
    backend: deepseek
    name: deepseek-chat
```

## Context Length Options

- 4096 (4k) - Default
- 8192 (8k)
- 16384 (16k)
- 32768 (32k)
- 65536 (64k)
- 131072 (128k)
- 262144 (256k)

## Automatic Model Management

When using the LLM executor with Ollama, models are automatically downloaded and served based on the backend configuration. This happens transparently when the workflow is executed.

### Manual Model Management

You can also manage models manually:

```go
import "github.com/kdeps/kdeps/v3/pkg/executor/llm"

manager := llm.NewModelManager(logger)

// Download a model
err := manager.DownloadModel("ollama", "llama3.2:1b")

// Serve a model
err := manager.ServeModel("ollama", "llama3.2:1b", "localhost", 11434)
```
