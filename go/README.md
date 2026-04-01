# Go

Build and containerize Go applications with automated version detection, dependency management, testing, and registry publishing.

## Requirements

- Go 1.26.1 or the version specified in your `go.mod`
- `go.mod` file (required)
- `go.sum` file (auto-generated if missing)
- Dagger CLI v0.20.3 or compatible
- Container registry credentials (for publishing)

## Functions

### `goVersion`

Returns the Go version from `go.mod`.

**Example:**
```sh
dagger call go-version
```

### `base`

Container with Go SDK, source mounted at `/src`, working directory `/src`.

**Example:**
```sh
dagger call base terminal
```

### `download`

Runs `go mod download` and `go mod tidy` if needed.

**Example:**
```sh
dagger call download terminal
```

### `test`

Runs `go test ./...` with optional custom arguments.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| args | []string | [] | Additional test flags (e.g., `-v`, `-cover`, `-race`) |

**Example:**
```sh
dagger call test --args="-v" --args="-cover"
```

### `lint`

Runs `golangci-lint run` with optional custom arguments.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| args | []string | [] | Additional lint flags (e.g., `--fast`, `--enable=golint`) |

**Example:**
```sh
dagger call lint --args="--fast"
```

### `build`

Builds a statically-linked Linux binary. Output at `/out/app`.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| path | string | . | Go package or file path to build |

**Example:**
```sh
dagger call build --path="./cmd/myapp" export --path="./bin"
```

### `container`

Creates a production container with distroless base image.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| path | string | . | Go package or file path to build |

**Example:**
```sh
dagger call container publish --address="docker.io/myuser/myapp:latest"
```

### `debugContainer`

Creates a debug container with Alpine Linux base image.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| path | string | . | Go package or file path to build |

**Example:**
```sh
dagger call debug-container terminal
```

### `publish`

Builds and publishes container to a registry.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| path | string | . | Go package or file path to build |
| image-name | []string | - | Image names with tags (e.g., `myuser/myapp:1.0.0`) |
| registry | string | docker.io | Registry hostname |
| username | string | - | Registry username (for authentication) |
| password | Secret | - | Registry password (for authentication) |

**Example:**
```sh
dagger call publish \
  --image-name="myuser/myapp:latest" \
  --registry="docker.io" \
  --username="myuser" \
  --password=env:DOCKER_PASSWORD
```

## Examples

**Run tests with coverage:**
```sh
dagger call test --args="-cover" --args="-race"
```

**Build and export binary:**
```sh
dagger call build --path="./cmd/server" export --path="./bin"
```

**Build and run debug container:**
```sh
dagger call debug-container terminal
```

**Publish to Docker Hub:**
```sh
dagger call publish \
  --registry="docker.io" \
  --image-name="myuser/myserver:latest" \
  --username="myuser" \
  --password=env:DOCKER_PASSWORD
```

**Publish to Google Container Registry:**
```sh
dagger call publish \
  --registry="gcr.io" \
  --image-name="my-project/myapp:latest" \
  --username="_json_key" \
  --password=env:GCR_JSON_KEY
```
