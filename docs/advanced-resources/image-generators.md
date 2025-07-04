---
outline: deep
---

# AI Image Generators

Kdeps enables AI image generation through Python resources that execute image generation models such as Stable Diffusion. While Kdeps doesn't directly generate images using LLMs, it provides a powerful framework for integrating advanced image generation models through Python scripts, with optional Anaconda support and compatibility with GGUF models from Hugging Face.

## Overview

AI image generation in Kdeps works by:

1. **Python Resource**: Executes image generation scripts using models like Stable Diffusion
2. **Model Integration**: Supports Hugging Face models, GGUF format, and custom models
3. **API Integration**: Provides RESTful endpoints for image generation requests
4. **File Management**: Handles image output, encoding, and response formatting
5. **Authentication**: Supports Hugging Face token authentication for gated models

## Basic Image Generation Setup

### Workflow Configuration

Configure your `workflow.pkl` file with the necessary Python packages and API endpoints:

```apl
Name = "sd35api"
Version = "1.0.0"
AgentID = "sd35api"
Description = "Stable Diffusion 3.5 Image Generation API"
TargetActionID = "APIResponseResource"

Settings {
    APIServerMode = true
    APIServer {
        HostIP = "127.0.0.1"
        PortNum = 3000
        Routes {
            new {
                Path = "/api/v1/image_generator"
                Method = "POST"
            }
        }
    }
    
    AgentSettings {
        PythonPackages {
            "torch"
            "diffusers[torch]"
            "transformers"
            "accelerate"
            "huggingface_hub[cli]"
        }
        
        Packages {
            "python3-dev"
            "build-essential"
        }
        
        Models {}  // No LLM models needed for image generation
    }
}
```

### Image Generation Script

Create a Python script for image generation. Save this as `data/sd3_5.py`:

```python
import os
import torch
from diffusers import StableDiffusion3Pipeline
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def generate_image():
    """Generate an image using Stable Diffusion 3.5"""
    
    # Get configuration from environment variables
    prompt = os.getenv("PROMPT", "A capybara holding a sign that reads 'Hello World'")
    output_path = os.getenv("OUTPUT_PATH", "/tmp/image.png")
    num_inference_steps = int(os.getenv("INFERENCE_STEPS", "28"))
    guidance_scale = float(os.getenv("GUIDANCE_SCALE", "3.5"))
    
    logger.info(f"Generating image with prompt: {prompt}")
    
    # Clean up existing file
    if os.path.exists(output_path):
        os.remove(output_path)
        logger.info(f"Removed existing file: {output_path}")
    
    try:
        # Load the Stable Diffusion model
        logger.info("Loading Stable Diffusion 3.5 model...")
        pipe = StableDiffusion3Pipeline.from_pretrained(
            "stabilityai/stable-diffusion-3.5-large", 
            torch_dtype=torch.bfloat16,
            cache_dir="/.kdeps/huggingface"
        )
        
        # Move to GPU if available
        device = "cuda" if torch.cuda.is_available() else "cpu"
        pipe = pipe.to(device)
        logger.info(f"Model loaded on device: {device}")
        
        # Generate the image
        logger.info("Generating image...")
        image = pipe(
            prompt,
            num_inference_steps=num_inference_steps,
            guidance_scale=guidance_scale,
        ).images[0]
        
        # Save the generated image
        image.save(output_path)
        logger.info(f"Image saved to: {output_path}")
        
        return True
        
    except Exception as e:
        logger.error(f"Error generating image: {str(e)}")
        # Create a simple error indicator file
        with open(output_path + ".error", "w") as f:
            f.write(str(e))
        return False

if __name__ == "__main__":
    generate_image()
```

### Python Resource Configuration

Configure the Python resource to execute the image generation script:

```apl
ActionID = "pythonResource"
Name = "Image Generation Python Script"
Description = "Executes Stable Diffusion image generation"
Category = "ai"

Run {
    RestrictToHTTPMethods { "POST" }
    RestrictToRoutes { "/api/v1/image_generator" }
    AllowedParams { "prompt"; "steps"; "guidance" }
    
    PreflightCheck {
        Validations { 
            "@(request.data('prompt'))" != ""
            "@(request.data('prompt').length())" > 3
            "@(request.data('prompt').length())" < 500
        }
        Retry = false
        RetryTimes = 1
    }
    
    Python {
        local pythonScriptPath = "@(data.filepath("sd35api/1.0.0", "sd3_5.py"))"
        local pythonScript = "@(read?("\(pythonScriptPath)")?.text)"
        
        Script = """
\(pythonScript)
"""
        
        Env {
            ["PROMPT"] = "@(request.data('prompt'))"
            ["OUTPUT_PATH"] = "/tmp/generated_image.png"
            ["INFERENCE_STEPS"] = "@(request.data('steps') ?? '28')"
            ["GUIDANCE_SCALE"] = "@(request.data('guidance') ?? '3.5')"
        }
        
        TimeoutDuration = 5.min
    }
}
```

### API Response Resource

Configure the response resource to return the generated image:

```apl
ActionID = "APIResponseResource"
Name = "Image Generation API Response"
Description = "Returns generated images as base64-encoded data"
Category = "api"

Requires { "pythonResource" }

Run {
    local imagePath = "/tmp/generated_image.png"
    local errorPath = "/tmp/generated_image.png.error"
    
    local imageExists = "@(read?("\(imagePath)")?.base64)"
    local errorExists = "@(read?("\(errorPath)")?.text)"
    
    local generatedFileBase64 = "@(read("\(imagePath)").base64)"
    
    local successResponse = new Mapping {
        ["success"] = true
        ["image"] = "data:image/png;base64,\(generatedFileBase64)"
        ["prompt"] = "@(request.data('prompt'))"
        ["timestamp"] = "@(now())"
    }
    
    local errorResponse = new Mapping {
        ["success"] = false
        ["error"] = "\(errorExists ?? 'Unknown error occurred')"
        ["prompt"] = "@(request.data('prompt'))"
        ["timestamp"] = "@(now())"
    }
    
    APIResponse {
        Response {
            Data {
                if (imageExists != null) successResponse else errorResponse
            }
        }
    }
}
```

## Authenticated Model Access

Many advanced models require Hugging Face authentication. Here's how to set it up securely:

### Environment Configuration

Create a `.env` file in your project root (outside the agent directory):

```bash
# .env file
HF_TOKEN=hf_your_token_here
CACHE_DIR=/.kdeps/huggingface
```

### Workflow Authentication Setup

Update your workflow configuration to handle authentication:

```apl
AgentSettings {
    PythonPackages {
        "torch"
        "diffusers[torch]"
        "transformers"
        "accelerate"
        "huggingface_hub[cli]"
    }
    
    Args {
        ["HF_TOKEN"] = ""
        ["CACHE_DIR"] = "/.kdeps/huggingface"
    }
    
    Env {
        ["HUGGINGFACE_HUB_CACHE"] = "/.kdeps/huggingface"
        ["HF_HOME"] = "/.kdeps/huggingface"
    }
}
```

### Model Download Resource

Create an exec resource to handle model authentication and download:

```apl
ActionID = "modelDownloadResource"
Name = "Model Download and Authentication"
Description = "Downloads and authenticates Hugging Face models"
Category = "setup"

Run {
    local downloadMarker = "/.kdeps/sd35-model-ready"
    local markerExists = "@(read?("\(downloadMarker)")?.text)"
    
    SkipCondition {
        markerExists != null
    }
    
    Exec {
        Command = """
#!/bin/bash
set -e

echo "Authenticating with Hugging Face..."
huggingface-cli login --token $HF_TOKEN

echo "Downloading Stable Diffusion 3.5 model..."
huggingface-cli download stabilityai/stable-diffusion-3.5-large \
    --cache-dir $CACHE_DIR \
    --local-dir-use-symlinks False

echo "Model download completed successfully"
echo "$(date)" > /.kdeps/sd35-model-ready
"""
        
        Env {
            ["HF_TOKEN"] = "@(read("env:HF_TOKEN"))"
            ["CACHE_DIR"] = "/.kdeps/huggingface"
        }
        
        TimeoutDuration = 0.s  // No timeout for large downloads
    }
}
```

### Updated Python Resource with Authentication

Modify the Python resource to depend on model download:

```apl
ActionID = "pythonResource"
Name = "Authenticated Image Generation"
Description = "Generates images using authenticated models"
Category = "ai"

Requires { "modelDownloadResource" }

Run {
    Python {
        Script = """
import os
import torch
from diffusers import StableDiffusion3Pipeline
from huggingface_hub import login
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Authenticate if token is available
hf_token = os.getenv("HF_TOKEN")
if hf_token:
    login(token=hf_token)
    logger.info("Authenticated with Hugging Face")

# Load model from cache
cache_dir = os.getenv("CACHE_DIR", "/.kdeps/huggingface")
model_path = "stabilityai/stable-diffusion-3.5-large"

pipe = StableDiffusion3Pipeline.from_pretrained(
    model_path,
    torch_dtype=torch.bfloat16,
    cache_dir=cache_dir,
    local_files_only=True  # Use cached model
)

# Rest of image generation code...
"""
        
        Env {
            ["HF_TOKEN"] = "@(read("env:HF_TOKEN"))"
            ["CACHE_DIR"] = "/.kdeps/huggingface"
            ["PROMPT"] = "@(request.data('prompt'))"
        }
    }
}
```

## Advanced Configuration

### Multiple Model Support

Configure support for multiple image generation models:

```apl
Python {
    Script = """
import os
from diffusers import StableDiffusion3Pipeline, StableDiffusionXLPipeline

model_name = os.getenv("MODEL", "sd3")
models = {
    "sd3": "stabilityai/stable-diffusion-3.5-large",
    "sdxl": "stabilityai/stable-diffusion-xl-base-1.0",
    "turbo": "stabilityai/sd-turbo"
}

if model_name in models:
    model_path = models[model_name]
    if model_name == "sd3":
        pipe = StableDiffusion3Pipeline.from_pretrained(model_path)
    else:
        pipe = StableDiffusionXLPipeline.from_pretrained(model_path)
else:
    raise ValueError(f"Unsupported model: {model_name}")

# Generate image with selected model...
"""
    
    Env {
        ["MODEL"] = "@(request.data('model') ?? 'sd3')"
        ["PROMPT"] = "@(request.data('prompt'))"
    }
}
```

### Image Processing Pipeline

Add image post-processing capabilities:

```python
import os
from PIL import Image, ImageEnhance, ImageFilter

def enhance_image(image_path, output_path):
    """Apply post-processing enhancements to generated image"""
    
    with Image.open(image_path) as img:
        # Apply enhancements based on environment variables
        brightness = float(os.getenv("BRIGHTNESS", "1.0"))
        contrast = float(os.getenv("CONTRAST", "1.0"))
        sharpness = float(os.getenv("SHARPNESS", "1.0"))
        
        if brightness != 1.0:
            enhancer = ImageEnhance.Brightness(img)
            img = enhancer.enhance(brightness)
            
        if contrast != 1.0:
            enhancer = ImageEnhance.Contrast(img)
            img = enhancer.enhance(contrast)
            
        if sharpness != 1.0:
            enhancer = ImageEnhance.Sharpness(img)
            img = enhancer.enhance(sharpness)
        
        # Apply blur if requested
        blur_radius = float(os.getenv("BLUR", "0"))
        if blur_radius > 0:
            img = img.filter(ImageFilter.GaussianBlur(radius=blur_radius))
        
        # Save enhanced image
        img.save(output_path, "PNG", quality=95)
        
    return True
```

## Testing and Usage

### API Testing

Test your image generation API using curl:

```bash
# Basic image generation
curl -X POST "http://localhost:3000/api/v1/image_generator" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "A beautiful sunset over mountains"}'

# Advanced parameters
curl -X POST "http://localhost:3000/api/v1/image_generator" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "A futuristic city with flying cars",
    "steps": 50,
    "guidance": 7.5,
    "model": "sd3"
  }'
```

### Expected Response Format

```json
{
  "success": true,
  "response": {
    "data": [
      {
        "success": true,
        "image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
        "prompt": "A beautiful sunset over mountains",
        "timestamp": "2024-01-15T10:30:00Z"
      }
    ]
  },
  "errors": []
}
```

### Python Client Example

```python
import requests
import base64
from PIL import Image
from io import BytesIO

def generate_image(prompt, api_url="http://localhost:3000/api/v1/image_generator"):
    """Generate an image using the Kdeps API"""
    
    response = requests.post(api_url, json={"prompt": prompt})
    
    if response.status_code == 200:
        data = response.json()
        if data["success"]:
            # Extract base64 image data
            image_data = data["response"]["data"][0]["image"]
            # Remove data URL prefix
            base64_data = image_data.split(",")[1]
            # Decode and open image
            image_bytes = base64.b64decode(base64_data)
            image = Image.open(BytesIO(image_bytes))
            return image
    
    return None

# Usage
image = generate_image("A cat wearing a space helmet")
if image:
    image.save("generated_image.png")
```

## Performance Optimization

### GPU Configuration

Optimize for different GPU configurations:

```python
import torch

def get_optimal_device_config():
    """Get optimal device configuration based on available hardware"""
    
    if torch.cuda.is_available():
        device = "cuda"
        # Use mixed precision for memory efficiency
        torch_dtype = torch.float16
        enable_memory_efficient_attention = True
    elif hasattr(torch.backends, 'mps') and torch.backends.mps.is_available():
        device = "mps"  # Apple Silicon
        torch_dtype = torch.float16
        enable_memory_efficient_attention = True
    else:
        device = "cpu"
        torch_dtype = torch.float32
        enable_memory_efficient_attention = False
    
    return {
        "device": device,
        "torch_dtype": torch_dtype,
        "enable_memory_efficient_attention": enable_memory_efficient_attention
    }
```

### Memory Management

Implement memory optimization strategies:

```python
import gc
import torch

def optimize_memory():
    """Clean up GPU memory after generation"""
    if torch.cuda.is_available():
        torch.cuda.empty_cache()
    gc.collect()

# Use in generation script
def generate_with_cleanup():
    try:
        # Generate image
        image = pipe(prompt, **generation_params).images[0]
        image.save(output_path)
    finally:
        # Always clean up memory
        optimize_memory()
```

## Best Practices

### Error Handling

Implement comprehensive error handling:

```python
import logging
from contextlib import contextmanager

@contextmanager
def error_handler(operation_name):
    """Context manager for consistent error handling"""
    try:
        logging.info(f"Starting {operation_name}")
        yield
        logging.info(f"Completed {operation_name}")
    except Exception as e:
        logging.error(f"Error in {operation_name}: {str(e)}")
        # Create error marker file
        with open("/tmp/generation_error.txt", "w") as f:
            f.write(f"{operation_name}: {str(e)}")
        raise

# Usage
with error_handler("model loading"):
    pipe = StableDiffusion3Pipeline.from_pretrained(model_path)
```

### Security Considerations

1. **Input Validation**: Always validate prompts and parameters
2. **File System Security**: Restrict file access to designated directories
3. **Resource Limits**: Set appropriate timeouts and memory limits
4. **Authentication**: Secure API endpoints with proper authentication

### Production Deployment

For production environments:

1. **Model Caching**: Pre-download models during container build
2. **Health Checks**: Implement health check endpoints
3. **Monitoring**: Add comprehensive logging and metrics
4. **Scaling**: Use appropriate resource allocation and auto-scaling
5. **Rate Limiting**: Implement rate limiting to prevent abuse

## Troubleshooting

### Common Issues

**Model Download Failures:**
```bash
# Check Hugging Face authentication
huggingface-cli whoami

# Verify token permissions
huggingface-cli auth show
```

**GPU Memory Issues:**
```python
# Monitor GPU memory usage
import torch
print(f"GPU memory allocated: {torch.cuda.memory_allocated() / 1024**2:.2f} MB")
print(f"GPU memory cached: {torch.cuda.memory_reserved() / 1024**2:.2f} MB")
```

**Generation Timeouts:**
- Increase `TimeoutDuration` for complex prompts
- Use smaller models for faster generation
- Optimize inference steps and guidance scale

## Next Steps

- **[Python Resource](../core-resources/python.md)**: Learn more about Python resource configuration
- **[Multimodal Resources](./multimodal.md)**: Combine image generation with other modalities
- **[Tools Integration](./tools.md)**: Create tools for image processing workflows
- **[API Response](../core-resources/response.md)**: Advanced API response formatting

AI image generation with Kdeps provides a powerful foundation for building creative AI applications. Experiment with different models, optimize for your hardware, and implement proper error handling for production-ready image generation services.

