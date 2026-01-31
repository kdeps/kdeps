# Vision Model Example

This example demonstrates vision model integration in KDeps v2, allowing you to analyze images using multimodal LLMs.

## Features

- ✅ Image file upload via multipart form-data
- ✅ Vision model integration (moondream, llava, etc.)
- ✅ Image analysis with natural language queries
- ✅ JSON structured responses
- ✅ File metadata access

## Prerequisites

You need a vision-capable model installed in Ollama:

```bash
# Install a vision model (choose one)
ollama pull moondream:1.8b    # Lightweight, fast
ollama pull llava:7b          # More capable
ollama pull llava:13b         # Best quality
```

## Run Locally

```bash
# From examples/vision directory
kdeps run workflow.yaml --dev

# Or from root
kdeps run examples/vision/workflow.yaml --dev
```

## Test

### Upload Image and Query

```bash
curl -X POST 'http://localhost:3000/api/v1/vision?q=What%20is%20in%20this%20image?' \
  -F "file=@image.jpg"
```

### Multiple Images

```bash
curl -X POST 'http://localhost:3000/api/v1/vision?q=Compare%20these%20images' \
  -F "file[]=@image1.jpg" \
  -F "file[]=@image2.jpg"
```

### Response

```json
{
  "success": true,
  "data": {
    "query": "What is in this image?",
    "analysis": {
      "description": "A red panda sitting on a branch...",
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

## Structure

```
vision/
├── workflow.yaml              # Main workflow configuration
└── resources/
    ├── vision-llm.yaml       # Vision LLM resource
    └── vision-response.yaml  # Response handler
```

## Key Concepts

### Vision Model Configuration

**Files field in ChatConfig**:
```yaml
run:
  chat:
    model: "moondream:1.8b"
    prompt: "{{ get('q') }}"
    files:
      - "{{ get('file', 'file') }}"  # Image file from upload
```

### Supported Models

- **moondream:1.8b** - Lightweight, fast, good for simple queries
- **llava:7b** - Balanced performance and capability
- **llava:13b** - Best quality, slower

### File Handling

Images can come from:
1. **Uploaded files**: `get('file', 'file')` - from multipart upload
2. **Local files**: `get('path/to/image.jpg')` - from filesystem
3. **Multiple images**: Array of file paths in `files` field

### Multimodal Content

The LLM executor automatically:
- Loads image files
- Encodes them as base64
- Formats them for Ollama's multimodal API
- Combines with text prompt

## Example Use Cases

1. **Image Description**: "Describe what you see in this image"
2. **Object Detection**: "List all objects in this image"
3. **Scene Analysis**: "What type of scene is this?"
4. **Image Comparison**: "Compare these two images"
5. **OCR Alternative**: "Extract text from this image"

## Notes

- Vision models work best with clear, specific queries
- Image size affects processing time (larger = slower)
- Supported formats: JPEG, PNG, WebP
- Multiple images are processed in sequence
