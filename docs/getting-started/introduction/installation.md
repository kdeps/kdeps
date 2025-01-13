---
outline: deep
---

# Installation

Kdeps requires three key components to operate effectively:

- **Kdeps CLI Application**: The primary command-line interface for Kdeps. [Kdeps CLI GitHub
  Repository](https://github.com/kdeps/kdeps).
- **Apple's PKL Programming Language**: A specialized programming language developed by Apple, which is integral to
  Kdeps. [Apple PKL Official WebsiteL](https://pkl-lang.org/index.html).
- **Docker**: A powerful containerization tool used to build and manage AI Agent images and containers. [Docker Official
  Website](https://www.docker.com).

All of these components must be installed to ensure Kdeps functions as intended.

> **Note:** When using the `--latest` flag, setting the `GITHUB_TOKEN` environment variable is optional but recommended
> to fetch the latest schema version from the repository. Without the token, unauthenticated access will be attempted,
> which may be subject to rate limiting.

## Kdeps CLI Installation Guide

### macOS
For macOS users, the simplest way to install Kdeps is via `brew`:

```shell
brew install kdeps/tap/kdeps
```

### Windows, Linux, and macOS
On macOS, Linux, or Windows, you can use `curl` to install Kdeps:

```shell
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh | sh
```

Alternatively, you can use `wget`:

```shell
wget -qO- https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh | sh
```

> **Note for Windows Users:**
> Ensure that the installation command is executed in either [Git Bash](https://git-scm.com/downloads/win) or a [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) terminal for proper functionality.

---

## Troubleshooting

### 'Permission Denied' Error During Installation

#### **Issue:**
You may encounter a `'Permission Denied'` error during installation. This typically occurs when the installer lacks permission to write the Kdeps binary to the `~/.local/bin` directory.

#### **Solution:**
To fix this, grant the necessary write permissions by running the installation command with `sudo`:

```shell
curl -LsSf https://raw.githubusercontent.com/kdeps/kdeps/refs/heads/main/install.sh | sudo sh
```

This ensures the installer has the required access to complete the setup successfully.
