# Secrets

The vault keeps API keys and passwords encrypted in `secrets.age`. A workflow reads a secret with `secret('NAME')`. The file is safe to commit and ships inside the appliance; only the vault password comes from outside.

```text
kdeps secret set NAME -> secrets.age (encrypted) -> secret('NAME') when the resource runs
```

Store the value once, then use it in a resource:

```yaml
# kdeps secret set github_token   (asks for the value and the vault password)
headers:
  Authorization: "Bearer {{ secret('github_token') }}"   # decrypted at run time, never written to disk
```

## Read next

- [Secrets reference](/reference/workflow/secrets) - vault files, commands, password sources, `config.yaml`
- [Data access](/workflow/data-access) - `get()`, `env()` and the other accessors
- [Docker](/deploy/docker) - give the appliance its vault password
