# ChatGPT Clone

A ChatGPT-like web interface powered by open-source LLMs running locally via KDeps.

## Features

- Modern, dark-themed UI similar to ChatGPT
- Multiple model selection (Llama, Mistral, Phi, Gemma, Qwen)
- Persistent chat history (saved in browser localStorage)
- Real-time loading indicators
- Markdown-like formatting (code blocks, bold, italic)
- Fully local - your data never leaves your machine

## Quick Start

### Prerequisites

1. [Install KDeps](https://github.com/kdeps/kdeps#installation)
2. [Install Ollama](https://ollama.ai/) and ensure it's running
3. Pull at least one model:
   ```bash
   ollama pull llama3.2:1b
   ```

### Run Locally

```bash
# Run from the project root
kdeps run examples/chatgpt-clone
```

This starts both servers:
- **API Server** on http://localhost:3000 (handles chat requests)
- **Web Interface** on http://localhost:8080 (serves the frontend)

Open http://localhost:8080 in your browser to use the chat interface.

## Architecture

```
examples/chatgpt-clone/
├── workflow.yaml           # Main workflow configuration
├── resources/
│   ├── llm.yaml           # LLM chat handler
│   ├── models.yaml        # Available models endpoint
│   └── response.yaml      # Response formatting
└── public/
    ├── index.html         # Web interface
    ├── styles.css         # Dark theme styles
    └── app.js             # Frontend JavaScript
```

## API Endpoints

### POST /api/v1/chat

Send a chat message to the LLM.

**Request:**
```json
{
  "message": "What is AI?",
  "model": "llama3.2:1b"
}
```

**Response:**
```json
{
  "success": true,
  "response": {
    "message": "AI (Artificial Intelligence) is...",
    "model": "llama3.2:1b",
    "query": "What is AI?"
  }
}
```

### GET /api/v1/models

Get list of available models.

**Response:**
```json
{
  "success": true,
  "response": {
    "models": [
      {"id": "llama3.2:1b", "name": "Llama 3.2 (1B)", "description": "..."},
      ...
    ]
  }
}
```

## Available Models

| Model | Size | Description |
|-------|------|-------------|
| llama3.2:1b | 1B | Meta's smallest and fastest model |
| llama3.2:3b | 3B | Balanced speed and quality |
| mistral:7b | 7B | Excellent performance |
| phi3:3.8b | 3.8B | Microsoft's efficient model |
| gemma2:2b | 2B | Google's lightweight model |
| qwen2.5:3b | 3B | Multilingual support |

To add more models, pull them with Ollama:
```bash
ollama pull codellama:7b
ollama pull deepseek-coder:6.7b
```

Then add them to the model selector in `public/index.html`.

## Docker Deployment

Build a Docker image with pre-loaded models:

```bash
kdeps build workflow.yaml
```

The resulting image will include:
- Ollama with pre-loaded models
- KDeps runtime
- Static web interface

Run the container:
```bash
docker run -p 3000:3000 -p 8080:8080 chatgpt-clone:latest
```

## Customization

### Change Default Model

Edit `resources/llm.yaml`:
```yaml
expr:
  - set('selectedModel', default(get('model'), 'mistral:7b'))
```

### Add System Prompt

Edit `resources/llm.yaml`:
```yaml
expr:
  - set('systemPrompt', 'You are a coding assistant. Always provide examples.')
```

### Change UI Theme

Edit `public/styles.css` to customize colors, fonts, and layout.

## Troubleshooting

### "Unable to connect to the server"

1. Check if KDeps is running: `kdeps run workflow.yaml`
2. Check if Ollama is running: `ollama list`
3. Verify the API: `curl http://localhost:3000/api/v1/models`

### "Model not found"

Pull the model with Ollama:
```bash
ollama pull llama3.2:1b
```

### Slow responses

- Use a smaller model (1B or 3B)
- Ensure you have enough RAM (8GB+ recommended)
- For GPU acceleration, install Ollama with CUDA support

## License

MIT License - See [LICENSE](../../LICENSE) for details.
