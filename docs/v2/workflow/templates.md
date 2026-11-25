# Jinja2 templates

kdeps uses [Jinja2](https://jinja.palletsprojects.com/)-compatible templates (with [gonja](https://github.com/nikolalohinski/gonja)). They run on every workflow and resource file before kdeps reads it.

<div v-pre>

```yaml
# workflow.yaml
deploy:
  apiServer:
    portNum: {{ env.PORT | default(8080) }}
```

</div>

## Read next

- [Jinja2 templates reference](/reference/workflow/templates) - every key, option and detail
- [Expressions](/workflow/expressions)
- [Unified API](/workflow/data-access)
