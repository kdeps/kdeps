---
outline: deep
---

# Kdeps Resource Types

Kdeps currently supports the following resource types with v0.3.1 schema:

- **`Exec`**: [Exec Resource](../core-resources/exec.md)
  Executes shell scripts and commands.

- **`Python`**: [Python Resource](../core-resources/python.md)
  Executes Python scripts.

- **`Response`**: [Response Resource](../core-resources/response.md)
  Handles API responses by generating JSON output to an API endpoint.

- **`Client`**: [HTTP Client Resource](../core-resources/client.md)
  Performs HTTP client requests, facilitating communication with external APIs or services.

- **`LLM`**: [LLM Resource](../core-resources/llm.md)
  Provides interaction with language models (LLMs).

## Resource Type Examples

### Exec Resource
```apl
ActionID = "execResource"
Name = "Shell Command"
Description = "Execute system commands"
Category = "system"
Run {
  Exec {
    Command = "ls -la"
    Args { "-la" }
    Env { ["PATH"] = "/usr/bin:/bin" }
  }
}
```

### Python Resource
```apl
ActionID = "pythonResource"
Name = "Python Script"
Description = "Execute Python code"
Category = "scripting"
Run {
  Python {
    Script = "print('Hello, World!')"
    Input { ["data"] = "@(request.data())" }
    Output { ["result"] = "stdout" }
  }
}
```

### Response Resource
```apl
ActionID = "responseResource"
Name = "API Response"
Description = "Return structured JSON response"
Category = "output"
Run {
  APIResponse {
    Success = true
    Response {
      Data { "@(llm.response('llmResource'))" }
    }
    Meta { Headers { ["Content-Type"] = "application/json" } }
  }
}
```

### Client Resource
```apl
ActionID = "clientResource"
Name = "HTTP Client"
Description = "Make HTTP requests"
Category = "network"
Run {
  Client {
    Method = "GET"
    URL = "https://api.example.com/data"
    Headers { ["Authorization"] = "Bearer token" }
  }
}
```

### LLM Resource
```apl
ActionID = "llmResource"
Name = "Language Model"
Description = "Process queries with LLM"
Category = "ai"
Run {
  Chat {
    Model = "llama3.2:1b"
    Prompt = "Answer this question: @(request.data().query)"
    JSONResponse = true
    JSONResponseKeys { "answer"; "confidence" }
    TimeoutDuration = 60.s
  }
}
```
