# File Upload Example

This example demonstrates file upload handling in KDeps v2 with multipart form-data support.

## Features

- ✅ Single file uploads
- ✅ Multiple file uploads
- ✅ File metadata access (count, names, types)
- ✅ File content, path, and MIME type access via `get()`
- ✅ File info via `info()` functions

## Run Locally

```bash
# From examples/file-upload directory
kdeps run workflow.yaml --dev

# Or from root
kdeps run examples/file-upload/workflow.yaml --dev
```

## Test

### Single File Upload

```bash
curl -X POST http://localhost:3000/api/v1/upload \
  -F "file=@example.txt"
```

### Multiple File Upload

```bash
curl -X POST http://localhost:3000/api/v1/upload \
  -F "file[]=@file1.txt" \
  -F "file[]=@file2.pdf" \
  -F "file[]=@file3.jpg"
```

### Response

```json
{
  "success": true,
  "data": {
    "message": "File processed successfully",
    "file_count": "2",
    "files": ["file1.txt", "file2.pdf"],
    "file_types": ["text/plain", "application/pdf"],
    "file_info": [
      {
        "filename": "...",
        "path": "/tmp/kdeps-uploads/...",
        "type": "text/plain"
      }
    ]
  }
}
```

## Structure

```
file-upload/
├── workflow.yaml              # Main workflow configuration
└── resources/
    └── file-processor.yaml   # File processing resource
```

## Key Concepts

### File Access Functions

**`info()` for metadata**:
- `info('filecount')` - Number of uploaded files
- `info('files')` - Array of file paths
- `info('filetypes')` - Array of MIME types

**`get()` for file data**:
- `get('filename', 'file')` - File content
- `get('filename', 'filepath')` - File path
- `get('filename', 'filetype')` - MIME type

### Auto-Detection

Files uploaded with field name `file` or `file[]` are automatically available via `get('file')`.

### File Field Names

- `file` - Single file
- `file[]` - Multiple files
- `files` - Alternative name for multiple files
- Any field name - Automatically detected
