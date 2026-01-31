# KDeps v2 Examples

This directory contains example workflows demonstrating KDeps v2 features.

## Examples

### 🤖 [Chatbot](./chatbot/)
Basic LLM chatbot with validation and error handling.
- YAML configuration
- HTTP API with CORS
- Preflight validation
- LLM integration with Ollama

**Run:**
```bash
cd chatbot && kdeps run workflow.yaml --dev
```

### 🌐 [HTTP Advanced](./http-advanced/)
Advanced HTTP client features with authentication, caching, and retries.
- Multiple authentication methods (Bearer, API Key, Basic, OAuth2)
- Intelligent retry logic with exponential backoff
- Response caching with TTL
- Proxy and TLS configuration
- Custom headers and timeout management

**Run:**
```bash
cd http-advanced && kdeps run workflow.yaml --dev
```

### 🐚 [Shell Exec](./shell-exec/)
Shell command execution with timeout and output capture.
- System command execution
- Output parsing and formatting
- Timeout handling
- Cross-platform shell support

**Run:**
```bash
cd shell-exec && kdeps run workflow.yaml --dev
```

### 🗄️ [SQL Advanced](./sql-advanced/)
Advanced SQL operations with batch processing and multiple result formats.
- Named database connections
- Batch operations (`paramsBatch`)
- CSV and JSON result formatting
- Transaction support
- Connection pooling

**Features:**
- PostgreSQL analytics queries (CSV output)
- MySQL inventory management
- Batch user status updates
- Multi-database transactions

**Run:**
```bash
cd sql-advanced && kdeps run workflow.yaml --dev
```

### 📤 [File Upload](./file-upload/)
File upload handling with multipart form-data support.
- Single and multiple file uploads
- File metadata access (count, names, types)
- File content, path, and MIME type access
- Integration with `get()` and `info()` functions

**Run:**
```bash
cd file-upload && kdeps run workflow.yaml --dev
```

### 👁️ [Vision](./vision/)
Vision model integration with image analysis.
- Image file upload via multipart form-data
- Vision model integration (moondream, llava)
- Image analysis with natural language queries
- JSON structured responses

**Prerequisites**: Install vision model: `ollama pull moondream:1.8b`

**Run:**
```bash
cd vision && kdeps run workflow.yaml --dev
```

### 🔧 [Tools](./tools/)
LLM tool calling / function calling with automatic tool execution.
- Tool/function definitions in ChatConfig
- Tool parameter schemas (OpenAI Functions format)
- Automatic tool execution when LLM calls tools
- Tool results fed back to LLM for final response
- Multi-turn conversations with tools
- Multiple tools per request

**Features:**
- ✅ Tool definitions work
- ✅ Automatic tool execution
- ✅ Tool resources (Python, SQL, HTTP, Exec, APIResponse)
- ✅ Tool arguments automatically stored in memory

**Run:**
```bash
cd tools && kdeps run workflow.yaml --dev
```

### 🔐 [Session Auth](./session-auth/)
Session management with persistent storage.
- SQLite-backed persistent session storage
- Session TTL configuration
- Session data storage and retrieval
- Preflight checks using session data

**Run:**
```bash
cd session-auth && kdeps run workflow.yaml --dev
```

## Quick Start

1. Choose an example directory
2. Update database connections if needed
3. Run: `kdeps run workflow.yaml --dev`
4. Test endpoints with curl or your browser

## Learning Path

1. **Start with Chatbot** - Learn basic YAML, HTTP API, and LLM integration
2. **Advanced SQL** - Master database operations, batch processing, and result formatting
3. **Build Your Own** - Combine features from examples

## Database Setup

Most examples require databases. See each example's README for setup instructions.

For quick testing, you can use SQLite (no setup required) by changing connection strings to:
```yaml
connection: "sqlite:///data.db"
```

## Development Mode

Use `--dev` flag for hot reload during development:
```bash
kdeps run workflow.yaml --dev
```

This automatically restarts the server when you modify workflow files or resources.
