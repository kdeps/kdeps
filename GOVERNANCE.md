# Governance

This document describes how the kdeps project is run.

## Roles

### Users

Anyone who uses kdeps. Users contribute by filing bug reports, requesting
features, joining discussions, and improving documentation.

### Contributors

Anyone who submits a pull request, issue, or documentation change that is
merged or acted on. Contributors are credited in the release notes.

### Maintainers

Contributors with write access to the repository. Maintainers review and merge
pull requests, triage issues, cut releases, and enforce the
[code of conduct](CODE_OF_CONDUCT.md). The current maintainers are listed on
the [GitHub organization page](https://github.com/kdeps).

## Decision making

- Day-to-day changes (bug fixes, docs, dependency bumps) are merged by any
  maintainer after review.
- Changes to the `workflow.yaml` schema, the CLI surface, or the execution
  model require agreement from at least two maintainers and a tracking issue on
  the [project board](https://github.com/orgs/kdeps/projects/1).
- Breaking changes ship under a new `apiVersion`. Existing workflows keep
  working on their declared version.

## Becoming a maintainer

A contributor with a sustained history of high-quality contributions and
reviews may be invited to become a maintainer by the existing maintainers.

## Changing this document

Open a pull request. Changes require agreement from a majority of maintainers.

## See also

- [Contributing guide](CONTRIBUTING.md)
- [Code of conduct](CODE_OF_CONDUCT.md)
- [Support](SUPPORT.md)
