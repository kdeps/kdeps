# File Upload Processing

This tutorial demonstrates how to handle file uploads in KDeps v2, including single and multiple file uploads, file metadata access, and processing uploaded files.

## Prerequisites

- KDeps installed (see [Installation](../getting-started/installation))
- Basic understanding of HTTP multipart form-data

## Overview

KDeps automatically handles file uploads from multipart form-data requests. Files are temporarily stored and accessible via the unified API using `get()` and `info()` functions.

## Step 1: Create the Workflow

Create `workflow.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Workflow

metadata:
  name: file-upload
  description: File upload handling example
  version: "1.0.0"
  targetActionId: fileProcessor

settings:
  apiServerMode: true
  apiServer:
    hostIp: "127.0.0.1"
    portNum: 16395
    routes:
      - path: /api/v1/upload
        methods: [POST]
    cors:
      enableCors: true
      allowOrigins:
        - http://localhost:16395

  agentSettings:
    timezone: Etc/UTC
    pythonVersion: "3.12"
```

## Step 2: Create the File Processor Resource

Create `resources/file-processor.yaml`:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: fileProcessor
  name: File Processor

run:
  apiResponse:
    success: true
    response:
      message: "File processed successfully"
      file_count: info('filecount')
      files: info('files')
      file_types: info('filetypes')
      file_info:
        - filename: get('file', 'filename')
          path: get('file', 'filepath')
          type: get('file', 'filetype')
```

## Step 3: Understanding File Access

### Using `info()` for Metadata

The `info()` function provides file metadata:

```yaml
file_count: info('filecount')    # Number of uploaded files
files: info('files')              # Array of file paths
file_types: info('filetypes')     # Array of MIME types
```

### Using `get()` for File Data

The `get()` function accesses file content and properties:

```yaml
# File content (as string or bytes)
content: get('filename', 'file')

# File path (temporary storage location)
path: get('filename', 'filepath')

# MIME type
type: get('filename', 'filetype')
```

## Step 4: Single File Upload

### Upload a File

```bash
curl -X POST http://localhost:16395/api/v1/upload \
  -F "file=@example.txt"
```

### Access the File

```yaml
run:
  apiResponse:
    response:
      content: get('file', 'file')
      path: get('file', 'filepath')
      type: get('file', 'filetype')
```

## Step 5: Multiple File Upload

### Upload Multiple Files

```bash
curl -X POST http://localhost:16395/api/v1/upload \
  -F "file[]=@file1.txt" \
  -F "file[]=@file2.pdf" \
  -F "file[]=@file3.jpg"
```

### Process Multiple Files

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: fileProcessor
  name: File Processor

run:
  apiResponse:
    success: true
    response:
      file_count: info('filecount')
      files:
        - filename: get('file1.txt', 'filename')
          content: get('file1.txt', 'file')
        - filename: get('file2.pdf', 'filename')
          content: get('file2.pdf', 'file')
```

## Step 6: Processing Files with Python

Process uploaded files using the Python resource:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: fileProcessor
  name: File Processor

run:
  python:
    script: |
      import json
      from pathlib import Path
      
      # Get file path
      file_path = get('file', 'filepath')
      
      # Read and process file
      with open(file_path, 'r') as f:
          content = f.read()
      
      # Process content
      processed = content.upper()
      
      return {
          'original': content,
          'processed': processed,
          'length': len(content)
      }
```

## Step 7: File Validation

Add validation to check file properties:

```yaml
run:
  validations:
    - info('filecount') > 0
    - info('filecount') <= 5
    - get('file', 'filetype') == 'text/plain'
  apiResponse:
    success: true
    response:
      message: "File validated and processed"
```

## Step 8: Processing Images

Process image files with vision models:

<div v-pre>

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: imageProcessor
  name: Image Processor
  requires:
    - visionLLM

run:
  chat:
    model: moondream:1.8b
    prompt: "Describe this image"
    files:
      - "{{ get('file', 'filepath') }}"
    jsonResponse: true
    jsonResponseKeys:
      - description
      - objects
```

</div>

## File Field Names

KDeps supports multiple field names for file uploads:

- `file` - Single file
- `file[]` - Multiple files (recommended)
- `files` - Alternative name for multiple files
- Any field name - Automatically detected

### Custom Field Names

```bash
# Upload with custom field name
curl -X POST http://localhost:16395/api/v1/upload \
  -F "document=@report.pdf"
```

Access with the field name:

```yaml
content: get('document', 'file')
path: get('document', 'filepath')
```

## Complete Example

Here's a complete file processor that handles text files:

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: fileProcessor
  name: File Processor

run:
  validations:
    - info('filecount') > 0
    - get('file', 'filetype') == 'text/plain'
  
  python:
    script: |
      from pathlib import Path
      import json
      
      # Get file path
      file_path = get('file', 'filepath')
      
      # Read file
      with open(file_path, 'r') as f:
          content = f.read()
      
      # Process: count words and lines
      words = len(content.split())
      lines = len(content.splitlines())
      
      return {
          'filename': get('file', 'filename'),
          'word_count': words,
          'line_count': lines,
          'size': len(content)
      }
```

## Response Format

Example response:

```json
{
  "success": true,
  "data": {
    "message": "File processed successfully",
    "file_count": "2",
    "files": [
      "/tmp/kdeps-uploads/abc123/file1.txt",
      "/tmp/kdeps-uploads/abc123/file2.txt"
    ],
    "file_types": ["text/plain", "text/plain"],
    "file_info": [
      {
        "filename": "file1.txt",
        "path": "/tmp/kdeps-uploads/abc123/file1.txt",
        "type": "text/plain"
      }
    ]
  }
}
```

## File Cleanup

Files are automatically cleaned up after the request completes. Temporary files are stored in the system temp directory and removed when:

- The request completes successfully
- An error occurs
- The workflow shuts down

## Best Practices

1. **Validate File Types**: Check MIME types before processing
2. **Limit File Count**: Validate `info('filecount')` to prevent abuse
3. **Process Efficiently**: Use Python resources for complex processing
4. **Handle Errors**: Add error handling for file operations
5. **Use Appropriate Field Names**: Use `file[]` for multiple files

## Troubleshooting

### Files Not Detected

- Ensure `Content-Type: multipart/form-data` header is set
- Check field name matches (`file`, `file[]`, or custom name)
- Verify file size is within limits (default: 10MB)

### File Access Errors

- Use `get('filename', 'filepath')` for file paths
- Use `get('filename', 'file')` for file content
- Check file exists with `info('filecount')`

## Next Steps

- **Vision Processing**: Learn about [vision models](vision) for image analysis
- **Batch Processing**: Use [items iteration](../concepts/items) for multiple files
- **Storage**: Integrate with cloud storage for permanent file storage
- **Validation**: Add comprehensive file validation

## Related Documentation

- [Unified API](../concepts/unified-api) - Understanding `get()` and `info()`
- [Request Object](../concepts/request-object) - File access methods (`request.file()`, `request.filepath()`, etc.)
- [Python Resource](../resources/python) - Processing files with Python
- [Vision Models](vision) - Image processing with vision LLMs
- [Workflow Configuration](../configuration/workflow) - API server settings
