# Release-Please Module

Automate GitHub releases with release-please using Dagger.

## Requirements

- Dagger v0.20.3 or later
- GitHub token with `repo` and `workflows` permissions

## Functions

| Function | Description | Key Parameters |
|----------|-------------|-----------------|
| `Container` | Container with release-please installed | — |
| `ReleasePr` | Create a release pull request | `releaseType` (required), `repoUrl` (required) |
| `GithubRelease` | Create a GitHub release from merged PR | `releaseType` (required), `repoUrl` (required) |
| `Run` | Create release PR then release (both operations) | `releaseType` (required), `repoUrl` (required) |
| `Bootstrap` | Initialize release-please in repository | `repoUrl` (required) |

## Supported Release Types

`dart`, `elixir`, `go`, `helm`, `java`, `krm-blueprint`, `maven`, `node`, `ocaml`, `php`, `python`, `ruby`, `rust`, `salesforce`, `simple`, `terraform-module`

## Usage Examples

Initialize release-please in a repository:

```bash
dagger call -m release-please --token env:GITHUB_TOKEN bootstrap \
  --repo-url github.com/owner/repo
```

Create a release PR:

```bash
dagger call -m release-please --token env:GITHUB_TOKEN release-pr \
  --release-type go \
  --repo-url github.com/owner/repo
```

Create a GitHub release (after PR is merged):

```bash
dagger call -m release-please --token env:GITHUB_TOKEN github-release \
  --release-type go \
  --repo-url github.com/owner/repo
```

Create release PR and release in one command:

```bash
dagger call -m release-please --token env:GITHUB_TOKEN run \
  --release-type go \
  --repo-url github.com/owner/repo
```

For a Node.js repository:

```bash
dagger call -m release-please --token env:GITHUB_TOKEN release-pr \
  --release-type node \
  --repo-url github.com/owner/repo
```

## GitHub Actions

Use the release-please module in GitHub Actions to automate releases:

```yaml
name: Release

on:
  push:
    branches: [main]

permissions:
  id-token: write
  contents: write
  pull-requests: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Create release PR
        run: dagger -m release-please \
          --token ${{ secrets.GITHUB_TOKEN }} \
          call release-pr \
          --release-type go \
          --repo-url github.com/${{ github.repository }}
      - name: Create GitHub release
        run: dagger -m release-please \
          --token ${{ secrets.GITHUB_TOKEN }} \
          call github-release \
          --release-type go \
          --repo-url github.com/${{ github.repository }}
```
