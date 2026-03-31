// A generated module for Go functions
//
// This module has been generated via dagger init and serves as a reference to
// basic module structure as you get started with Dagger.
//
// Two functions have been pre-created. You can modify, delete, or add to them,
// as needed. They demonstrate usage of arguments and return types using simple
// echo and grep commands. The functions can be called from the dagger CLI or
// from one of the SDKs.
//
// The first line in this comment block is a short description line and the
// rest is a long description with more detail on the module's purpose or usage,
// if appropriate. All modules should have a short description.

package main

import (
	"context"
	"dagger/go/internal/dagger"
	"fmt"
	"regexp"
)

// goVersionRegex is a regular expression to extract the Go version from the go.mod file.
// It looks for a line like "go 1.16" and captures the version number.
var goVersionRegex = regexp.MustCompile(`(?m)^go (\d+\.\d+(?:\.\d+)?)$`)

const (
	MOUNT_PATH        = "/src"
	DEBUG_CONTAINER   = "alpine:latest"
	RUNTIME_CONTAINER = "gcr.io/distroless/static-debian13"
)

type Go struct {
	Source *dagger.Directory
}

func New(
	// The source directory to use for Go commands. This should contain a go.mod file.
	// +optional
	// +defaultPath=.
	source *dagger.Directory,
) *Go {
	return &Go{Source: source}
}

// GoVersion returns the Go version specified in the go.mod file, or "unknown" if it cannot be determined.
func (g *Go) GoVersion(ctx context.Context) string {
	goModFile, err := g.Source.File("go.mod").Contents(ctx)
	if err != nil {
		panic("go.mod file not found in source directory")
	}
	version := "unknown"
	// Extract the Go version from the go.mod file if it exists.
	matches := goVersionRegex.FindStringSubmatch(goModFile)
	if len(matches) == 2 {
		version = matches[1]
	}
	return version
}

// Base returns a container with the Go source code mounted at /src and the working directory set to /src.
func (g *Go) Base(ctx context.Context) *dagger.Container {
	version := g.GoVersion(ctx)
	// Use the Go version specified in the go.mod file if it exists, otherwise use the latest version.
	return dag.Container().From("golang:"+version).WithDirectory(MOUNT_PATH, g.Source).WithWorkdir(MOUNT_PATH)
}

// Download runs the `go mod download` command in the source directory to download Go module dependencies.
func (g *Go) Download(ctx context.Context) *dagger.Container {
	// Check if go.mod and go.sum files exist in the source directory
	if exists, _ := g.Source.Exists(ctx, "go.mod"); !exists {
		panic("go.mod file not found in source directory")
	}
	if exists, _ := g.Source.Exists(ctx, "go.sum"); !exists {
		g.Base(ctx).WithExec([]string{"go", "mod", "tidy"})
	}
	return g.Base(ctx).WithExec([]string{"go", "mod", "download"})
}

// Lint runs the `golangci-lint run` command in the source directory to lint the Go code. It uses the latest version of golangci-lint.
func (g *Go) Lint(
	ctx context.Context,
	// Lint args to pass to the golangci-lint command (e.g., "--fast", "--enable=golint"). These are optional and default to an empty list.
	// +optional
	// +default=[]
	args []string,
	// Path to the Go code to lint. This is optional and defaults to the current directory (".").
	// +optional
	// +defaultPath=.
	path *dagger.Directory,
) (string, error) {
	g.Source = path
	commands := []string{"golangci-lint", "run"}
	commands = append(commands, args...)
	return dag.Container().From("golangci/golangci-lint:v2.11.4").WithDirectory(MOUNT_PATH, g.Source).WithWorkdir(MOUNT_PATH).WithExec(commands).Stdout(ctx)
}

// Test runs the `go test` command in the source directory.
// +check
func (g *Go) Test(
	ctx context.Context,
	// Test args
	// +optional
	// +default=[]
	args []string,
	// Path
	// +optional
	// +defaultPath=.
	path *dagger.Directory,
) (string, error) {
	g.Source = path
	commands := []string{"go", "test"}
	commands = append(commands, args...)
	// Run tests with the specified flags and arguments
	commands = append(commands, "./...")
	return g.Download(ctx).WithExec(commands).Stdout(ctx)
}

// Build runs the `go build` command in the source directory.
func (g *Go) Build(
	ctx context.Context,
	// The path to the Go package or file to build.
	// +optional
	// +default=.
	path string) *dagger.Directory {
	// CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s"
	return g.
		Download(ctx).
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOOS", "linux").
		WithExec([]string{"go", "build", "-ldflags=-w -s", "-o", "/out/app", path}).
		Directory("/out")
}

// Container builds the Go application and creates a container with the built binary.
func (g *Go) Container(
	ctx context.Context,
	// The path to the Go package or file to build.
	// +optional
	// +default=.
	path string,
) *dagger.Container {
	// Build the Go application
	binary := g.Build(ctx, path).File("app")
	// Create a container with the built binary
	return dag.Container().From(RUNTIME_CONTAINER).WithFile("/app", binary).WithWorkdir("/").WithEntrypoint([]string{"/app"})
}

// DebugContainer builds the Go application and creates a container with the built binary, using a debug-friendly base image.
func (g *Go) DebugContainer(
	ctx context.Context,
	// The path to the Go package or file to build.
	// +optional
	// +default=.
	path string,
) *dagger.Container {
	// Build the Go application
	binary := g.Build(ctx, path).File("app")
	// Create a container with the built binary
	return dag.Container().From(DEBUG_CONTAINER).WithFile("/app", binary).WithWorkdir("/").WithEntrypoint([]string{"/app"})
}

// Publish builds the Go application and pushes it to a container registry with the specified image name and registry credentials.
func (g *Go) Publish(
	ctx context.Context,
	// The path to the Go package or file to build.
	// +optional
	// +default=.
	path string,
	// imageName is the name of the container image to publish, including the tag (e.g., "my-nginx:latest").
	// +optional
	// +default=test:latest
	imageName string,
	// Registry is the registry to which the container should be pushed, defaulting to docker.io
	// +optional
	// +default=docker.io
	registry string,
	// Username for the container registry, if authentication is required.
	// +optional
	username string,
	// Password for the container registry, if authentication is required.
	// +optional
	password *dagger.Secret,
) (string, error) {
	return g.Container(ctx, path).WithRegistryAuth(registry, username, password).Publish(ctx, fmt.Sprintf("%s/%s", registry, imageName))
}
