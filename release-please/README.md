# ReleasePlease

Automate GitHub releases and release pull requests using release-please.

## Requirements

- GitHub token with appropriate permissions
  - Public repos: `contents: read/write`, `pull-requests: read/write`
  - Private repos: add `repo: read/write`
  - Releases: `contents: read/write`
- Dagger CLI v0.20.3 or compatible
- Conventional commits in repository

## Functions

### `container`

Returns a container with release-please pre-installed.

**Example:**
```sh
dagger call --token enn://GITHUB_TOKEN container terminal
```

### `bootstrap`

Initializes release-please in a repository.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| repo-url | string | - | GitHub repository URL (e.g., `github.com/owner/repo`) |

**Example:**
```sh
dagger call --token env://GITHUB_TOKEN bootstrap --repo-url=github.com/myorg/myproject
```

### `releasePr`

Creates a release pull request with automated version bumping and changelog.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| release-type | string | - | Release type: `node`, `python`, `go`, `rust`, `java`, `ruby`, `php`, `deno` |
| repo-url | string | - | GitHub repository URL |

**Example:**
```sh
dagger call --token env://GITHUB_TOKEN release-pr \
  --release-type=go \
  --repo-url=github.com/myorg/myproject
```

### `githubRelease`

Creates a GitHub release based on changelog entries.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| release-type | string | - | Release type |
| repo-url | string | - | GitHub repository URL |

**Example:**
```sh
dagger call --token env://GITHUB_TOKEN github-release \
  --release-type=go \
  --repo-url=github.com/myorg/myproject
```

### `run`

Executes full release workflow: creates release and PR.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| release-type | string | - | Release type |
| repo-url | string | - | GitHub repository URL |

**Example:**
```sh
dagger call --token env://GITHUB_TOKEN run \
  --release-type=go \
  --repo-url=github.com/myorg/myproject
```

## Examples

**Bootstrap a Go project:**
```sh
dagger call --token env://GITHUB_TOKEN bootstrap \
  --repo-url=github.com/myorg/myproject
```

**Create release PR:**
```sh
dagger call --token env://GITHUB_TOKEN release-pr \
  --release-type=go \
  --repo-url=github.com/myorg/myproject
```

**Create release after PR merges:**
```sh
dagger call --token env://GITHUB_TOKEN github-release \
  --release-type=go \
  --repo-url=github.com/myorg/myproject
```