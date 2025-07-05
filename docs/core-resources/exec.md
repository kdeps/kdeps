---
outline: deep
---

# Exec Resource

The `exec` resource enables execution of shell scripts and commands within your AI agent workflow. This powerful resource provides access to the underlying system, allowing you to run Ubuntu packages, system utilities, custom scripts, and integrate with external tools and services.

## Overview

The exec resource enables:

- **Shell Command Execution**: Run bash, shell scripts, and system commands
- **Environment Variable Access**: Use custom and system environment variables
- **Package Integration**: Access Ubuntu packages defined in workflow configuration
- **Output Capture**: Retrieve stdout, stderr, and exit codes from executed commands
- **Timeout Management**: Control execution time with configurable timeouts
- **Error Handling**: Implement retry logic and validation for command execution

## Creating a New Exec Resource

To create a new `exec` resource, you can either generate a new AI agent using the `kdeps new` command or scaffold the resource directly.

**Scaffolding an Exec Resource:**
```bash
kdeps scaffold [aiagent] exec
```

This command adds an `exec` resource to the `aiagent/resources` folder, generating the following structure:

```bash
aiagent
└── resources
    └── exec.pkl
```

The generated file includes essential metadata and common configurations, such as [Skip Conditions](../workflow-control/skip.md) and [API Validations](../workflow-control/api-request-validations.md). For more details, refer to the [Resource Configuration](./resource.md) documentation.

## Basic Exec Resource Structure

A complete exec resource file looks like this:

```apl
amends "resource.pkl"

ActionID = "execResource"
Name = "Shell Command Executor"
Description = "Executes system commands and scripts"
Category = "system"
Requires { "dataResource" }

Run {
    RestrictToHTTPMethods { "POST" }
    RestrictToRoutes { "/api/v1/execute" }
    AllowedParams { "command"; "args" }
    
    PreflightCheck {
        Validations { 
            "@(request.data('command'))" != ""
        }
        Retry = false
        RetryTimes = 1
    }
    
    PostflightCheck {
        Validations { 
            "@(exec.exitCode('execResource'))" == 0
        }
        Retry = true
        RetryTimes = 2
    }
    
    Exec {
        Command = """
#!/bin/bash
echo "Hello World"
"""
        Env {
            ["CUSTOM_VAR"] = "example_value"
        }
        TimeoutDuration = 60.s
    }
}
```

## Exec Configuration Block

The `Exec` block defines the shell execution parameters:

```apl
Exec {
    Command = """
#!/bin/bash
set -e  # Exit on error
set -u  # Exit on undefined variable

echo "Starting script execution..."
echo "Current directory: $(pwd)"
echo "User: $(whoami)"
echo "Date: $(date)"

# Your commands here
ls -la /tmp
"""
    
    Env {
        ["SCRIPT_MODE"] = "production"
        ["LOG_LEVEL"] = "INFO"
        ["WORKING_DIR"] = "/tmp"
    }
    
    TimeoutDuration = 120.s
}
```

### Core Properties

- **`Command`**: The shell command(s) to execute, enclosed in triple quotes (`"""`) for multi-line support
- **`Env`**: Environment variables available during execution
- **`TimeoutDuration`**: Execution timeout (e.g., `60.s`, `5.min`), after which the command is terminated

### Command Structure

**Single Commands:**
```apl
Command = "echo 'Hello World'"
```

**Multi-line Scripts:**
```apl
Command = """
#!/bin/bash
set -e

# Script header
echo "=== Starting Process ==="

# Main logic
for i in {1..5}; do
    echo "Processing item $i"
    sleep 1
done

echo "=== Process Complete ==="
"""
```

**Dynamic Commands:**
```apl
Command = """
#!/bin/bash
COMMAND="@(request.data('command'))"
ARGS="@(request.data('args'))"

if [[ -n "$COMMAND" ]]; then
    echo "Executing: $COMMAND $ARGS"
    $COMMAND $ARGS
else
    echo "No command specified"
    exit 1
fi
"""
```

## Environment Variables

### Setting Custom Environment Variables

```apl
Env {
    ["API_URL"] = "https://api.example.com"
    ["API_KEY"] = "@(request.data('api_key'))"
    ["PROCESSING_MODE"] = "@(request.data('mode') ?? 'default')"
    ["DEBUG"] = "true"
    ["MAX_RETRIES"] = "3"
}
```

### Accessing System Variables

```apl
Command = """
#!/bin/bash

# Access custom environment variables
echo "API URL: $API_URL"
echo "Processing Mode: $PROCESSING_MODE"

# Access system variables
echo "Home Directory: $HOME"
echo "Path: $PATH"
echo "Hostname: $HOSTNAME"

# Access Kdeps variables
echo "Container ID: $KDEPS_CONTAINER_ID"
echo "Agent ID: $KDEPS_AGENT_ID"
"""
```

### Environment Variable Patterns

**Configuration Management:**
```apl
Env {
    ["CONFIG_FILE"] = "/app/config/production.json"
    ["LOG_FILE"] = "/var/log/agent.log"
    ["DATA_DIR"] = "/data/processing"
    ["TEMP_DIR"] = "/tmp/agent-work"
}
```

**Dynamic Values:**
```apl
Env {
    ["USER_ID"] = "@(request.headers('X-User-ID'))"
    ["REQUEST_ID"] = "@(uuid())"
    ["TIMESTAMP"] = "@(now())"
    ["SESSION_TOKEN"] = "@(memory.get('session_token'))"
}
```

## Common Use Cases

### File Operations

```apl
Exec {
    Command = """
#!/bin/bash
set -e

SOURCE_FILE="@(request.data('source_file'))"
DEST_FILE="@(request.data('dest_file'))"

# Validate inputs
if [[ ! -f "$SOURCE_FILE" ]]; then
    echo "Source file not found: $SOURCE_FILE"
    exit 1
fi

# Create backup
cp "$SOURCE_FILE" "${SOURCE_FILE}.backup"

# Process file
sed 's/old_text/new_text/g' "$SOURCE_FILE" > "$DEST_FILE"

echo "File processed successfully"
echo "Source: $SOURCE_FILE"
echo "Destination: $DEST_FILE"
"""
    
    TimeoutDuration = 30.s
}
```

### Package Management

```apl
Exec {
    Command = """
#!/bin/bash
set -e

PACKAGE_NAME="@(request.data('package'))"

# Update package list
apt-get update -qq

# Install package
apt-get install -y "$PACKAGE_NAME"

# Verify installation
if command -v "$PACKAGE_NAME" &> /dev/null; then
    echo "Package $PACKAGE_NAME installed successfully"
    $PACKAGE_NAME --version
else
    echo "Package installation failed"
    exit 1
fi
"""
    
    TimeoutDuration = 300.s  # 5 minutes for package installation
}
```

### Data Processing

```apl
Exec {
    Command = """
#!/bin/bash
set -e

INPUT_FILE="@(request.data('input_file'))"
OUTPUT_FILE="@(request.data('output_file'))"
PROCESSING_MODE="@(request.data('mode') ?? 'standard')"

echo "Processing file: $INPUT_FILE"
echo "Mode: $PROCESSING_MODE"

case "$PROCESSING_MODE" in
    "compress")
        gzip -c "$INPUT_FILE" > "$OUTPUT_FILE.gz"
        echo "File compressed to $OUTPUT_FILE.gz"
        ;;
    "convert")
        # Convert CSV to JSON
        python3 -c "
import csv, json, sys
with open('$INPUT_FILE', 'r') as f:
    reader = csv.DictReader(f)
    data = list(reader)
with open('$OUTPUT_FILE', 'w') as f:
    json.dump(data, f, indent=2)
"
        echo "File converted to JSON: $OUTPUT_FILE"
        ;;
    *)
        cp "$INPUT_FILE" "$OUTPUT_FILE"
        echo "File copied: $OUTPUT_FILE"
        ;;
esac
"""
    
    Env {
        ["PYTHONPATH"] = "/usr/local/lib/python3.8/site-packages"
    }
    
    TimeoutDuration = 180.s
}
```

### External API Integration

```apl
Exec {
    Command = """
#!/bin/bash
set -e

API_ENDPOINT="@(request.data('endpoint'))"
API_METHOD="@(request.data('method') ?? 'GET')"
API_DATA="@(request.data('data'))"

# Prepare curl command
CURL_CMD="curl -s -X $API_METHOD"

# Add headers
CURL_CMD="$CURL_CMD -H 'Content-Type: application/json'"
CURL_CMD="$CURL_CMD -H 'Authorization: Bearer $API_TOKEN'"

# Add data for POST/PUT requests
if [[ "$API_METHOD" == "POST" || "$API_METHOD" == "PUT" ]] && [[ -n "$API_DATA" ]]; then
    CURL_CMD="$CURL_CMD -d '$API_DATA'"
fi

# Execute request
echo "Making API request to: $API_ENDPOINT"
RESPONSE=$(eval "$CURL_CMD $API_ENDPOINT")
EXIT_CODE=$?

if [[ $EXIT_CODE -eq 0 ]]; then
    echo "API request successful"
    echo "$RESPONSE" > /tmp/api_response.json
else
    echo "API request failed with exit code: $EXIT_CODE"
    exit $EXIT_CODE
fi
"""
    
    Env {
        ["API_TOKEN"] = "@(memory.get('api_token'))"
    }
    
    TimeoutDuration = 60.s
}
```

## Advanced Configuration

### Conditional Execution

```apl
Exec {
    Command = """
#!/bin/bash
set -e

MODE="@(request.data('mode'))"
USER_ROLE="@(request.headers('X-User-Role'))"

# Check user permissions
if [[ "$USER_ROLE" != "admin" ]] && [[ "$MODE" == "admin" ]]; then
    echo "Insufficient permissions for admin mode"
    exit 1
fi

# Execute based on mode
case "$MODE" in
    "backup")
        echo "Creating backup..."
        tar -czf "/backup/data-$(date +%Y%m%d).tar.gz" /data
        ;;
    "restore")
        echo "Restoring from backup..."
        BACKUP_FILE="@(request.data('backup_file'))"
        tar -xzf "$BACKUP_FILE" -C /
        ;;
    "cleanup")
        echo "Cleaning up temporary files..."
        find /tmp -type f -mtime +7 -delete
        ;;
    *)
        echo "Unknown mode: $MODE"
        exit 1
        ;;
esac
"""
    
    TimeoutDuration = 600.s  # 10 minutes
}
```

### Error Handling and Logging

```apl
Exec {
    Command = """
#!/bin/bash

# Error handling function
handle_error() {
    local exit_code=$1
    local line_number=$2
    echo "Error on line $line_number: Exit code $exit_code" >&2
    echo "Command: $BASH_COMMAND" >&2
    exit $exit_code
}

# Set up error handling
set -e
trap 'handle_error $? $LINENO' ERR

# Logging function
log() {
    local level=$1
    shift
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $*" | tee -a "$LOG_FILE"
}

# Main script
log "INFO" "Starting script execution"

TASK="@(request.data('task'))"
log "INFO" "Task: $TASK"

# Execute task with error handling
case "$TASK" in
    "process")
        log "INFO" "Processing data..."
        # Add processing logic here
        log "INFO" "Processing completed successfully"
        ;;
    *)
        log "ERROR" "Unknown task: $TASK"
        exit 1
        ;;
esac

log "INFO" "Script execution completed"
"""
    
    Env {
        ["LOG_FILE"] = "/var/log/exec-resource.log"
        ["DEBUG"] = "@(request.data('debug') ?? 'false')"
    }
    
    TimeoutDuration = 300.s
}
```

## Accessing Exec Outputs

Use exec functions to retrieve command results:

```apl
// Get stdout output
@(exec.stdout('execResource'))

// Get stderr output
@(exec.stderr('execResource'))

// Get exit code
@(exec.exitCode('execResource'))

// Check if execution was successful
@(exec.success('execResource'))

// Get execution duration
@(exec.duration('execResource'))
```

### Using Outputs in Other Resources

```apl
ActionID = "responseResource"
Requires { "execResource" }

Run {
    local execOutput = "@(exec.stdout('execResource'))"
    local execSuccess = "@(exec.success('execResource'))"
    local exitCode = "@(exec.exitCode('execResource'))"
    
    APIResponse {
        Response {
            Data {
                if (execSuccess) {
                    new Mapping {
                        ["success"] = true
                        ["output"] = execOutput
                        ["exit_code"] = exitCode
                    }
                } else {
                    new Mapping {
                        ["success"] = false
                        ["error"] = "@(exec.stderr('execResource'))"
                        ["exit_code"] = exitCode
                    }
                }
            }
        }
    }
}
```

## Security Considerations

### Input Validation

```apl
Exec {
    Command = """
#!/bin/bash
set -e

# Validate inputs
FILENAME="@(request.data('filename'))"
ACTION="@(request.data('action'))"

# Sanitize filename - only allow alphanumeric, dots, dashes, underscores
if [[ ! "$FILENAME" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    echo "Invalid filename format"
    exit 1
fi

# Whitelist allowed actions
case "$ACTION" in
    "read"|"write"|"delete")
        echo "Action allowed: $ACTION"
        ;;
    *)
        echo "Action not allowed: $ACTION"
        exit 1
        ;;
esac

# Prevent path traversal
if [[ "$FILENAME" == *".."* ]]; then
    echo "Path traversal not allowed"
    exit 1
fi
"""
}
```

### Safe Command Execution

```apl
Exec {
    Command = """
#!/bin/bash
set -e -u -o pipefail

# Set safe defaults
umask 077
export PATH="/usr/local/bin:/usr/bin:/bin"

# Validate user input
USER_COMMAND="@(request.data('command'))"

# Whitelist of allowed commands
ALLOWED_COMMANDS="ls cat grep head tail wc sort uniq"

# Extract command name
COMMAND_NAME=$(echo "$USER_COMMAND" | awk '{print $1}')

# Check if command is allowed
if [[ " $ALLOWED_COMMANDS " =~ " $COMMAND_NAME " ]]; then
    echo "Executing allowed command: $USER_COMMAND"
    eval "$USER_COMMAND"
else
    echo "Command not allowed: $COMMAND_NAME"
    exit 1
fi
"""
    
    TimeoutDuration = 30.s
}
```

## Best Practices

### Script Structure

1. **Use shebang**: Always start with `#!/bin/bash`
2. **Set error handling**: Use `set -e` to exit on errors
3. **Add logging**: Include informative echo statements
4. **Validate inputs**: Check all input parameters
5. **Use meaningful variable names**: Make scripts readable

### Performance Optimization

1. **Set appropriate timeouts**: Balance functionality with resource usage
2. **Use efficient commands**: Choose optimal tools for tasks
3. **Minimize external dependencies**: Reduce complexity where possible
4. **Cache results**: Store outputs for reuse when appropriate

### Error Handling

1. **Use exit codes**: Return meaningful exit codes (0 for success, 1+ for errors)
2. **Capture errors**: Use `2>&1` to capture stderr
3. **Implement retries**: Use PostflightCheck for retry logic
4. **Log errors**: Provide detailed error information

### Security

1. **Validate all inputs**: Never trust user-provided data
2. **Use least privilege**: Run with minimal required permissions
3. **Sanitize paths**: Prevent directory traversal attacks
4. **Whitelist commands**: Only allow safe operations

## Complete Example: File Processing Service

```apl
amends "resource.pkl"

ActionID = "fileProcessorExec"
Name = "File Processing Service"
Description = "Processes uploaded files with various operations"
Category = "file-processing"

Run {
    RestrictToHTTPMethods { "POST" }
    RestrictToRoutes { "/api/v1/process-file" }
    AllowedParams { "operation"; "format"; "options" }
    
    PreflightCheck {
        Validations { 
            "@(request.files().length())" > 0
            "@(request.data('operation'))" != ""
        }
        Retry = false
        RetryTimes = 1
    }
    
    PostflightCheck {
        Validations { 
            "@(exec.exitCode('fileProcessorExec'))" == 0
            "@(exec.stdout('fileProcessorExec'))" != ""
        }
        Retry = true
        RetryTimes = 2
    }
    
    Exec {
        Command = """
#!/bin/bash
set -e -u -o pipefail

# Configuration
UPLOAD_DIR="/tmp/uploads"
OUTPUT_DIR="/tmp/processed"
LOG_FILE="/var/log/file-processor.log"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG_FILE"
}

# Create directories
mkdir -p "$UPLOAD_DIR" "$OUTPUT_DIR"

# Get parameters
OPERATION="$OPERATION_TYPE"
FORMAT="$OUTPUT_FORMAT"
INPUT_FILE="$UPLOAD_DIR/input.${FILE_EXTENSION}"

log "Starting file processing operation: $OPERATION"

# Process uploaded file
case "$OPERATION" in
    "convert")
        log "Converting file to format: $FORMAT"
        case "$FORMAT" in
            "pdf")
                libreoffice --headless --convert-to pdf --outdir "$OUTPUT_DIR" "$INPUT_FILE"
                ;;
            "jpg")
                convert "$INPUT_FILE" "$OUTPUT_DIR/output.jpg"
                ;;
            *)
                log "Unsupported format: $FORMAT"
                exit 1
                ;;
        esac
        ;;
    "compress")
        log "Compressing file"
        gzip -c "$INPUT_FILE" > "$OUTPUT_DIR/compressed.gz"
        ;;
    "analyze")
        log "Analyzing file"
        file "$INPUT_FILE" > "$OUTPUT_DIR/analysis.txt"
        wc -l "$INPUT_FILE" >> "$OUTPUT_DIR/analysis.txt"
        ;;
    *)
        log "Unknown operation: $OPERATION"
        exit 1
        ;;
esac

log "File processing completed successfully"
echo "Operation: $OPERATION completed"
"""
        
        Env {
            ["OPERATION_TYPE"] = "@(request.data('operation'))"
            ["OUTPUT_FORMAT"] = "@(request.data('format') ?? 'txt')"
            ["FILE_EXTENSION"] = "@(request.files()[0].extension())"
        }
        
        TimeoutDuration = 300.s
    }
}
```

## Next Steps

- **[Python Resource](./python.md)**: Combine shell commands with Python scripts
- **[HTTP Client Resource](./client.md)**: Make API calls from shell scripts
- **[Response Resource](./response.md)**: Format exec outputs for API responses
- **[Functions & Utilities](../functions-utilities/functions.md)**: Available exec functions
- **[Skip Conditions](../workflow-control/skip.md)**: Conditional exec execution

The exec resource provides powerful system-level capabilities for your AI agents. Use it responsibly with proper input validation, error handling, and security measures to build robust and secure applications.
