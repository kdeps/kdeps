# Wrap a shell command in an API

Build an API endpoint that runs a shell command with the `exec` component
and returns its output as JSON. Use this pattern to give a script, a
CLI tool, or a system check as an HTTP service.

The resource that runs the command:

```yaml
# resources/system-info.yaml
actionId: systemInfo
name: System information
validations:
  methods: [GET]              # only GET reaches this resource
  routes: [/api/v1/exec]
component:                    # the exec component calls the kdeps API's exec tool
  name: exec
  with:
    command: "echo 'System Info:' && uname -a && echo 'Date:' && date"
    timeout: 10s              # kill the command and fail after 10s
```

## Read next

- [Full walkthrough](/reference/examples/shell-command-api) - every step, file and command
- [exec component](/workflow/resources/exec) - stdin, environment, exit-code access
- [Error handling (onError)](/workflow/error-handling) - retry and fallback on a failed command
