---
outline: deep
---

# AI Image Generators

KDeps does not directly generate images using an LLM (Large Language Model). However, it includes a dedicated Python
resource capable of executing Python scripts that can produce images through advanced models such as
`stable-diffusion`. You can optionally use using Anaconda. Additionally, any `GGUF` models from HuggingFace can also
be used for image generation.

## Setting Up for Image Generation

To begin, you need to include the `torch` and `diffusers` in the `pythonPackages` block in `workflow.pkl`
file. Optionally, add the `huggingface_hub` package if you plan to use gated models.

Since no LLM models are required for this process, the `models` block can remain empty.

We will also set the `targetActionID` to `APIResponseResource` and create a route `/api/v1/image_generator`.

Example `workflow.pkl` configuration:

```js
Name = "sd35api"
Version = "1.0.0"
...
Settings {
  ...
  APIServer {
    HostIP = "127.0.0.1"
    PortNum = 16395

    Routes {
      new {
        Path = "/api/v1/image_generator"
        Methods {
          "POST"
        }
      }
    }
  }

  AgentSettings {
    ...
    PythonPackages {
      "torch"
      "diffusers"
      "huggingface_hub[cli]"
    }
    ...
    Models {}
  }
}
```

### Writing the Image Generation Script

Save the following Python script in the `data/` folder as `data/sd3_5.py`. This script loads the `stable-diffusion`
model, retrieves a prompt from the `PROMPT` environment variable, and saves the generated image to `/tmp/image.png`.

```python
import os
import torch
from diffusers import StableDiffusion3Pipeline

# Retrieve the prompt from the environment variable
Prompt = os.getenv("PROMPT", "A capybara holding a sign that reads 'Hello World'")

# Remove the file if it already exists
file_Path = "/tmp/image.png"
if os.path.exists(file_path):
    os.remove(file_path)

# Load the Stable Diffusion model
pipe = StableDiffusion3Pipeline.from_pretrained("stabilityai/stable-diffusion-3.5-large", torch_dtype=torch.bfloat16)
pipe = pipe.to("cuda")

# Generate the image
image = pipe(
    prompt,
    num_inference_steps=28,
    guidance_scale=3.5,
).images[0]

# Save the generated image
image.save(file_path)
```

### Configuring the Python Resource

To integrate the script into the workflow, configure the Python resource. Reference the script's file path within the
`data` folder and set the `PROMPT` environment variable using the `env` block. This variable is dynamically populated
using the `request.params("q")` function.


```json
ActionID = "pythonResource"

Python {
  local pythonScriptPath = "@(data.filepath("sd35api/1.0.0", "sd3_5.py"))"
  local pythonScript = "@(read?("\(pythonScriptPath)")?.text)"

  Script = """
\(pythonScript)
"""
  Env {
    ["PROMPT"] = "@(request.params("q"))"
  }
...
}
```

### Linking to the API Response Resource

With the Python resource prepared, include it in the `requires` block of the API Response resource. This ensures the
script is executed as part of the workflow.

```js
ActionID = "APIResponseResource"
Requires {
  "pythonResource"
}
```

Finally, the generated image file (`/tmp/image.png`) can be encoded as a `base64` PNG file and included in the API
response.

```json
local generatedFileBase64 = "@(read("/tmp/image.png").base64)"
local responseJson = new Mapping {
  ["file"] = "data:image/png;base64,\(generatedFileBase64)"
}

APIResponse {
...
  Response {
    Data {
        responseJson
    }
  }
}
```

### Testing the AI Image Generator API

To test our newly created AI image generator API, we can use `curl`, which will output the file as a base64 string.

```bash
curl "http://localhost:16395/api/v1/image_generator?q=A+Teddy+Bear"

{
  "errors": [
    {
      "code": 0,
      "message": ""
    }
  ],
  "response": {
    "data": [
      {
        "file": "data:image/png;base64,...."
      }
    ]
  },
  "success": true
}
```

Here's an improved and rephrased version of your content:

---

## Authenticating and Downloading Models from Hugging Face

For some image models, authentication with `huggingface.co` is required. In this guide, we'll use a Hugging Face access
token and configure it securely.

### Setting Up the `.env` File

Create a `.env` file outside the AI agent directory to store the access token:

```text
HF_TOKEN=hf_****
```

### Updating `workflow.pkl`

Add `huggingface_hub[cli]` to the `pythonPackages` block in the `workflow.pkl` file. Additionally, declare an `ARG` for
`HF_TOKEN`, which will reference the token stored in the `.env` file.

```js
AgentSettings {
    ...
    PythonPackages {
        "torch"
        "diffusers"
        "huggingface_hub[cli]"
    }
    ...
    Args {
        ["HF_TOKEN"] = "secret"
    }
}
```

### Creating an `exec` Resource

Generate an `exec` resource using the following command:

```bash
kdeps scaffold sd35api exec
```

In the `exec` resource, include a script that logs in to Hugging Face using the `HF_TOKEN` from the `.env` file and
downloads the model. Additionally, set the cache directory to `/agent/volume/`, a shared folder for KDeps in Docker, and create a
marker file (`/agent/volume/sd35-downloaded`) upon successful download.

```json
ActionID = "execResource"
...
Exec {
    Command = """
    huggingface-cli login --token $HF_TOKEN
    huggingface-cli download stabilityai/stable-diffusion-3.5-large --cache-dir /agent/volume/
    echo downloaded > /agent/volume/sd35-downloaded
    """
    Env {
        ["HF_TOKEN"] = "\(read("env:HF_TOKEN"))"
    }
    TimeoutDuration = 0.s
}
```

### Adding a `skipCondition`

To ensure the `exec` script runs only when necessary, add a `skipCondition`. This condition checks for the existence of
the `/agent/volume/sd35-downloaded` file. If the file exists, the script will be skipped.

```json
ActionID = "execResource"
...
run {
    local stampFile = read?("file:/agent/volume/sd35-downloaded")?.base64?.isEmpty

    SkipCondition {
        stampFile != null || stampFile != false
    }
...
```
