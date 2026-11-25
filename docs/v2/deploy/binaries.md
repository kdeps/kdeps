# Prepackage - standalone executables

`kdeps bundle prepackage` puts a `.kdeps` workflow archive into the kdeps binary. The result is one self-contained executable per architecture: copy it to the target machine and run it.

```text
kdeps new -> kdeps run -> kdeps bundle package -> kdeps bundle prepackage -> one binary per target
```

```bash
kdeps bundle package my-agent/ --output dist/
kdeps bundle prepackage dist/my-agent-1.0.0.kdeps --output dist/
./dist/my-agent-1.0.0-linux-amd64
```

## Read next

- [Prepackage - standalone executables reference](/reference/deploy/binaries) - every key, option and detail
- [Package command](/workflow/quickstart) - Creating `.kdeps` archives
- [Docker deployment](/deploy/docker) - Containerized deployment
