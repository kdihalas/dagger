# Go Module

Build, test, lint, and containerize Go applications with Dagger.

## Requirements

- Dagger v0.20.3 or later
- Go v1.18+

## Functions

| Function | Description | Key Parameters |
|----------|-------------|-----------------|
| `GoVersion` | Extract Go version from go.mod | — |
| `Base` | Container with Go and source mounted | — |
| `Download` | Run `go mod download` (with tidy if needed) | — |
| `Lint` | Run golangci-lint | `args` ([]string, optional) |
| `Test` | Run `go test ./...` | `args` ([]string, optional) |
| `Build` | Build statically-linked Linux binary | `path` (string, optional, default: `.`) |
| `Container` | Build production container (distroless) | `path` (string, optional, default: `.`) |
| `DebugContainer` | Build Alpine-based debug container | `path` (string, optional, default: `.`) |
| `Publish` | Build and push container to registry | `path`, `imageName` ([]string, required), `registry`, `username`, `password` |

## Usage Examples

Extract Go version:

```bash
dagger call -m github.com/kdihalas/dagger/go --source . go-version
```

Run tests:

```bash
dagger call -m github.com/kdihalas/dagger/go --source . test
```

Run tests with arguments:

```bash
dagger call -m github.com/kdihalas/dagger/go --source . test --args "-v" --args "-race"
```

Run linter:

```bash
dagger call -m github.com/kdihalas/dagger/go --source . lint
```

Build a binary:

```bash
dagger call -m github.com/kdihalas/dagger/go --source . build
```

Build a production container:

```bash
dagger call -m github.com/kdihalas/dagger/go --source . container
```

Build a debug container (Alpine + shell):

```bash
dagger call -m github.com/kdihalas/dagger/go --source . debug-container
```

Publish to a registry:

```bash
dagger call -m github.com/kdihalas/dagger/go --source . publish \
  --image-name myapp:latest \
  --registry docker.io \
  --username myuser \
  --password env:DOCKER_PASSWORD
```

Publish multiple image tags:

```bash
dagger call -m github.com/kdihalas/dagger/go --source . publish \
  --image-name myapp:latest,myapp:v1.0.0 \
  --registry docker.io \
  --username myuser \
  --password env:DOCKER_PASSWORD
```

## GitHub Actions

Use the Go module in GitHub Actions to build, test, and lint your Go applications:

```yaml
name: Build Go App

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test-and-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Run tests
        run: dagger -m github.com/kdihalas/dagger/go call --source . test
      - name: Run linter
        run: dagger -m github.com/kdihalas/dagger/go call --source . lint
      - name: Build binary
        run: dagger -m github.com/kdihalas/dagger/go call --source . build
      - name: Build container
        run: dagger -m github.com/kdihalas/dagger/go call --source . container --path .
      - name: Publish container
        run: dagger -m github.com/kdihalas/dagger/go call --source . publish \
          --path . \
          --image-name myapp:latest \
          --registry docker.io \
          --username ${{ secrets.DOCKER_USERNAME }} \
          --password ${{ secrets.DOCKER_PASSWORD }}
```
