## Auto-update Dependencies

This PR automatically updates dependencies to their latest versions.

### PKL Update

**Version:** `0.29.1` → `0.30.2`

**Changes:**
- Updated Dockerfile
- Updated .github/workflows/build-test.yml
- Updated .github/workflows/release.yml
- Updated pkg/version/version.go
- Updated pkg/docker/image_test.go

**Release Notes:** https://github.com/apple/pkl/releases/tag/v0.30.2

### Ollama Update

**Version:** `0.9.6` → `0.13.5`

**Changes:**
- Updated pkg/version/version.go (DefaultOllamaImageTag)
- Updated README.md examples
- Updated docs/index.md examples
- Updated docs/KDeps_Whitepaper.md examples
- Updated docs/getting-started/configuration/workflow.md examples

**Docker Image:** `ollama/ollama:0.13.5`
**Verified:** ✓ Image exists and is pullable

**Release Notes:** https://github.com/ollama/ollama/releases/tag/v0.13.5


---
*This PR was automatically created by the auto-update-dependencies workflow.*
