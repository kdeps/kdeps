---
outline: deep
---

# AI Image Generators

Kdeps does not directly generate images using an LLM (Large Language Model). However, it includes a dedicated Python
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
name = "sd35api"                           // [!code ++]
version = "1.0.0"                          // [!code ++]
...
settings {
  ...
  APIServer {
    hostIP = "127.0.0.1"
    portNum = 3000

    routes {
      new {
        path = "/api/v1/image_generator"  // [!code ++]
        methods {
          "POST"
        }
      }
    }
  }

  agentSettings {
    ...
    pythonPackages {
      "torch"                             // [!code ++]
      "diffusers"                         // [!code ++]
      "huggingface_hub[cli]"              // [!code ++]
    }
    ...
    models {}
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
prompt = os.getenv("PROMPT", "A capybara holding a sign that reads 'Hello World'")

# Remove the file if it already exists
file_path = "/tmp/image.png"
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
actionID = "pythonResource"

python {
  local pythonScriptPath = "@(data.filepath("sd35api/1.0.0", "sd3_5.py"))" // [!code ++]
  local pythonScript = "@(read?("\(pythonScriptPath)")?.text)"             // [!code ++]

  script = """                                                             // [!code ++]
\(pythonScript)                                                            // [!code ++]
"""                                                                        // [!code ++]
  env {
    ["PROMPT"] = "@(request.params("q"))"                                  // [!code ++]
  }
...
}
```

### Linking to the API Response Resource

With the Python resource prepared, include it in the `requires` block of the API Response resource. This ensures the
script is executed as part of the workflow.

```js
actionID = "APIResponseResource"
requires {
  "pythonResource"                                                         // [!code ++]
}
```

Finally, the generated image file (`/tmp/image.png`) can be encoded as a `base64` PNG file and included in the API
response.

```json
local generatedFileBase64 = "@(read("/tmp/image.png").base64)"            // [!code ++]
local responseJson = new Mapping {                                        // [!code ++]
  ["file"] = "data:image/png;base64,\(generatedFileBase64)"               // [!code ++]
}                                                                         // [!code ++]

APIResponse {
...
  response {
    data {
        responseJson                                                      // [!code ++]
    }
  }
}
```

### Testing the AI Image Generator API

To test our newly created AI image generator API, we can use `curl`, which will output the file as a base64 string.

```bash
curl "http://localhost:3000/api/v1/image_generator?q=A+Teddy+Bear"

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
agentSettings {
    ...
    pythonPackages {
        "torch"
        "diffusers"
        "huggingface_hub[cli]"            // [!code ++]
    }
    ...
    args {
        ["HF_TOKEN"] = "secret"           // [!code ++]
    }
}
```

### Creating an `exec` Resource

Generate an `exec` resource using the following command:

```bash
kdeps scaffold sd35api exec
```

In the `exec` resource, include a script that logs in to Hugging Face using the `HF_TOKEN` from the `.env` file and
downloads the model. Additionally, set the cache directory to `/root/.kdeps/`, a shared folder for Kdeps, and create a
marker file (`/root/.kdeps/sd35-downloaded`) upon successful download.

```json
actionID = "execResource"
...
exec {
    command = """
    huggingface-cli login --token $HF_TOKEN                                                      // [!code ++]
    huggingface-cli download stabilityai/stable-diffusion-3.5-large --cache-dir /root/.kdeps/    // [!code ++]
    echo downloaded > /root/.kdeps/sd35-downloaded
    """
    env {
        ["HF_TOKEN"] = "\(read("env:HF_TOKEN"))"                                                 // [!code ++]
    }
    timeoutSeconds = 0
}
```

### Adding a `skipCondition`

To ensure the `exec` script runs only when necessary, add a `skipCondition`. This condition checks for the existence of
the `/root/.kdeps/sd35-downloaded` file. If the file exists, the script will be skipped.

```json
actionID = "execResource"
...
run {
    local stampFile = read?("file:/root/.kdeps/sd35-downloaded")?.base64?.isEmpty                // [!code ++]

    skipCondition {
        stampFile != null || stampFile != false                                                  // [!code ++]
    }
...
```
