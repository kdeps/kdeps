# Vision Models

This tutorial demonstrates how to use vision-capable LLMs in KDeps v2 to analyze images, extract information, and perform multimodal tasks.

## Prerequisites

- KDeps installed (see [Installation](../getting-started/installation.md))
- Ollama installed and running
- A vision model pulled: `ollama pull moondream:1.8b` or `ollama pull llava:7b`

## Overview

Vision models can process images along with text prompts. KDeps supports:
- Image file uploads via multipart form-data
- Local image files
- Multiple images in a single request
- Structured JSON responses

## Step 1: Install a Vision Model

Install a vision-capable model in Ollama:

```bash
# Lightweight and fast
ollama pull moondream:1.8b

# More capable
ollama pull llava:7b

# Best quality (slower)
ollama pull llava:13b
```

## Step 2: Create the Workflow

Create `workflow.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: vision
  description: Vision model example
  version: "1.0.0"
  targetActionId: visionResponse

settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/vision
        methods: [POST]
    cors:
      enableCors: true
      allowOrigins:
        - http://localhost:16395

  agentSettings:
    timezone: Etc/UTC
    pythonVersion: "3.12"
    models:
      - moondream:1.8b
      - llava:7b
```

## Step 3: Create the Vision LLM Resource

Create `resources/vision-llm.yaml`:

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: visionLLM
  name: Vision LLM

run:
  chat:
    model: moondream:1.8b
    role: user
    prompt: "{{ get('q', 'param') }}"
    files:
      # Get uploaded file path
      - "{{ get('file', 'filepath') }}"
    jsonResponse: true
    jsonResponseKeys:
      - description
      - objects
      - scene
```

</div>

**Key Points:**
- `files` field accepts an array of file paths
- Use `get('file', 'filepath')` for uploaded files
- `jsonResponse: true` ensures structured output

## Step 4: Create the Response Resource

Create `resources/vision-response.yaml`:

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: visionResponse
  name: Vision Response
  requires:
    - visionLLM

run:
  apiResponse:
    success: true
    response:
      query: get('q', 'param')
      analysis: get('visionLLM')
      file_info:
        filename: get('file', 'filename')
        filetype: get('file', 'filetype')
```

</div>

## Step 5: Test with Image Upload

Upload an image and query it:

```bash
curl -X POST 'http://localhost:16395/api/v1/vision?q=What%20is%20in%20this%20image?' \
  -F "file=@image.jpg"
```

Expected response:

```json
{
  "success": true,
  "data": {
    "query": "What is in this image?",
    "analysis": {
      "description": "A red panda sitting on a tree branch...",
      "objects": ["panda", "branch", "tree"],
      "scene": "forest"
    },
    "file_info": {
      "filename": "image.jpg",
      "filetype": "image/jpeg"
    }
  }
}
```

## Image Sources

### Uploaded Files

From multipart form-data uploads:

<div v-pre>

```yaml
files:
  - "{{ get('file', 'filepath') }}"
```

</div>

### Local Files

From the filesystem:

<div v-pre>

```yaml
files:
  - "./images/photo.jpg"
  - "{{ get('image_path') }}"
```

</div>

### Multiple Images

Process multiple images:

<div v-pre>

```yaml
files:
  - "{{ get('file1', 'filepath') }}"
  - "{{ get('file2', 'filepath') }}"
```

</div>

Or using file array:

<div v-pre>

```yaml
files:
  - "{{ get('file[]', 'filepath', 0) }}"
  - "{{ get('file[]', 'filepath', 1) }}"
```

</div>

## Supported Models

### moondream:1.8b

- **Best for**: Fast, lightweight queries
- **Use cases**: Simple descriptions, object detection
- **Speed**: Very fast
- **Quality**: Good for basic tasks

```yaml
model: moondream:1.8b
```

### llava:7b

- **Best for**: Balanced performance
- **Use cases**: Detailed descriptions, scene analysis
- **Speed**: Moderate
- **Quality**: High quality

```yaml
model: llava:7b
```

### llava:13b

- **Best for**: Best quality
- **Use cases**: Complex analysis, detailed descriptions
- **Speed**: Slower
- **Quality**: Highest quality

```yaml
model: llava:13b
```

## Use Cases

### Image Description

<div v-pre>

```yaml
run:
  chat:
    model: moondream:1.8b
    prompt: "Describe this image in detail"
    files:
      - "{{ get('file', 'filepath') }}"
```

</div>

### Object Detection

<div v-pre>

```yaml
run:
  chat:
    model: llava:7b
    prompt: "List all objects in this image"
    jsonResponse: true
    jsonResponseKeys:
      - objects
      - count
    files:
      - "{{ get('file', 'filepath') }}"
```

</div>

### Scene Analysis

<div v-pre>

```yaml
run:
  chat:
    model: llava:7b
    prompt: "Analyze the scene: location, time of day, weather, mood"
    jsonResponse: true
    jsonResponseKeys:
      - location
      - time_of_day
      - weather
      - mood
    files:
      - "{{ get('file', 'filepath') }}"
```

</div>

### Image Comparison

<div v-pre>

```yaml
run:
  chat:
    model: llava:13b
    prompt: "Compare these two images and describe the differences"
    files:
      - "{{ get('file1', 'filepath') }}"
      - "{{ get('file2', 'filepath') }}"
    jsonResponse: true
    jsonResponseKeys:
      - differences
      - similarities
```

</div>

### OCR Alternative

<div v-pre>

```yaml
run:
  chat:
    model: llava:7b
    prompt: "Extract all text from this image"
    jsonResponse: true
    jsonResponseKeys:
      - text
      - confidence
    files:
      - "{{ get('file', 'filepath') }}"
```

</div>

## Advanced Configuration

### With System Prompt

<div v-pre>

```yaml
run:
  chat:
    model: llava:7b
    scenario:
      - role: system
        prompt: "You are an expert image analyst. Provide detailed, accurate descriptions."
      - role: user
        prompt: "{{ get('q') }}"
    files:
      - "{{ get('file', 'filepath') }}"
```

</div>

### With Tools

Combine vision with function calling:

<div v-pre>

```yaml
run:
  chat:
    model: llava:7b
    prompt: "{{ get('q') }}"
    files:
      - "{{ get('file', 'filepath') }}"
    tools:
      - name: save_analysis
        description: Save the image analysis
        parameters:
          description:
            type: string
            description: The image description
```

</div>

## Image Formats

Supported image formats:
- JPEG (`.jpg`, `.jpeg`)
- PNG (`.png`)
- WebP (`.webp`)

## Performance Tips

1. **Choose the Right Model**: Use `moondream:1.8b` for speed, `llava:13b` for quality
2. **Image Size**: Smaller images process faster
3. **Batch Processing**: Process multiple images in one request
4. **Caching**: Cache results for repeated queries

## Error Handling

Handle errors gracefully:

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: visionLLM
  name: Vision LLM

run:
  validations:
    - info('filecount') > 0
    - get('file', 'filetype') in ['image/jpeg', 'image/png', 'image/webp']
  chat:
    model: moondream:1.8b
    prompt: "{{ get('q') }}"
    files:
      - "{{ get('file', 'filepath') }}"
  onError:
    apiResponse:
      success: false
      response:
        error: "Failed to process image"
```

</div>

## Complete Example

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: vision-demo
  version: "1.0.0"
  targetActionId: visionResponse

settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/vision
        methods: [POST]

  agentSettings:
    models:
      - moondream:1.8b

---
# resources/vision-llm.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: visionLLM
  name: Vision LLM

run:
  chat:
    model: moondream:1.8b
    prompt: "{{ get('q', 'param') }}"
    files:
      - "{{ get('file', 'filepath') }}"
    jsonResponse: true
    jsonResponseKeys:
      - description
      - objects

---
# resources/vision-response.yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: visionResponse
  name: Vision Response
  requires:
    - visionLLM

run:
  apiResponse:
    success: true
    response:
      query: get('q', 'param')
      analysis: get('visionLLM')
```

</div>

## Testing

### Single Image

```bash
curl -X POST 'http://localhost:16395/api/v1/vision?q=Describe%20this%20image' \
  -F "file=@photo.jpg"
```

### Multiple Images

```bash
curl -X POST 'http://localhost:16395/api/v1/vision?q=Compare%20these%20images' \
  -F "file[]=@image1.jpg" \
  -F "file[]=@image2.jpg"
```

## Troubleshooting

### Model Not Found

- Ensure the model is pulled: `ollama pull moondream:1.8b`
- Check model name matches exactly
- Verify Ollama is running

### Image Not Processed

- Check file format is supported (JPEG, PNG, WebP)
- Verify file path is correct
- Ensure file was uploaded successfully

### Slow Processing

- Use smaller images
- Try a faster model (`moondream:1.8b`)
- Check system resources

## Next Steps

- **File Uploads**: Learn about [file upload handling](file-upload)
- **Tools**: Combine vision with [function calling](../concepts/tools)
- **Batch Processing**: Process multiple images with [items iteration](../concepts/items)
- **LLM Configuration**: See [LLM resource](../resources/llm) for advanced options

## Related Documentation

- [LLM Resource](../resources/llm) - Complete LLM configuration reference
- [File Upload](file-upload) - Handling file uploads
- [Unified API](../concepts/unified-api) - Accessing file data
- [Tools](../concepts/tools) - Function calling with vision
