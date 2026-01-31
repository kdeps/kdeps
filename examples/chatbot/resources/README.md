# LLM Backend Examples

This directory contains examples of different LLM backend configurations.

## Files

- **llm.yaml** - Basic Ollama configuration (default backend)
- **llm-backend-example.yaml** - Shows backend selection with context length
- **response.yaml** - API response resource example

## Docker Build Configuration

To use Ollama in Docker builds, set `installOllama: true` in your workflow:

```yaml
agentSettings:
  installOllama: true  # Install Ollama for local LLM support
  models:
    - llama3.2:1b      # Pre-download models
```

For cloud-only workflows:
```yaml
agentSettings:
  installOllama: false  # Skip Ollama installation
```

## Backend Options

### Local Backend

#### Ollama (Default)
```yaml
chat:
  backend: ollama  # or omit for default
  baseUrl: http://localhost:11434  # Optional
  contextLength: 4096  # 4k tokens (default)
```

### Cloud Backends

#### OpenAI
```yaml
chat:
  backend: openai
  apiKey: "{{ get('OPENAI_API_KEY', 'env') }}"
  model: gpt-4o
  contextLength: 8192  # 8k tokens
```

#### Anthropic (Claude)
```yaml
chat:
  backend: anthropic
  apiKey: "{{ get('ANTHROPIC_API_KEY', 'env') }}"
  model: claude-3-5-sonnet-20241022
  contextLength: 32768  # 32k tokens
```

#### Google (Gemini)
```yaml
chat:
  backend: google
  apiKey: "{{ get('GOOGLE_API_KEY', 'env') }}"
  model: gemini-1.5-pro
  contextLength: 65536  # 64k tokens
```

#### Groq
```yaml
chat:
  backend: groq
  apiKey: "{{ get('GROQ_API_KEY', 'env') }}"
  model: llama-3.1-70b-versatile
  contextLength: 131072  # 128k tokens
```

#### Together AI
```yaml
chat:
  backend: together
  apiKey: "{{ get('TOGETHER_API_KEY', 'env') }}"
  model: meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo
```

#### Cohere
```yaml
chat:
  backend: cohere
  apiKey: "{{ get('COHERE_API_KEY', 'env') }}"
  model: command-r-plus
```

#### Mistral AI
```yaml
chat:
  backend: mistral
  apiKey: "{{ get('MISTRAL_API_KEY', 'env') }}"
  model: mistral-large-latest
```

#### Perplexity
```yaml
chat:
  backend: perplexity
  apiKey: "{{ get('PERPLEXITY_API_KEY', 'env') }}"
  model: llama-3.1-sonar-large-128k-online
```

#### DeepSeek
```yaml
chat:
  backend: deepseek
  apiKey: "{{ get('DEEPSEEK_API_KEY', 'env') }}"
  model: deepseek-chat
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
import "github.com/kdeps/kdeps/v2/pkg/executor/llm"

manager := llm.NewModelManager(logger)

// Download a model
err := manager.DownloadModel("ollama", "llama3.2:1b")

// Serve a model
err := manager.ServeModel("ollama", "llama3.2:1b", "localhost", 11434)
```
