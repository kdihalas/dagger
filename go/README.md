# Dagger Go Module

A Dagger module for building and containerizing Go applications. This module provides functions to extract Go versions, download dependencies, run tests, build binaries, and publish Docker images to container registries.

## Overview

The Go module automates common Go development tasks within a Dagger pipeline. It handles dependency management, testing, building statically-linked binaries, and containerizing applications with minimal setup.

**Key Features:**
- Automatic Go version detection from `go.mod`
- Dependency downloading with `go mod download`
- Test execution with customizable flags
- Static binary builds optimized for containerization
- Container image creation with distroless runtime
- Container registry publishing with authentication support

## Prerequisites

- **Go 1.26.1** or the version specified in your project's `go.mod`
- **go.mod file** in your source directory (required)
- **go.sum file** (auto-generated if missing via `go mod tidy`)
- **Dagger CLI** (v0.20.3 or compatible) - [Installation Guide](https://docs.dagger.io/install)
- For publishing: container registry credentials (Docker Hub, GCR, etc.)

## Installation

This module is part of the Dagger ecosystem and is initialized with a `dagger.json` file in the `go/` directory.

To use this module in your Dagger pipeline:

```bash
dagger install github.com/your-org/dagger-go-module@latest
```

Or reference it directly in your project configuration.

## Usage

All functions are called using the `dagger call` command. The module provides a `Go` struct initialized with your source directory.

### Basic Syntax

```bash
dagger call -m path/to/module [function] [flags]
```

## Available Functions

### `goVersion`

Returns the Go version specified in the `go.mod` file.

**Usage:**
```bash
dagger call go-version
```

**Output:**
```
1.26.1
```

**Example:**
```bash
# Get the Go version from your project
dagger call go-version
# Output: 1.26.1
```

---

### `base`

Creates a container with the Go source code mounted at `/src` and working directory set to `/src`. This is the foundation for other operations.

**Usage:**
```bash
dagger call base terminal
```

**Returns:** A Dagger Container with Go SDK, source code mounted, and working directory configured.

---

### `download`

Runs `go mod download` to download all Go module dependencies. Automatically runs `go mod tidy` if `go.sum` is missing.

**Usage:**
```bash
dagger call download terminal
```

**Example:**
```bash
# Download dependencies for your Go project
dagger call download terminal
# This will output the container shell for inspection if needed
```

---

### `test`

Runs `go test ./...` with optional custom arguments. Tests are executed in the context of a container with downloaded dependencies.

**Parameters:**
- `args` (optional, default: empty list): Additional test flags and arguments

**Usage:**
```bash
dagger call test
```

**Examples:**
```bash
# Run all tests with defaults
dagger call test

# Run tests with verbose output
dagger call test --args="-v"

# Run tests with coverage
dagger call test --args="-cover"

# Run specific tests with verbose and coverage
dagger call test --args="-v" --args="-cover" --args="-run=TestSpecificFunc"

# Run tests with race detector
dagger call test --args="-race"
```

---

### `build`

Builds a statically-linked Linux binary from your Go source code. Creates an optimized binary suitable for containerization.

**Parameters:**
- `path` (optional, default: `.`): The path to the Go package or file to build

**Configuration:**
- Environment: `CGO_ENABLED=0`, `GOOS=linux`
- Flags: `-ldflags="-w -s"` (removes debug symbols for minimal size)
- Output: Binary is placed at `/out/app` in the container

**Usage:**
```bash
dagger call build
```

**Returns:** A Dagger Directory containing the built `app` binary.

**Examples:**
```bash
# Build the default package (current directory)
dagger call build

# Build a specific package
dagger call build --path="./cmd/myapp"

# Build and export the binary locally
dagger call build \
  --path="./cmd/server" \
  export --path="./bin"
```

---

### `container`

Builds your Go application and creates a minimal production container using `distroless/static-debian13` as the base image.

**Parameters:**
- `path` (optional, default: `.`): The path to the Go package or file to build

**Features:**
- Minimal image size (distroless runtime)
- Binary runs as `/app` with entrypoint configured
- No shell, package manager, or unnecessary tools

**Usage:**
```bash
dagger call container
```

**Returns:** A Dagger Container ready to publish or run.

**Examples:**
```bash
# Create a container for the default package
dagger call container

# Create a container for a specific package
dagger call container --path="./cmd/api"

# Inspect the container's filesystem
dagger call container terminal

# Run the container locally
dagger call container publish \
  --address="localhost:5000/myapp:latest"
```

---

### `debugContainer`

Similar to `container`, but uses `alpine:latest` as the base image instead of distroless. Useful for development and debugging.

**Parameters:**
- `path` (optional, default: `.`): The path to the Go package or file to build

**Features:**
- Alpine Linux base image with shell access
- Includes package manager and debugging utilities
- Larger image size than production container

**Usage:**
```bash
dagger call debug-container
```

**Examples:**
```bash
# Create a debug container
dagger call debug-container

# Create a debug container and open a terminal
dagger call debug-container terminal

# Run the debug container locally
dagger call debug-container publish \
  --address="localhost:5000/myapp:debug"
```

---

### `publish`

Builds the Go application, creates a container, and publishes it to a container registry with authentication support.

**Parameters:**
- `path` (optional, default: `.`): The path to the Go package or file to build
- `imageName` (optional, default: `test:latest`): Image name with tag (e.g., `myapp:1.0.0`)
- `registry` (optional, default: `docker.io`): Registry hostname (e.g., `docker.io`, `gcr.io`, `ghcr.io`)
- `username` (optional): Registry username for authentication
- `password` (optional): Registry password as a Secret

**Usage:**
```bash
dagger call publish \
  --registry="docker.io" \
  --image-name="myusername/myapp:latest" \
  --username="myusername" \
  --password=env:DOCKER_PASSWORD
```

**Returns:** The published image reference as a string.

**Examples:**

**Docker Hub:**
```bash
dagger call publish \
  --path="./cmd/server" \
  --registry="docker.io" \
  --image-name="myusername/myapp:1.0.0" \
  --username="myusername" \
  --password=env:DOCKER_PASSWORD
```

**Google Container Registry:**
```bash
dagger call publish \
  --registry="gcr.io" \
  --image-name="my-project/myapp:latest" \
  --username="_json_key" \
  --password=env:GCR_JSON_KEY
```

**GitHub Container Registry:**
```bash
dagger call publish \
  --registry="ghcr.io" \
  --image-name="myusername/myapp:latest" \
  --username="myusername" \
  --password=env:GITHUB_TOKEN
```

**No authentication (local/public registry):**
```bash
dagger call publish \
  --registry="localhost:5000" \
  --image-name="myapp:latest"
```

---

## Complete Workflow Examples

### Example 1: Build and Test a Go Application

```bash
# Run tests
dagger call test --args="-v" --args="-cover"

# Build the binary if tests pass
dagger call build --path="./cmd/myapp"
```

### Example 2: Build and Containerize a Go Server

```bash
# Create a production container
dagger call container --path="./cmd/server"

# Create a debug container for development
dagger call debug-container --path="./cmd/server"
```

### Example 3: Complete CI/CD Pipeline

```bash
#!/bin/bash
set -e

# Install the module
dagger install github.com/your-org/dagger-go-module@latest

# Run tests
echo "Running tests..."
dagger call test --args="-v" --args="-race"

# Build and publish to Docker Hub
echo "Building and publishing..."
dagger call publish \
  --path="./cmd/server" \
  --registry="docker.io" \
  --image-name="myusername/myserver:$(git describe --tags)" \
  --username="$DOCKER_USERNAME" \
  --password=env:DOCKER_PASSWORD

echo "Published successfully!"
```

### Example 4: Multi-Stage Setup

```bash
# Download dependencies first (useful for caching in CI/CD)
dagger call download

# Get Go version
GO_VERSION=$(dagger call go-version)
echo "Building with Go $GO_VERSION"

# Run tests
dagger call test --args="-cover"

# Build production binary
dagger call build --path="./cmd/api"
```

## Configuration

### Container Defaults

The module uses predefined container images:
- **Build Base**: `golang:VERSION` (where VERSION is from your `go.mod`)
- **Production Runtime**: `gcr.io/distroless/static-debian13`
- **Debug Runtime**: `alpine:latest`

### Build Flags

The `build` function uses these default flags:
- `CGO_ENABLED=0`: Disable cgo for static linking
- `GOOS=linux`: Target Linux OS
- `ldflags=-w -s`: Remove debug symbols for minimal binary size

### Mount Path

Source code is mounted at `/src` within containers. Working directory is set to `/src`.

## Environment Variables

For publishing to private registries, pass credentials as Dagger Secrets:

```bash
# Using environment variables
dagger call publish \
  --username="user" \
  --password=env:REGISTRY_PASSWORD

# Using CLI flags (not recommended for sensitive data)
dagger call publish \
  --username="user" \
  --password=env:MY_PASSWORD
```

## Troubleshooting

### go.mod file not found
Ensure your source directory contains a `go.mod` file.

```bash
# Check if go.mod exists
ls -la go.mod

# If missing, initialize one
go mod init github.com/myusername/myproject
```

### Build fails with "go.sum not found"
The module automatically runs `go mod tidy` to generate `go.sum` if missing. If this fails:

```bash
# Manually tidy dependencies
go mod tidy

# Commit the updated files
git add go.mod go.sum
git commit -m "Update dependencies"
```

### Container registry authentication fails
Verify your credentials and registry format:

```bash
# Test Docker login locally first
docker login docker.io

# Ensure password is provided as environment variable
echo $DOCKER_PASSWORD | dagger call publish --registry="docker.io" ...
```

### Binary size too large
The `build` function already uses `-ldflags="-w -s"` for optimization. If needed, use the debug container for development and the production container for releases.

## Module Information

- **Engine Version**: v0.20.3
- **SDK**: Go
- **Go Version**: 1.26.1
- **Location**: `go/` directory in the repository

## Contributing

To extend this module, modify `/var/home/kostas/dev/dagger/go/main.go` and regenerate the module:

```bash
dagger develop
```

## License

This module is provided as-is. See the LICENSE file in the `go/` directory for details.

## Related Resources

- [Dagger Documentation](https://docs.dagger.io/)
- [Go Official Documentation](https://golang.org/doc/)
- [Distroless Container Images](https://github.com/GoogleContainerTools/distroless)
- [Alpine Linux](https://alpinelinux.org/)
