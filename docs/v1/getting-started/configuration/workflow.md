---
outline: deep
---

# Workflow

The `workflow.pkl` contains configuration about the AI Agent, namely:

- AI agent `Name`, `Description`, `website`, `authors`, `documentation`, and `repository`.
- The semver `version` of this AI agent.
- The `targetActionID` resource to be executed when running the AI agent. This is the ID of the resource.
- Existing AI agents `workflows` to be reused in this AI agent. The agent needs to be installed first via
the `kdeps install` command.`

## Settings

The `settings` block allows advanced configuration of the AI agent, covering API settings, web server settings, routing,
Ubuntu and Python packages, and default LLM models.

```apl
Settings {
    APIServerMode = true
    APIServer {...}
    WebServerMode = false
    WebServer {...}
    AgentSettings {...}
}
```

### Overview

The `settings` block includes the following configurations:

- `APIServerMode`: A boolean flag that enables or disables API server mode for the project. When set to `false`, the
  default action is executed directly, and the program exits upon completion.
- `APIServer`: A configuration block that specifies API settings such as `HostIP`, `PortNum`, and `routes`.
- `WebServerMode`: A boolean flag that enables or disables the web server for serving frontends or proxying web
  applications.
- `WebServer`: A configuration block that specifies web server settings such as `HostIP`, `PortNum`, and `routes`.
- `agentSettings`: A configuration block that includes settings for installing Anaconda, `condaPackages`,
  `pythonPackages`, custom or PPA Ubuntu `repositories`, Ubuntu `packages`, Ollama LLM `models`, and `OfflineMode` for offline model deployment.


### API Server Settings

The `APIServer` block defines API routing configurations for the AI agent. These settings are only applied when
`APIServerMode` is set to `true`.

- `HostIP` **and** `PortNum`: Define the IP address and port for the Docker container. The default values are
  `"127.0.0.1"` for `HostIP` and `16395` for `PortNum`.

#### TrustedProxies

The `TrustedProxies` allows setting the allowable `X-Forwarded-For` header IPv4, IPv6, or CIDR addresses, used to limit
trusted requests to the service. You can obtain the client's IP address through `@(request.IP())`.

Example:

```apl
TrustedProxies {
  "127.0.0.1"
  "192.168.1.2"
  "10.0.0.0/8"
}
```

#### CORS Configuration

The `cors` block configures Cross-Origin Resource Sharing for the API server, controlling which origins, methods, and
headers are allowed for cross-origin requests. It enables secure access from web applications hosted on different
domains.

Example:

```apl
CORS {
    EnableCORS = true
    AllowOrigins {
        "https://example.com"
    }
    AllowMethods {
        "GET"
        "POST"
    }
    AllowHeaders {
        "Content-Type"
        "Authorization"
    }
    AllowCredentials = true
    MaxAge = 24.h
}
```

See the [CORS Configuration](/getting-started/configuration/cors.md) for more details.

#### API Routes

- `routes`: API paths can be configured within the `routes` block. Each route is defined using a `new` block,
  specifying:
  - `path`: The defined API endpoint, e.g., `"/api/v1/items"`.
  - `methods`: HTTP methods allowed for the route. Supported HTTP methods include: `GET`, `POST`, `PUT`, `PATCH`,
    `OPTIONS`, `DELETE`, and `HEAD`.

Example:

```apl
Routes {
    new {
        Path = "/api/v1/user"
        Methods {
            "GET"
        }
    }
    new {
        Path = "/api/v1/items"
        Methods {
            "POST"
        }
    }
}
```

Each route targets a single `targetActionID`, meaning every route points to the main action specified in the workflow
configuration. If multiple routes are defined, you must use a `skipCondition` logic to specify which route a resource
should target. See the Workflow for more details.

For instance, to run a resource only on the `"/api/v1/items"` route, you can define the following `skipCondition` logic:

```apl
local allowedPath = "/api/v1/items"
local requestPath = "@(request.path())"

SkipCondition {
    requestPath != allowedPath
}
```

In this example:

- The resource is skipped if the `skipCondition` evaluates to `true`.
- The resource runs only when the request path equals `"/api/v1/items"`.

For more details, refer to the Skip Conditions documentation.

#### Lambda Mode

When the `APIServerMode` is set to `false` in the workflow configuration, the AI agent operates in a **single-execution
lambda mode**. In this mode, the AI agent is designed to execute a specific task or serve a particular purpose,
completing its function in a single, self-contained execution cycle.

For example, an AI agent in single-execution lambda mode might be used to analyze data from a form submission, generate
a report, be executed as a scheduled `cron` job function, or provide a response to a one-time query, without the need
for maintaining an ongoing state or connection.

### Web Server Settings

The `WebServer` block defines configurations for serving frontend interfaces or proxying to web applications, enabling
KDeps to deliver full-stack AI applications with integrated UIs. These settings are only applied when `WebServerMode` is
set to `true`.

- `HostIP` **and** `PortNum`: Define the IP address and port for the web server. The default values are `"127.0.0.1"`
  for `HostIP` and `16395` for `PortNum`.


#### WebServerMode

- `WebServerMode`: A boolean flag that enables or disables the web server. When set to `true`, KDeps can serve static
  frontends (e.g., HTML, CSS, JS) or proxy to local web applications (e.g., Streamlit, Node.js). When `false`, the web
  server is disabled.

Example:

```apl
WebServerMode = true
```

#### WebServer

- `WebServer`: A configuration block that defines settings for the web server, including `HostIP`, `PortNum`,
  `TrustedProxies`, and `routes`. It is only active when `WebServerMode` is `true`.

Example:

```apl
WebServer {
    HostIP = "0.0.0.0"
    PortNum = 16395
    TrustedProxies {
        "192.168.1.0/24"
    }
}
```

#### Web Server Routes

- `routes`: Web server paths are configured within the `routes` block of the `WebServer` section. Each route is defined
  using a `web` block, specifying:
  - `path`: The HTTP path to serve, e.g., `"/dashboard"` or `"/app"`.
  - `serverType`: The serving mode: `"static"` for file hosting or `"app"` for reverse proxying.

Example:

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

Each route directs requests to static files (e.g., HTML, CSS, JS) or a local web app (e.g., Streamlit, Node.js),
enabling frontend integration with KDeps' AI workflows.

##### Static File Serving

- **`static`**: Serves files like HTML, CSS, or JS from a specified directory, ideal for hosting dashboards or
  frontends. The block with `ServerType = "static"` defines the path and directory relative to `/data/`,
  delivering files directly to clients.

Example:

```apl
WebServer {
    Routes {
        new {
            Path = "/dashboard"
            ServerType = "static"
            PublicPath = "/agentX/1.0.0/dashboard/"
        }
    }
}
```

This serves files from `/data/agentX/1.0.0/dashboard/` at `http://<host>:16395/dashboard`.

##### Reverse Proxying

- **`app`**: Forwards requests to a local web application (e.g., Streamlit, Node.js) running on a specified port. The
  block with `ServerType = "app"` defines the path, port, and optional command to start the app, proxying client
  requests to the app’s server.

Example:

```apl
WebServer {
    Routes {
        new {
            Path = "/app"
            ServerType = "app"
            PublicPath = "/agentX/1.0.0/streamlit-app/"
            AppPort = 8501
            Command = "streamlit run app.py"
        }
    }
}
```

This proxies requests from `http://<host>:16395/app` to a Streamlit app on port 8501, launched with `streamlit run
app.py`. For more details, see the [Web Server](/getting-started/configuration/webserver.md) documentation.

### AI Agent Settings

This section contains the agent settings that will be used to build the agent's Docker image.

```apl
AgentSettings {
    Timezone = "Etc/UTC"
    installAnaconda = false
    condaPackages { ... }
    PythonPackages { ... }
    Repositories { ... }
    Packages { ... }
    Models { ... }
    OllamaImageTag = "0.13.5"
    Env { ... }
    Args { ... }
}
```

#### Timezone Settings

Configure the `timezone` setting with a valid tz database identifier (e.g., `America/New_York`) for the Docker image;
see https://en.wikipedia.org/wiki/List_of_tz_database_time_zones for valid identifiers.

#### Enabling Anaconda

- `installAnaconda`: **"The Operating System for AI"**, Anaconda, will be installed when set to `true`. However, please
  note that if Anaconda is installed, the Docker image size will grow to &gt; 20GB. This does not include additional
  `condaPackages`. Defaults to `false`.

##### Anaconda Packages

- `condaPackages`: Anaconda packages to be installed if `installAnaconda` is `true`. The environment, channel, and
  packages can be defined in a single entry.

```apl
condaPackages {
    ["base"] {
        ["main"] = "pip diffusers numpy"
        ["pytorch"] = "pytorch"
        ["conda-forge"] = "tensorflow pandas keras transformers"
    }
}
```

This configuration will:

- Create the `base` isolated Anaconda environment.
- Use the `main` channel to install `pip`, `diffusers`, and `numpy` Anaconda packages.
- Use the `pytorch` channel to install `pytorch`.
- Use the `conda-forge` channel to install `tensorflow`, `pandas`, `keras`, and `transformers`.

To use the isolated environment, the Python resource should specify the Anaconda environment via the `CondaEnvironment`
setting.

#### Python Packages

Python packages can also be installed even without Anaconda installed.

```apl
PythonPackages {
    "diffusers[torch]"
    "streamlit"
    "openai-whisper"
}
```

#### Ubuntu Repositories

Additional Ubuntu and Ubuntu PPA repositories can be defined in the `repositories` settings.

```apl
Repositories {
    "ppa:alex-p/tesseract-ocr-devel"
}
```

In this example, a PPA repository is added for installing the latest `tesseract-ocr` package.

#### Ubuntu Packages

Specify the Ubuntu packages that should be pre-installed when building this image.

```apl
Packages {
    "tesseract-ocr"
    "poppler-utils"
    "npm"
    "ffmpeg"
}
```

#### LLM Models

List the local Ollama LLM models that will be pre-installed. You can specify multiple models.

```apl
Models {
    "tinydolphin"
    "llama3.3"
    "llama3.2-vision"
    "llama3.2:1b"
    "mistral"
    "gemma"
    "mistral"
}
```

KDeps uses Ollama as its LLM backend. You can define as many Ollama-compatible models as needed to fit your use case.

For a comprehensive list of available Ollama-compatible models, visit the Ollama model library.

#### Offline Mode

The `OfflineMode` setting controls whether models are pulled during Docker build time for offline operation:

```apl
OfflineMode = true
```

When `OfflineMode` is set to `true`:
- Models specified in the `Models` block are pulled during Docker image build time
- KDeps uses `OLLAMA_MODELS` exclusively to control model storage locations:
  - Build-time: `OLLAMA_MODELS=/models` inside the image (baked into the image)
  - Runtime: `OLLAMA_MODELS` environment variable (defaults to `/root/.ollama` if unset)
- At runtime, models are copied from the image (`/models`) into `${OLLAMA_MODELS}` (e.g., the mounted volume) instead of being downloaded
- This enables true offline operation without requiring internet connectivity

When `OfflineMode` is set to `false`:
- Models are pulled at runtime from the internet when the container starts
- Requires internet connectivity during container startup

This feature is particularly useful for:
- Deployments in air-gapped environments
- Reducing container startup time by avoiding model downloads
- Ensuring consistent model availability regardless of internet connectivity

#### Ollama Docker Image Tag

The `ollamaImageTag` configuration property allows you to dynamically specify the version of the Ollama base image tag
used in your Docker image.

When used in conjunction with a GPU configuration in the `.kdeps.pkl` file, this configuration can automatically adjust
the image version to include hardware-specific extensions, such as `1.0.0-rocm` for AMD environments.

#### Arguments and Environment Variables

KDeps allows you to define `ENV` (environment variables) that persist across both the Docker image and container
runtime, and `ARG` (arguments) that are used for passing values during the build process.

To declare `ENV` or `ARG` parameters, use the `env` and `args` sections in your workflow configuration:

```apl
Env {
  ["API_KEY"] = "example_value"
}

Args {
  ["API_TOKEN"] = ""
}
```

In this example:

- `API_KEY` is declared as an environment variable with the value `"example_value"`. This variable will persist in both
  the Docker image and the container at runtime.
- `API_TOKEN` is an argument that does not have a default value and will accept a value at container runtime.

**Environment File Support:**

Additionally, any `.env` file in your project will be automatically loaded via `kdeps run`, and the variables defined
within it will populate the `env` or `args` sections accordingly.

**Important Notes:**

- `ENV` variables must always be assigned a value during declaration.
- `ARG` variables can be declared without a value (e.g., `""`). These will act as standalone runtime arguments.
- Values defined in the `.env` file will override default values for any matching `ENV` or `ARG` keys.
