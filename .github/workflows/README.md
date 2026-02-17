# GitHub Actions Workflows

This directory contains the CI/CD workflows for the KDeps project.

## Workflows

### build-test.yml
**Trigger**: Push to main, Pull Requests

Runs on every push to main and on pull requests. This workflow:
- Runs linting with golangci-lint
- Runs unit tests with coverage reporting
- Runs integration tests
- Runs E2E tests
- Requires Ollama and Python (via uv) for full test suite

### release.yml
**Trigger**: Version tags (v*)

Creates official releases when version tags are pushed. This workflow:
- Builds binaries for all supported platforms (macOS, Linux, Windows)
- Uses GoReleaser for cross-platform compilation with Zig
- Publishes releases to GitHub
- Updates Homebrew tap
- Triggers deployment to kdeps-io website

**Usage**:
```bash
git tag v2.1.0
git push origin v2.1.0
```

### nightly-release.yml
**Trigger**: Daily at 2 AM UTC, Manual via workflow_dispatch

Automatically updates Go modules and creates nightly releases. This workflow:
- Checks for code changes since the last release tag
- Updates all Go modules to their latest versions using `go get -u ./...`
- Runs `go mod tidy` to clean up dependencies
- Validates changes with linting, building, and unit tests
- Commits updated `go.mod` and `go.sum` to main branch (if module updates exist)
- Creates a nightly tag with format `vX.X.X-nightlyYYYYMMDDHHMM`
- Builds and publishes binaries with updated dependencies
- **Release type**: Marked as "latest" when all validation checks pass, or as "prerelease" if any checks fail
- **Release trigger**: Creates a release if either code changes exist since last release OR module updates are available
- Skips release only if no code changes AND no module updates exist

**Manual Trigger**:
1. Go to Actions tab in GitHub
2. Select "Nightly Release" workflow
3. Click "Run workflow"

### docs.yml
**Trigger**: Push to main (docs changes), Manual

Builds and deploys the documentation site.

## Secrets Required

The workflows require the following secrets to be configured in the repository:

- `RELEASE_TOKEN`: Personal Access Token with repo and packages write permissions
  - Used by: release.yml, nightly-release.yml
  - Purpose: Push commits, create releases, trigger other workflows

## Best Practices

1. **Testing**: Always run `make test` locally before pushing
2. **Linting**: Run `make lint` to catch issues before CI
3. **Version Tags**: Follow semantic versioning (v2.x.x)
4. **Nightly Builds**: Monitor nightly workflow results to catch dependency issues early
5. **Dependencies**: Review nightly update commits before merging critical changes

## Troubleshooting

### Nightly Release Issues

If the nightly release workflow fails:

1. **Linting Failures**: Module updates may introduce linting issues
   - Check the workflow logs for specific errors
   - Fix issues in a separate PR
   - The next nightly run will apply updates again

2. **Test Failures**: Updated dependencies may break tests
   - Review the test failure logs
   - Update tests or pin problematic dependencies
   - Consider adding go.mod constraints if needed

3. **No Updates**: Workflow skips if no code changes AND no module updates exist
   - This is expected behavior
   - Check logs to confirm "No changes since last release" message
   - A release is triggered if either code changes or module updates exist

4. **Authentication Issues**: RELEASE_TOKEN problems
   - Verify token has correct permissions
   - Token must have repo and packages:write scopes
   - Token must not be expired

### General CI Issues

- **macOS Runner Failures**: Release workflows use macOS runners which can have limited availability
- **Timeout Issues**: E2E tests have 15-minute timeout; extend if needed
- **Dependency Downloads**: First run after cache clear will be slower

## Monitoring

- View workflow runs: [Actions Tab](https://github.com/kdeps/kdeps/actions)
- Check workflow status badges in README.md
- Review nightly commits with `[nightly]` tag in commit message
