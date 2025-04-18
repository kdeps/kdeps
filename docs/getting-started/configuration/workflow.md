---
outline: deep
---

# Workflow

The `workflow.pkl` contains configuration about the AI Agent, namely:

- AI agent `name`, `description`, `website`, `authors`, `documentation` and `repository`.
- The [semver](https://semver.org) `version` of this AI agent.
> **Note on version:**
> kdeps uses the version for mapping the graph-based dependency workflow execution order. For this reason, the version
> is *required*.

- The `targetActionID` resource to be executed when running the AI agent. This is the ID of the resource.
- Existing AI agents `workflows` to be reused in this AI agent. The agent needed to be installed first via `kdeps
  install` command.

## Settings

The `settings` block allows advanced configuration of the AI agent, covering API settings, routing, Ubuntu and Python
packages, and default LLM models.

```apl
settings {
    APIServerMode = true
    APIServer {...}
    agentSettings {...}
}
```

### Overview

The `settings` block includes the following configurations:

- **`APIServerMode`**: A boolean flag that enables or disables API server mode for the project. When set to `false`, the
  default action is executed directly, and the program exits upon completion.

- **`APIServer`**: A configuration block that specifies API settings such as `hostIP`, `portNum`, and `routes`.

- **`agentSettings`**: A configuration block that includes settings for installing Anaconda, `condaPackages`,
  `pythonPackages`, custom or PPA Ubuntu `repositories`, Ubuntu `packages`, and Ollama LLM `models`.


### API Server Settings

The `APIServer` block defines API routing configurations for the AI agent. These settings are only applied when
`APIServerMode` is set to `true`.

- **`hostIP` and `portNum`**: Define the IP address and port for the Docker container. The default values are
  `"127.0.0.1"` for `hostIP` and `3000` for `portNum`.

#### TrustedProxies

The `trustedProxies` allows setting the allowable `X-Forwarded-For` header IPv4, IPv5, CIDR addresses, used to limit the
trusted request using the service. You can obtain the client's IP address through `@(request.IP())`.

Example:

```apl
trustedProxies {
  "127.0.0.1"
  "192.168.1.2"
  "10.0.0.0/8"
}
```

#### CORS Configuration

The `CORS` block configures Cross-Origin Resource Sharing for the API server, controlling which origins, methods, and
headers are allowed for cross-origin requests. It enables secure access from web applications hosted on different
domains.

Example:

```pkl
CORS {
    enableCORS = true
    allowOrigins = new {
        "https://example.com"
    }
    allowMethods = new {
        "GET"
        "POST"
    }
    allowHeaders = new {
        "Content-Type"
        "Authorization"
    }
    allowCredentials = true
    maxAge = 24.h
}
```

See the [CORS Configuration](/getting-started/resources/cors.md) for more details.
#### API Routes

- **`routes`**: API paths can be configured within the `routes` block. Each route is defined using a `new` block,
 specifying:
   - **`path`**: The defined API endpoint, i.e. `"/api/v1/items"`.
   - **`methods`**: HTTP methods allowed for the route. Supported HTTP methods include: `GET`, `POST`, `PUT`, `PATCH`,
     `OPTIONS`, `DELETE`, and `HEAD`.

Example:

```apl
routes {
    new {
        path = "/api/v1/user"
        methods {
            "GET"
        }
    }
    new {
        path = "/api/v1/items"
        methods {
            "POST"
        }
    }
}
```

Each route targets a single `targetActionID`, meaning every route points to the main action specified in the workflow
configuration. If multiple routes are defined, you must use a `skipCondition` logic to specify which route a resource
should target. See the [Workflow](#workflow) for more details.

For instance, to run a resource only on the `"/api/v1/items"` route, you can define the following `skipCondition` logic:

```apl
local allowedPath = "/api/v1/items"
local requestPath = "@(request.path())"

skipCondition {
    requestPath != allowedPath
}
```

In this example:
- The resource is skipped if the `skipCondition` evaluates to `true`.
- The resource runs only when the request path equals `"/api/v1/items"`.

For more details, refer to the [Skip Conditions](/getting-started/resources/skip.md) documentation.

#### Lambda Mode

When the `APIServerMode` is set to `false` in the workflow configuration, the AI agent operates in a **single-execution
lambda mode**. In this mode, the AI agent is designed to execute a specific task or serve a particular purpose,
completing its function in a single, self-contained execution cycle.

For example, an AI agent in single-execution lambda mode might be used to analyze data from a form submission, generate
a report, be executed as a scheduled `cron` job function or provide a response to a one-time query, without the need for
maintaining an ongoing state or connection.

### AI Agent Settings

This section contains the agent settings that will be used to build the agent's Docker image.

```apl
agentSettings {
    timezone = "Etc/UTC"
    installAnaconda = false
    condaPackages { ... }
    pythonPackages { ... }
    repositories { ... }
    packages { ... }
    models { ... }
    ollamaImageTag = "0.5.4"
    env { ... }
    args { ... }
}
```

#### Timezone Settings

Configure the `timezone` setting with a valid tz database identifier (e.g., `America/New_York`) for the Docker image;
see https://en.wikipedia.org/wiki/List_of_tz_database_time_zones for valid identifiers.

#### Enabling Anaconda

- **`installAnaconda`**: **"The Operating System for AI"**, [Anaconda](https://www.anaconda.com),  will be installed when
  set to `true`. However, please take note that if Anaconda is installed, the Docker image size will grow to >
  20Gb. That does not includes the additional `condaPackages`. Defaults to `false`.

##### Anaconda Packages

- **`condaPackages`**: Anaconda packages to be installed if `installAnaconda` is `true`. The environment, channel and
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
- Creates the `base` isolated Anaconda environment.
- Use the channels `main` to install `pip`, `diffusers` and `numpy` Anaconda packages.
- Use the `pytorch` channel to install `pytorch`.
- Use the `conda-forge` channel to install `tensorflow`, `pandas`, `keras`, and `transformers`.

In order to use the isolated environment, the Python resource should specify the Anaconda environment via the
`condaEnvironment` setting.

#### Python Packages

Python packages can also be installed even without Anaconda installed.

```apl
pythonPackages {
    "diffusers[torch]"
}
```

#### Ubuntu Repositories

Additional Ubuntu and Ubuntu PPA repositories can be defined in the `repositories` settings.

```apl
repositories {
    "ppa:alex-p/tesseract-ocr-devel"
}
```

In this example, a PPA repository is added to installing the latest `tesseract-ocr` package.

#### Ubuntu Packages

Specify the Ubuntu packages that should be pre-installed when building this image.

```apl
packages {
    "tesseract-ocr"
    "poppler-utils"
}
```

#### LLM Models
List the local Ollama LLM models that will be pre-installed. You can specify multiple models.

```apl
models {
    "tinydolphin"
    "llama3.3"
    "llama3.2-vision"
    "mistral"
    "gemma"
    "mistral"
}
```

Kdeps uses [Ollama](https://ollama.com) as it's LLM backend. You can define as many Ollama compatible models as needed
to fit your use case.

For a comprehensive list of available Ollama compatible models, visit the [Ollama model
library](https://ollama.com/library).

#### Ollama Docker Image Tag
The `ollamaImageTag` configuration property allows you to dynamically specify the version of the Ollama base image tag
used in your Docker image.

When used in conjunction with a GPU configuration in `.kdeps.pkl` file, this configuration can automatically adjust the
image version to include hardware-specific extensions, such as `1.0.0-rocm` for AMD environments.

#### Arguments and Environment Variables

Kdeps allows you to define `ENV` (environment variables) that persist across both the Docker image and container
runtime, and `ARG` (arguments) that are used for passing values during the build process.

To declare `ENV` or `ARG` parameters, use the `env` and `args` sections in your workflow configuration:

```apl
env {
  ["API_KEY"] = "example_value"
}

args {
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
