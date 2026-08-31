# Security policy

## Supported versions

kdeps follows a rolling-release model. Only the latest released version and the
`main` branch receive security fixes.

| Version | Supported |
| ------- | --------- |
| Latest release | Yes |
| `main` | Yes |
| Older releases | No |

## Reporting a vulnerability

Do not open a public issue for security vulnerabilities.

Report privately through one of these channels:

- GitHub private vulnerability reporting:
  <https://github.com/kdeps/kdeps/security/advisories/new>
- Email: **security@kdeps.com**

Include a description of the issue, the affected version or commit, reproduction
steps, and the impact you expect.

## What to expect

- Acknowledgement within 3 business days.
- An initial assessment and severity rating within 10 business days.
- Progress updates at least every 10 business days until resolution.
- Credit in the release notes and advisory once a fix ships, unless you ask to
  remain anonymous.

If a report is declined, the maintainers explain why. If it is accepted, the
maintainers coordinate a fix and a disclosure timeline with you.

## Scope

In scope: the `kdeps` CLI, the workflow engine, resource executors, the agent
loop, and the packaging and deployment tooling in this repository.

Out of scope: third-party model backends, external APIs called by `httpClient:`
or `browser:` resources, and user-authored workflows.
