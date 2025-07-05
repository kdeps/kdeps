---
outline: deep
---

# Workflow Configuration

The `workflow.pkl` file contains the comprehensive configuration for your AI agent, including:

- **Agent Metadata**: `AgentID`, `Description`, `Website`, `Authors`, `Documentation`, and `Repository` information
- **Version Management**: Semantic versioning (`Version`) for the AI agent
- **Target Execution**: `TargetActionID` specifying which resource to execute when running the agent
- **Workflow Composition**: `Workflows` section for reusing existing AI agents installed via `kdeps install`

## Settings Overview

The `Settings` block provides advanced configuration for the AI agent, covering API settings, web server configuration, routing, Ubuntu and Python packages, and default LLM models.

```apl
amends "workflow.pkl"

AgentID = "myAIAgent"
Description = "A sample AI agent for data processing"
Version = "1.0.0"
TargetActionID = "responseResource"

Settings {
    RateLimitMax = 100
    Environment = "production"
    APIServerMode = true
    APIServer { /* API configuration */ }
    WebServerMode = false
    WebServer { /* Web server configuration */ }
    AgentSettings { /* Agent-specific settings */ }
}
```
```

### Configuration Properties

The `Settings` block includes these key configurations:

- **`RateLimitMax`**: Maximum number of API requests allowed per minute for rate limiting control
- **`Environment`**: Build environment specification (`"development"`, `"staging"`, or `"production"`)
- **`APIServerMode`**: Boolean flag enabling or disabling API server mode. When set to `false`, the default action executes directly and the program exits upon completion
- **`APIServer`**: Configuration block for API settings including `HostIP`, `PortNum`, and `Routes`
- **`WebServerMode`**: Boolean flag enabling or disabling web server for serving frontends or proxying web applications
- **`WebServer`**: Configuration block for web server settings including `HostIP`, `PortNum`, and `Routes`
- **`AgentSettings`**: Configuration for Anaconda installation, `CondaPackages`, `PythonPackages`, custom Ubuntu `Repositories`, Ubuntu `Packages`, and Ollama LLM `Models`

## API Server Configuration

The `APIServer` block defines API routing configurations for the AI agent. These settings apply only when `APIServerMode` is set to `true`.

### Basic Configuration

```apl
APIServer {
    HostIP = "127.0.0.1"
    PortNum = 3000
    TrustedProxies {
        "127.0.0.1"
        "192.168.1.2"
        "10.0.0.0/8"
    }
    CORS {
        EnableCORS = true
        AllowOrigins {
            "https://example.com"
            "https://app.mydomain.com"
        }
        AllowMethods {
            "GET"
            "POST"
            "PUT"
            "DELETE"
        }
        AllowHeaders {
            "Content-Type"
            "Authorization"
            "X-API-Key"
        }
        AllowCredentials = true
        MaxAge = 24.h
    }
    Routes {
        new {
            Path = "/api/v1/user"
            Method = "GET"
        }
        new {
            Path = "/api/v1/items"
            Method = "POST"
        }
        new {
            Path = "/api/v1/health"
            Method = "GET"
        }
    }
}
```
```

- **`HostIP` and `PortNum`**: Define the IP address and port for the Docker container. Default values are `"127.0.0.1"` for `HostIP` and `3000` for `PortNum`

### Trusted Proxies

The `TrustedProxies` configuration allows setting allowable `X-Forwarded-For` header IPv4, IPv6, or CIDR addresses to limit trusted requests to the service. You can obtain the client's IP address using `@(request.IP())`.

**Example:**
```apl
TrustedProxies {
    "127.0.0.1"
    "192.168.1.2"
    "10.0.0.0/8"
}
```

### CORS Configuration

The `CORS` block configures Cross-Origin Resource Sharing for the API server, controlling which origins, methods, and headers are allowed for cross-origin requests. This enables secure access from web applications hosted on different domains.

**Example:**
```apl
CORS {
    EnableCORS = true
    AllowOrigins {
        "https://example.com"
        "https://app.mydomain.com"
    }
    AllowMethods {
        "GET"
        "POST"
        "PUT"
        "DELETE"
    }
    AllowHeaders {
        "Content-Type"
        "Authorization"
        "X-API-Key"
    }
    AllowCredentials = true
    MaxAge = 24.h
}
```

For detailed CORS configuration options, see the [CORS Configuration](./cors.md) documentation.

### API Routes

API paths are configured within the `Routes` block. Each route is defined using a `new` block with these properties:

- **`Path`**: The defined API endpoint (e.g., `"/api/v1/items"`)
- **`Method`**: HTTP method allowed for the route

**Supported HTTP Methods**: `GET`, `POST`, `PUT`, `PATCH`, `OPTIONS`, `DELETE`, and `HEAD`

**Example:**
```apl
Routes {
    new {
        Path = "/api/v1/user"
        Method = "GET"
    }
    new {
        Path = "/api/v1/items"
        Method = "POST"
    }
    new {
        Path = "/api/v1/health"
        Method = "GET"
    }
}
```

### Route-Specific Resource Execution

Each route targets a single `TargetActionID`, meaning every route points to the main action specified in the workflow configuration. To run different resources for different routes, use `SkipCondition` logic to specify route targeting.

**Example: Route-Specific Execution**

To run a resource only on the `"/api/v1/items"` route:

```apl
local allowedPath = "/api/v1/items"
local requestPath = "@(request.path())"

SkipCondition {
    requestPath != allowedPath
}
```

In this example:
- The resource is skipped if the `SkipCondition` evaluates to `true`
- The resource runs only when the request path equals `"/api/v1/items"`

For more details, refer to the [Skip Conditions](../../workflow-control/skip.md) documentation.

### Lambda Mode

When `APIServerMode` is set to `false`, the AI agent operates in **single-execution lambda mode**. In this mode, the agent executes a specific task or serves a particular purpose in a single, self-contained execution cycle.

**Use Cases for Lambda Mode:**
- Analyzing data from form submissions
- Generating reports or documents
- Scheduled `cron` job functions
- One-time query processing
- Batch data processing tasks

## Web Server Configuration

The `WebServer` block defines configurations for serving frontend interfaces or proxying to web applications, enabling Kdeps to deliver full-stack AI applications with integrated UIs.

### Basic Configuration

```apl
WebServerMode = true
WebServer {
    HostIP = "127.0.0.1"
    PortNum = 8080
    TrustedProxies { /* proxy settings */ }
    Routes { /* web routes */ }
}
```

- **`HostIP` and `PortNum`**: Define the IP address and port for the web server. Default values are `"127.0.0.1"` for `HostIP` and `8080` for `PortNum`
- **`TrustedProxies`**: Similar to API server, defines trusted proxy addresses

### Web Server Routes

Web server paths are configured within the `Routes` block of the `WebServer` section. Each route is defined using a `new` block with these properties:

- **`Path`**: The HTTP path to serve (e.g., `"/dashboard"` or `"/app"`)
- **`ServerType`**: The serving mode - `"static"` for file hosting or `"app"` for reverse proxying

**Example:**
```apl
WebServer {
    Routes {
        new {
            Path = "/dashboard"
            ServerType = "static"
            PublicPath = "/agentX/1.0.0/dashboard/"
        }
        new {
            Path = "/app"
            ServerType = "app"
            AppPort = 8501
            Command = "streamlit run app.py"
        }
    }
}
```

### Static File Serving

The `"static"` server type serves files like HTML, CSS, or JavaScript from a specified directory, ideal for hosting dashboards or frontends.

**Configuration:**
```apl
new {
    Path = "/dashboard"
    ServerType = "static"
    PublicPath = "/agentX/1.0.0/dashboard/"
}
```

This configuration serves files from `/data/agentX/1.0.0/dashboard/` at `http://<host>:8080/dashboard`.

**File Organization:**
- Place static files in the specified `PublicPath` directory
- Organize with standard web structure (HTML, CSS, JS, images)
- Ensure proper file permissions for web access

### Reverse Proxying

The `"app"` server type forwards requests to a local web application (e.g., Streamlit, Node.js) running on a specified port.

**Configuration:**
```apl
new {
    Path = "/app"
    ServerType = "app"
    PublicPath = "/agentX/1.0.0/streamlit-app/"
    AppPort = 8501
    Command = "streamlit run app.py"
}
```

This configuration:
- Proxies requests from `http://<host>:8080/app` to a Streamlit app on port 8501
- Launches the app with `streamlit run app.py`
- Serves the app from the specified `PublicPath`

**Supported Frameworks:**
- Streamlit applications
- Node.js web servers
- Flask/FastAPI applications
- Custom web applications

For more details, see the [Web Server Configuration](./webserver.md) documentation.

## Agent Settings

The `AgentSettings` section contains configuration for building the agent's Docker image with required dependencies, packages, and models.

```apl
AgentSettings {
    Timezone = "Etc/UTC"
    InstallAnaconda = false
    CondaPackages { /* Anaconda packages */ }
    PythonPackages { /* Python packages */ }
    Repositories { /* Ubuntu repositories */ }
    Packages { /* Ubuntu packages */ }
    Models { /* LLM models */ }
    OllamaVersion = "0.8.0"
    Env { /* environment variables */ }
    Args { /* build arguments */ }
    ExposedPorts { /* exposed ports */ }
}
```

### Timezone Configuration

Configure the timezone setting with a valid tz database identifier for the Docker image.

**Example:**
```apl
Timezone = "America/New_York"  // Eastern Time
Timezone = "Europe/London"     // GMT/BST
Timezone = "Asia/Tokyo"        // JST
```

See the [tz database time zones](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) for valid identifiers.

### Anaconda Installation

**Anaconda**: "The Operating System for AI" - When `InstallAnaconda` is set to `true`, Anaconda will be installed in the Docker image.

```apl
InstallAnaconda = true
```

**Important Considerations:**
- Installing Anaconda increases Docker image size to > 20GB
- Additional `CondaPackages` will further increase image size
- Consider using standard Python packages when possible
- Defaults to `false` for smaller image sizes

### Anaconda Packages

When `InstallAnaconda` is `true`, you can specify Anaconda packages to install with environment, channel, and package definitions.

**Configuration:**
```apl
CondaPackages {
    ["base"] {
        ["main"] = "pip diffusers numpy"
        ["pytorch"] = "pytorch torchvision"
        ["conda-forge"] = "tensorflow pandas keras transformers"
    }
    ["ml-env"] {
        ["conda-forge"] = "scikit-learn matplotlib seaborn"
        ["nvidia"] = "cudatoolkit"
    }
}
```

This configuration:
- Creates the `base` isolated Anaconda environment with:
  - `main` channel: `pip`, `diffusers`, `numpy`
  - `pytorch` channel: `pytorch`, `torchvision`
  - `conda-forge` channel: `tensorflow`, `pandas`, `keras`, `transformers`
- Creates the `ml-env` environment with machine learning packages

**Using Conda Environments:**
To use an isolated environment, specify the Anaconda environment in the Python resource via the `condaEnvironment` setting.

### Python Packages

Python packages can be installed even without Anaconda, using pip for package management.

**Configuration:**
```apl
PythonPackages {
    "diffusers[torch]"
    "streamlit>=1.28.0"
    "openai-whisper"
    "fastapi[all]"
    "python-multipart"
    "pandas>=1.5.0"
    "numpy>=1.21.0"
}
```

**Best Practices:**
- Specify version constraints for critical dependencies
- Use extras syntax (e.g., `fastapi[all]`) when needed
- Group related packages logically
- Consider package size and build time

### Ubuntu Repositories

Add additional Ubuntu and Ubuntu PPA (Personal Package Archive) repositories for accessing specialized packages.

**Configuration:**
```apl
Repositories {
    "ppa:alex-p/tesseract-ocr-devel"
    "ppa:deadsnakes/ppa"
    "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ focal main"
}
```

**Repository Types:**
- **PPA Repositories**: Easy-to-add Personal Package Archives
- **Custom Debian Repositories**: Full repository URLs with architecture and distribution
- **Official Ubuntu Repositories**: Additional official package sources

### Ubuntu Packages

Specify Ubuntu packages to pre-install when building the Docker image.

**Configuration:**
```apl
Packages {
    "tesseract-ocr"
    "tesseract-ocr-eng"
    "poppler-utils"
    "npm"
    "ffmpeg"
    "curl"
    "wget"
    "git"
    "build-essential"
}
```

**Common Package Categories:**
- **OCR Tools**: `tesseract-ocr`, `tesseract-ocr-eng`
- **Document Processing**: `poppler-utils`, `pandoc`
- **Media Processing**: `ffmpeg`, `imagemagick`
- **Development Tools**: `git`, `build-essential`, `curl`, `wget`
- **Language Runtimes**: `nodejs`, `npm`, `python3-dev`

### LLM Models

Define the local Ollama LLM models to pre-install in the Docker image.

**Configuration:**
```apl
Models {
    "tinydolphin"        // Lightweight model for development
    "llama3.3:latest"    // Latest LLaMA 3.3 model
    "llama3.2-vision"    // Multi-modal model for vision tasks
    "llama3.2:1b"        // Compact 1B parameter model
    "mistral:7b"         // Mistral 7B model
    "gemma:2b"           // Google Gemma 2B model
    "codellama:13b"      // Code-specialized model
}
```

**Model Selection Guidelines:**
- **Development**: Use lightweight models like `tinydolphin` or `llama3.2:1b`
- **Production**: Use larger, more capable models like `llama3.3` or `mistral:7b`
- **Vision Tasks**: Include `llama3.2-vision` for image processing
- **Code Generation**: Add `codellama` for programming tasks

Kdeps uses [Ollama](https://ollama.com) as its LLM backend. Visit the [Ollama model library](https://ollama.com/library) for a comprehensive list of available models.

### Ollama Version

The `OllamaVersion` property dynamically specifies the version of the Ollama base image tag.

```apl
OllamaVersion = "0.8.0"
```

**GPU-Specific Versions:**
When used with GPU configuration in the `.kdeps.pkl` file, this automatically adjusts the image version to include hardware-specific extensions:
- **AMD GPUs**: `0.8.0-rocm`
- **NVIDIA GPUs**: `0.8.0-cuda`
- **CPU Only**: `0.8.0`

### Environment Variables and Arguments

Define `ENV` (environment variables) that persist across Docker image and container runtime, and `ARG` (arguments) for passing values during the build process.

**Configuration:**
```apl
Env {
    ["API_KEY"] = "example_value"
    ["DATABASE_URL"] = "postgresql://localhost:5432/mydb"
    ["LOG_LEVEL"] = "INFO"
    ["MAX_WORKERS"] = "4"
}

Args {
    ["API_TOKEN"] = ""
    ["BUILD_VERSION"] = "1.0.0"
    ["CUSTOM_CONFIG"] = ""
}
```

**Environment Variables (`ENV`):**
- Must always be assigned a value during declaration
- Persist in both Docker image and container at runtime
- Available to all processes within the container
- Can be overridden at container runtime

**Build Arguments (`ARG`):**
- Can be declared without a value (e.g., `""`)
- Accept values at build time or container runtime
- Used for customizing the build process
- Not persisted in the final image unless explicitly set as ENV

### Environment File Support

Any `.env` file in your project will be automatically loaded via `kdeps run`, and the variables defined within it will populate the `Env` or `Args` sections accordingly.

**Example `.env` file:**
```bash
API_KEY=your_actual_api_key
DATABASE_URL=postgresql://prod.example.com:5432/production
LOG_LEVEL=DEBUG
BUILD_VERSION=2.1.0
```

**Important Notes:**
- Values in `.env` files override default values for matching `ENV` or `ARG` keys
- Use `.env` files for environment-specific configurations
- Never commit sensitive `.env` files to version control
- Use different `.env` files for different environments (development, staging, production)

## Best Practices

### Configuration Management
- Use environment-specific configurations for different deployment stages
- Keep sensitive data in `.env` files (not in version control)
- Document all configuration options and their purposes
- Use semantic versioning for agent versions

### Performance Optimization
- Choose appropriate model sizes for your use case
- Install only necessary packages to minimize image size
- Use multi-stage Docker builds when possible
- Consider package caching strategies

### Security Considerations
- Restrict CORS origins to specific domains in production
- Use trusted proxies configuration properly
- Validate all environment variables
- Implement proper rate limiting

### Resource Management
- Monitor Docker image sizes
- Set appropriate memory and CPU limits
- Use health checks for long-running services
- Implement proper logging and monitoring

## Next Steps

- **[CORS Configuration](./cors.md)**: Detailed CORS setup and security
- **[Web Server Configuration](./webserver.md)**: Advanced web server features
- **[System Configuration](./configuration.md)**: Global system settings
- **[Core Resources](../../core-resources/README.md)**: Understanding resource configuration

The workflow configuration is the foundation of your Kdeps AI agent. Take time to understand each section and configure it appropriately for your specific use case and deployment environment.
