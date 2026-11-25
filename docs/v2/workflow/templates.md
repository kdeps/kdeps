# Jinja2 templates

kdeps uses [Jinja2](https://jinja.palletsprojects.com/)-compatible templates (with [gonja](https://github.com/nikolalohinski/gonja)). This is the one template system for project scaffolding and for runtime YAML preprocessing. Preprocessing runs for every workflow file and every resource file.

<div v-pre>

```yaml
# workflow.yaml
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-api
  version: "1.0.0"
  targetActionId: response
deploy:
  apiServer:
{% if env.PORT %}
    portNum: {{ env.PORT | int }}
{% else %}
    portNum: 8080
{% endif %}
```

</div>

## Read next

- [Jinja2 templates reference](/reference/workflow/templates) - YAML preprocessing, template syntax, scaffolding templates
- [Expressions](/workflow/expressions)
- [Unified API](/workflow/data-access)
- [Jinja2 documentation](https://jinja.palletsprojects.com/en/stable/)
