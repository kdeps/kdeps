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

That's it - models run as local llamafiles (the default `file` backend) and
are downloaded to `~/.kdeps/models/` automatically on first use.

### Run Locally

```bash
# Run from the project root
kdeps run examples/chatgpt-clone
```

This starts both servers:
- **API Server** on http://localhost:16395 (handles chat requests)
- **Web Interface** on http://localhost:16395 (serves the frontend)

Open http://localhost:16395 in your browser to use the chat interface.

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
| llama3.2:1b | 1B | Meta's smallest and fastest model (~1.1 GB) |
| llama3.2:1b-q6 | 1B | Higher-quality quantization (~1.5 GB) |
| llama3.2:3b | 3B | Balanced speed and quality (~2.2 GB) |
| llama3.1:8b | 8B | Stronger reasoning (~5.2 GB) |
| qwen3.5:0.8b | 0.8B | Tiny, fast, multilingual (~1.3 GB) |
| qwen3.5:2b | 2B | Multilingual support (~3 GB) |

Model ids are llamafile registry aliases. List all known aliases (or refresh
the registry from HuggingFace):
```bash
kdeps llamafile list
kdeps llamafile update
```

To add more models, add their alias to `resources/models.yaml` and the model
selector in `public/index.html`.

## Docker Deployment

Build a Docker image with pre-loaded models:

```bash
kdeps build workflow.yaml
```

The resulting image will include:
- Pre-baked llamafile models (no LLM server install)
- KDeps runtime
- Static web interface

Run the container:
```bash
docker run -p 16395:16395 -p 16395:16395 chatgpt-clone:latest
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
2. Verify the API: `curl http://localhost:16395/api/v1/models`

### "Model not found"

Check the model id is a known llamafile alias:
```bash
kdeps llamafile list
```

### Slow responses

- Use a smaller model (1B or 0.8B)
- Ensure you have enough RAM (8GB+ recommended)
- The first request per model downloads the llamafile - subsequent requests use the cache

## License

MIT License - See [LICENSE](../../LICENSE) for details.
