# ReleasePlease Dagger Module

A Dagger module that provides an easy way to integrate [release-please](https://github.com/googleapis/release-please) into your CI/CD pipelines. This module automates the creation of release pull requests and GitHub releases using the release-please tool.

## Overview

The ReleasePlease module simplifies release management by:
- Creating automated release pull requests with changelog entries
- Generating GitHub releases with proper versioning
- Supporting multiple release types (Node, Python, Go, Rust, and more)
- Managing the entire release workflow through Dagger

## Prerequisites

- **Dagger**: A Dagger engine (v0.20.3 or compatible)
- **GitHub Token**: A GitHub personal access token with appropriate permissions
  - For public repositories: `contents: read/write`, `pull-requests: read/write`
  - For private repositories: add `repo: read/write`
  - To manage releases: `contents: read/write`

## Installation

### Using the Dagger CLI

You can call this module directly from the Dagger CLI without explicit installation. Dagger will fetch and cache the module automatically.

```bash
dagger call --help
```

To use in your local setup:

```bash
dagger init --name mymodule --import github.com/your-org/dagger/release-please
```

## Usage

### Basic Setup

All functions require a GitHub token as input. You can provide this via environment variables or direct input:

```bash
# Export your GitHub token
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Or pass it directly to dagger
dagger call <function> --token=$GITHUB_TOKEN
```

### Module Initialization

Initialize the ReleasePlease module with your GitHub token:

```bash
dagger call new --token env:GITHUB_TOKEN
```

## Available Functions

### 1. ReleasePr

Creates a release pull request with automated version bumping and changelog generation.

**Command:**
```bash
dagger call new --token env:GITHUB_TOKEN release-pr \
  --release-type=<type> \
  --repo-url=<url>
```

**Parameters:**
- `release-type` (required): The type of release (e.g., `node`, `python`, `go`, `rust`, `java`, `ruby`, `php`, `deno`)
- `repo-url` (required): GitHub repository URL (e.g., `github.com/owner/repo`)

**Example:**
```bash
dagger call new --token env:GITHUB_TOKEN release-pr \
  --release-type=go \
  --repo-url=github.com/dagger/dagger
```

**Output:**
Returns the stdout output from the `release-please release-pr` command, typically containing the PR URL or status message.

### 2. GithubRelease

Creates a GitHub release based on the current repository state. This typically runs after the release PR is merged.

**Command:**
```bash
dagger call new --token env:GITHUB_TOKEN github-release \
  --release-type=<type> \
  --repo-url=<url>
```

**Parameters:**
- `release-type` (required): The type of release (e.g., `node`, `python`, `go`, `rust`, `java`, `ruby`, `php`, `deno`)
- `repo-url` (required): GitHub repository URL (e.g., `github.com/owner/repo`)

**Example:**
```bash
dagger call new --token env:GITHUB_TOKEN github-release \
  --release-type=go \
  --repo-url=github.com/dagger/dagger
```

**Output:**
Returns the stdout output from the `release-please github-release` command, containing release details.

### 3. Bootstrap

Initializes release-please in a repository by creating the necessary configuration files and setting up the release workflow.

**Command:**
```bash
dagger call new --token env:GITHUB_TOKEN bootstrap \
  --repo-url=<url>
```

**Parameters:**
- `repo-url` (required): GitHub repository URL (e.g., `github.com/owner/repo`)

**Example:**
```bash
dagger call new --token env:GITHUB_TOKEN bootstrap \
  --repo-url=github.com/dagger/dagger
```

**Output:**
Returns the stdout output from the `release-please bootstrap` command, showing initialization status.

### 4. Run

Executes the complete release workflow: creates a GitHub release and then creates a release pull request. This is a convenience function that combines both operations.

**Command:**
```bash
dagger call new --token env:GITHUB_TOKEN run \
  --release-type=<type> \
  --repo-url=<url>
```

**Parameters:**
- `release-type` (required): The type of release (e.g., `node`, `python`, `go`, `rust`, `java`, `ruby`, `php`, `deno`)
- `repo-url` (required): GitHub repository URL (e.g., `github.com/owner/repo`)

**Example:**
```bash
dagger call new --token env:GITHUB_TOKEN run \
  --release-type=go \
  --repo-url=github.com/dagger/dagger
```

**Output:**
Returns combined output from both the `github-release` and `release-pr` commands.

### 5. Container

Returns a Dagger container with release-please pre-installed. This is useful for advanced use cases where you need to run custom release-please commands.

**Command:**
```bash
dagger call new --token env:GITHUB_TOKEN container
```

**Example Usage in Code:**
```bash
dagger call new --token env:GITHUB_TOKEN container \
  with-secret-variable --name GITHUB_TOKEN --secret env:GITHUB_TOKEN \
  with-exec --args sh,-c,"release-please --version"
```

## Common Examples

### Complete Release Workflow for a Go Project

```bash
# Step 1: Bootstrap the repository
dagger call new --token env:GITHUB_TOKEN bootstrap \
  --repo-url=github.com/myorg/myproject

# Step 2: Create a release pull request
dagger call new --token env:GITHUB_TOKEN release-pr \
  --release-type=go \
  --repo-url=github.com/myorg/myproject

# Step 3: After merging the PR, create the GitHub release
dagger call new --token env:GITHUB_TOKEN github-release \
  --release-type=go \
  --repo-url=github.com/myorg/myproject
```

### Node.js Project Release

```bash
dagger call new --token env:GITHUB_TOKEN run \
  --release-type=node \
  --repo-url=github.com/myorg/myproject
```

### Python Project Release

```bash
dagger call new --token env:GITHUB_TOKEN run \
  --release-type=python \
  --repo-url=github.com/myorg/myproject
```

## Configuration

### Environment Variables

The module uses the following environment variable:
- `GITHUB_TOKEN`: Your GitHub personal access token (automatically handled by the module)

### Release Types

Release-please supports various release types that determine version bumping and changelog formatting:
- `node` - Node.js projects
- `python` - Python projects
- `go` - Go projects
- `rust` - Rust projects
- `java` - Java projects
- `ruby` - Ruby projects
- `php` - PHP projects
- `deno` - Deno projects

For a complete list and details, refer to the [release-please documentation](https://github.com/googleapis/release-please).

## Container Details

The module uses:
- **Base Image**: `node:current-alpine3.23`
- **Release-Please**: Installed globally via npm
- **Git**: Pre-installed in the image for repository operations

## Error Handling

All functions return:
- **Success**: A string containing the command output
- **Error**: An error object with details about what went wrong

Common errors:
- Invalid GitHub token: Check token permissions and validity
- Repository not found: Verify the `repo-url` format and existence
- Insufficient permissions: Ensure the GitHub token has necessary scopes

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Release
on:
  push:
    branches:
      - main

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: dagger/dagger-for-github@v5
        with:
          verb: call
          args: new --token env:GITHUB_TOKEN run --release-type=go --repo-url=github.com/${{ github.repository }}
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Local Usage in Scripts

```bash
#!/bin/bash
set -e

REPO_URL="github.com/myorg/myproject"
RELEASE_TYPE="go"
GITHUB_TOKEN="ghp_xxxxxxxxxxxx"

echo "Running release-please workflow..."
dagger call new --token env:GITHUB_TOKEN release-pr \
  --release-type=$RELEASE_TYPE \
  --repo-url=$REPO_URL

echo "Release PR created successfully!"
```

## Troubleshooting

### "release-please: command not found"
- The container installation may have failed
- Try rebuilding the module or check your internet connection

### Authentication Errors
- Verify your GitHub token is valid: `curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user`
- Ensure the token has appropriate scopes for the operations you're performing

### No Changes Released
- Check if there are actually new commits since the last release
- Verify conventional commits are being used (required by release-please)

## Further Reading

- [release-please GitHub Repository](https://github.com/googleapis/release-please)
- [release-please Documentation](https://github.com/googleapis/release-please/tree/main/docs)
- [Dagger Module Documentation](https://docs.dagger.io/mods)
