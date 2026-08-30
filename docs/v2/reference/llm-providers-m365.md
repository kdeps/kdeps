# M365 Copilot

Talks to Microsoft 365 Copilot's chat service - a SignalR-over-WebSocket API, not
a REST endpoint. kdeps runs a small local OpenAI-compatible server in front of it
(`m365` backend), so it plugs into the same `chat:` resource and agent-loop paths
as any other provider. See [LLM Provider Reference](/reference/llm-providers) for the other backends.

```yaml
# ~/.kdeps/config.yaml
llm:
  backend: m365
```

There is no `api_key` field - authentication is a signed-in Microsoft 365
account, not an API key.

**Agent-loop mode (recommended): sign in through a real browser window.**
The first time you pick an m365 model (`kdeps --model m365-copilot --backend
m365`, or `/model m365-copilot` in the REPL) with no cached session yet,
kdeps opens a visible Chrome window at the Microsoft login page and waits
for you to complete sign-in yourself - password, MFA app, passkey, SSO tile,
whatever your tenant requires. Nothing is read from or written to a
credentials file for this path; kdeps never sees your password. Once you're
in, the resulting session is cached at
`~/.config/kdeps/m365/token-cache.json` and refreshed silently from then on
- every later launch goes straight to the model, no browser window. Run
`/login` in the REPL any time to force a fresh sign-in (switch accounts,
recover a revoked session).

The Chromium driver itself is installed automatically on first use if it's
missing - no manual `playwright install chromium` step required. Override
which browser launches with `CHROMIUM_PATH`, and the persistent browser
profile location with `M365_BROWSER_PROFILE`.

**Linux only:** downloading Chromium isn't enough - it also needs a handful
of shared libraries the OS doesn't ship by default. On Debian/Ubuntu:

```bash
sudo apt-get install -y libnss3 libnspr4 libasound2t64
```

(On older Debian/Ubuntu releases the package is `libasound2` instead of
`libasound2t64`.) If these are missing, kdeps' launch error names the exact
package manager command for your distro - Playwright detects it directly
from the host, so trust that command over this list if they differ.

**Headless hosts (CI, servers with no display): scripted `secrets.json`
fallback.** Write `~/.config/kdeps/m365/secrets.json` yourself before running
kdeps:

```json
// ~/.config/kdeps/m365/secrets.json
{
  "email": "you@yourtenant.com",
  "password": "your-password",
  "mfaSecret": "your-TOTP-seed"
}
```

`mfaSecret` is the TOTP seed (the same secret you'd scan into an authenticator
app), not a one-time code - kdeps generates codes from it itself, so your
account needs authenticator-app MFA enrolled (not push/SMS-only). When this
file is present, kdeps drives a **headless** Chromium browser through the
Azure AD login form with those credentials instead of opening a visible
window, caching the resulting refresh token the same way. Workflow mode's
`chat:` resource has no terminal or display to prompt on, so it always
requires this file to already exist.

Override paths with `M365_CACHE_FILE`, `M365_SECRETS_FILE`, and
`M365_BROWSER_PROFILE`. Tool calling routes through an auto-provisioned Copilot
Studio agent unless the resolved model tone is a `Claude_*` tone, in which case
kdeps stays agent-less to preserve that tone (attaching an agent forces GPT-5).

Reasoning-tone models (`think-deeper`, `*-think-deeper`) stream their
chain-of-thought summary as live reasoning feedback, same as native
extended-thinking models -- visible in the agent-loop REPL automatically
(thinking is on by default), or via `reasoning_content` in the raw
OpenAI-compatible response for direct API use.

| Model | Description |
|-------|-------------|
| `m365-copilot` | Default - service picks the model |
| `quick` | Fast, lower-latency responses |
| `think-deeper` | Extended reasoning |
| `claude-sonnet` | Claude Sonnet via M365 |
| `claude-opus` | Claude Opus via M365 |
| `gpt-5.5`, `gpt-5.4`, `gpt-5.3`, `gpt-5.2` | GPT-5.x family, `-quick`/`-think-deeper` variants |

## See also

- [LLM Provider Reference](/reference/llm-providers) -- all other backends
- [Agent Loop Mode](/modes/agent-loop-mode) -- `/login` command and REPL usage
