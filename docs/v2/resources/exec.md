# Exec Resource

The Exec resource enables execution of shell commands and scripts for system operations, file manipulation, and external tool integration.

## Basic Usage

```yaml
apiVersion: kdeps.io/v1
kind: Resource

metadata:
  actionId: execResource
  name: System Command

run:
  exec:
    command: "echo 'Hello, World!'"
    timeoutDuration: 30s
```

## Configuration Options

| Option | Description |
|--------|-------------|
| `command` | The shell command to execute. Supports multiline scripts. |
| `args` | List of arguments to pass to the command. |
| `workingDir` | The directory in which the command should execute. |
| `env` | Map of environment variables specific to this execution. |
| `timeoutDuration` | Maximum time allowed for execution (e.g., "30s", "1m"). |

## Command Execution

### Simple Command

```yaml
run:
  exec:
    command: "date"
    timeoutDuration: 10s
```

### Command with Arguments

```yaml
run:
  exec:
    command: "your-command"    # The command to execute
    args:                      # Optional: command arguments
      - "--flag"
      - "value"
    workingDir: "/tmp"         # Optional: working directory
    env:                       # Optional: resource-specific env vars
      TEMP_VAR: "value"
    timeoutDuration: 30s       # Execution timeout
```

### Multi-line Script

```yaml
run:
  exec:
    command: |
      echo "Starting process..."
      date
      uname -a
      echo "Done!"
    timeoutDuration: 30s
```

### With Interpolation

<div v-pre>

```yaml
run:
  exec:
    command: "curl -s https://api.example.com/users/{{ get('user_id') }}"
    timeoutDuration: 30s
```

</div>

## Examples

### System Information

```yaml
metadata:
  actionId: systemInfo

run:
  exec:
    command: |
      echo '{"hostname": "'$(hostname)'", "os": "'$(uname -s)'", "kernel": "'$(uname -r)'", "date": "'$(date -Iseconds)'"}'
    timeoutDuration: 10s
```

### File Operations

<div v-pre>

```yaml
metadata:
  actionId: fileOps

run:
  exec:
    command: |
      # Create directory
      mkdir -p /tmp/processing

      # Copy uploaded file
      cp "{{ get('file', 'filepath') }}" /tmp/processing/

      # Get file info
      FILE="/tmp/processing/$(basename "{{ get('file', 'filepath') }}")"
      SIZE=$(stat -f%z "$FILE" 2>/dev/null || stat -c%s "$FILE")
      MD5=$(md5sum "$FILE" | cut -d' ' -f1)

      echo "{\"path\": \"$FILE\", \"size\": $SIZE, \"md5\": \"$MD5\"}"
    timeoutDuration: 60s
```

</div>

### Git Operations

```yaml
metadata:
  actionId: gitInfo

run:
  exec:
    command: |
      cd /app
      BRANCH=$(git rev-parse --abbrev-ref HEAD)
      COMMIT=$(git rev-parse --short HEAD)
      AUTHOR=$(git log -1 --format='%an')
      MESSAGE=$(git log -1 --format='%s')

      echo "{\"branch\": \"$BRANCH\", \"commit\": \"$COMMIT\", \"author\": \"$AUTHOR\", \"message\": \"$MESSAGE\"}"
    timeoutDuration: 30s
```

### Process External Tools

<div v-pre>

```yaml
metadata:
  actionId: processImage

run:
  exec:
    command: |
      INPUT="{{ get('file', 'filepath') }}"
      OUTPUT="/tmp/processed_$(date +%s).jpg"

      # Resize with ImageMagick
      convert "$INPUT" -resize 800x600 -quality 85 "$OUTPUT"

      # Get dimensions
      DIMS=$(identify -format '{"width":%w,"height":%h}' "$OUTPUT")

      echo "{\"output\": \"$OUTPUT\", \"dimensions\": $DIMS}"
    timeoutDuration: 120s
```

</div>

### FFmpeg Video Processing

```yaml
metadata:
  actionId: extractAudio

run:
  exec:
    command: |
      INPUT="{{ get('video', 'filepath') }}"
      OUTPUT="/tmp/audio_$(date +%s).mp3"

      # Extract audio
      ffmpeg -i "$INPUT" -vn -acodec libmp3lame -q:a 2 "$OUTPUT" 2>/dev/null

      # Get duration
      DURATION=$(ffprobe -v error -show_entries format=duration -of default=noprint_wrappers=1:nokey=1 "$OUTPUT")

      echo "{\"audio_file\": \"$OUTPUT\", \"duration_seconds\": $DURATION}"
    timeoutDuration: 300s
```

### OCR with Tesseract

<div v-pre>

```yaml
metadata:
  actionId: ocrProcess

run:
  exec:
    command: |
      INPUT="{{ get('file', 'filepath') }}"
      OUTPUT="/tmp/ocr_result.txt"

      # Run OCR
      tesseract "$INPUT" "/tmp/ocr_result" -l eng 2>/dev/null

      # Read result and escape for JSON
      TEXT=$(cat "$OUTPUT" | jq -Rs .)

      echo "{\"text\": $TEXT}"
    timeoutDuration: 60s
```

</div>

### Docker Operations

```yaml
metadata:
  actionId: dockerInfo

run:
  exec:
    command: |
      CONTAINERS=$(docker ps --format '{{.Names}}' | wc -l | tr -d ' ')
      IMAGES=$(docker images --format '{{.Repository}}' | wc -l | tr -d ' ')

      echo "{\"running_containers\": $CONTAINERS, \"images\": $IMAGES}"
    timeoutDuration: 30s
```

### Curl API Call

<div v-pre>

```yaml
metadata:
  actionId: apiCall

run:
  exec:
    command: |
      RESPONSE=$(curl -s -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $API_TOKEN" \
        -d '{"query": "{{ get('q') }}"}' \
        https://api.example.com/search)

      echo "$RESPONSE"
    timeoutDuration: 30s
```

</div>

## Output Handling

The exec resource captures stdout as the result. For structured output, echo JSON:

```yaml
run:
  exec:
    command: |
      # Process something
      RESULT="success"
      COUNT=42

      # Output JSON
      echo "{\"status\": \"$RESULT\", \"count\": $COUNT}"
```

Access in other resources:

```yaml
metadata:
  requires: [execResource]

run:
  apiResponse:
    response:
      result: get('execResource')
      status: get('execResource').status
```

## Environment Variables

Environment variables from workflow settings are available:

```yaml
# In workflow.yaml
settings:
  agentSettings:
    env:
      API_TOKEN: "secret-token"
      DEBUG: "true"
```

```yaml
# In resource
run:
  exec:
    command: |
      echo "Token: $API_TOKEN"
      if [ "$DEBUG" = "true" ]; then
        echo "Debug mode enabled"
      fi
```

## Accessing Output Details

Access stdout, stderr, and exit codes from other resources:

```yaml
metadata:
  requires: [execResource]

run:
  expr:
    # Check if command succeeded
    - set('command_success', exec.exitCode('execResource') == 0)
    - set('error_output', exec.stderr('execResource'))
  
  apiResponse:
    response:
      output: get('execResource')  # stdout (default)
      errors: get('error_output')
      success: get('command_success')
```

See [Unified API](../concepts/unified-api.md#resource-specific-accessors) for details.

## Error Handling

Check command exit codes:

```yaml
run:
  exec:
    command: |
      if ! command -v ffmpeg &> /dev/null; then
        echo '{"error": "ffmpeg not installed"}' >&2  # Write to stderr
        exit 1
      fi

      # Continue with processing...
      echo '{"status": "success"}'
    timeoutDuration: 30s
```

**Note**: Errors written to stderr are accessible via `exec.stderr('resourceId')` in other resources.

## Installed Packages

Configure OS packages in your workflow:

```yaml
# workflow.yaml
settings:
  agentSettings:
    osPackages:
      - ffmpeg
      - imagemagick
      - tesseract-ocr
      - jq
      - curl
```

## Best Practices

1. **Always set timeouts** - Prevent hanging commands
2. **Output JSON** - Easier to parse in subsequent resources
3. **Handle errors** - Check for command availability and failures
4. **Escape user input** - Prevent command injection
5. **Use absolute paths** - Avoid directory confusion
6. **Prefer Python for complex logic** - Shell scripts can get unwieldy

## Security Notes

When using user input in commands, be careful about command injection:

```yaml
# Dangerous - command injection possible
<span v-pre>`command: "echo {{ get('user_input') }}"`</span>


# Safer - use Python for complex input handling
# Or validate input first
preflightCheck:
  validations:
    - get('user_input') matches '^[a-zA-Z0-9_-]+$'
```

## Next Steps

- [Python Resource](python) - For complex data processing
- [HTTP Client](http-client) - For API calls (preferred over curl)
- [Workflow Configuration](../configuration/workflow) - OS package settings
