# Secrets reference

The vault is a YAML map of `NAME: value`, encrypted with [age](https://age-encryption.org) and a password (scrypt). kdeps decrypts it in memory when a value is needed and never writes the decrypted values to disk.

```d2
direction: right
set: "kdeps secret set NAME" {shape: oval}
project: "secrets.age\nnext to workflow.yaml"
global: "~/.kdeps/secrets.age"
wf: "secret('NAME')\nin a workflow"
cfg: "secret:NAME reference\nin ~/.kdeps/config.yaml"
set -> project: "folder holds workflow.yaml"
set -> global: "other folders, or --global"
project -> wf: "1st"
global -> wf: "2nd"
global -> cfg
```

## Vault files

| File | Used by | Commit it? |
|---|---|---|
| `secrets.age` next to `workflow.yaml` or `agency.yaml` | `secret('NAME')` in that workflow | Yes. It is encrypted, and `kdeps bundle` puts it in the appliance. |
| `~/.kdeps/secrets.age` | `secret('NAME')` when the project vault does not have `NAME`, and `${secret:NAME}` in `~/.kdeps/config.yaml` | No. It holds the secrets of this machine. |

Both files are written with mode `0600`. Use the same password for both: kdeps asks for it one time per run.

## Commands

```bash
kdeps secret set github_token          # asks for the value (hidden) and the password
printf '%s' "$TOKEN" | kdeps secret set github_token   # value from stdin, for scripts
kdeps secret list                      # names only, never values
kdeps secret rm github_token
kdeps secret edit                      # decrypt into $EDITOR, encrypt again on exit
kdeps secret set --global db.pass      # always ~/.kdeps/secrets.age
```

A command works on `./secrets.age` when the current folder holds `workflow.yaml` or `agency.yaml`, and on `~/.kdeps/secrets.age` otherwise. A name uses letters, digits, `_`, `.` and `-`. The first `set` or `edit` on a new vault asks for the password two times.

`kdeps secret edit` writes the decrypted map to a temporary file (mode `0600`), opens it in `$KDEPS_EDITOR`, `$VISUAL`, `$EDITOR`, then `vi`, and removes the file after the edit.

## Password

kdeps reads the vault password from the first source that is set:

| Source | Use it for |
|---|---|
| `KDEPS_VAULT_PASSWORD` | CI, Docker, Kubernetes secrets |
| `KDEPS_VAULT_PASSWORD_FILE` | a mounted file; kdeps reads its first line |
| a prompt | a terminal session |

With no source, a `secret()` call fails with `the vault is locked`. A wrong password fails with `wrong password or not a kdeps vault`.

## In a workflow

`secret('NAME')` returns the value. It works in every expression, like `env('NAME')`.

<div v-pre>

```yaml
# resources/issues.yaml
actionId: issues
name: My issues
component:
  name: http_request
  with:
    url: https://api.github.com/user/issues
    headers:
      Authorization: "Bearer {{ secret('github_token') }}"   # project vault, then ~/.kdeps/secrets.age
```

</div>

An unknown name fails the resource with `secret not found: NAME` and the vault files it read. kdeps never prints a value in an error.

A secret in a chat prompt goes to the model. Pass secrets to tools and HTTP headers, not to prompts.

## In `~/.kdeps/config.yaml`

A value in `config.yaml` can hold `${NAME}` (an environment variable, empty when unset) and `${secret:NAME}` (a secret in `~/.kdeps/secrets.age`). Other `${...}` text stays as written, and `$NAME` without braces is not expanded.

```yaml
# ~/.kdeps/config.yaml
http_connections:
  github:
    auth:
      type: bearer
      token: "${secret:github_token}"     # from ~/.kdeps/secrets.age
  internal:
    auth:
      type: basic
      username: "${API_USER}"             # from the environment
      password: "${secret:internal_pass}"
```

kdeps expands the values in memory when it reads the file. The file on disk keeps the references. When the vault is locked or the name is missing, the value is empty and kdeps prints a warning with the line number; commands that do not use the value still run.

## In an appliance

`kdeps bundle` includes `secrets.age`. Give the password at run time:

```bash
docker run -e KDEPS_VAULT_PASSWORD="$VAULT_PASSWORD" myregistry/myagent:latest
```

In Kubernetes, put the password in a `Secret` and map it to `KDEPS_VAULT_PASSWORD`, or mount it as a file and set `KDEPS_VAULT_PASSWORD_FILE`.

## Read the vault without kdeps

The file is a standard age file. The `age` command line tool decrypts it with the same password:

```bash
age -d secrets.age       # prints the YAML map
```
