// Build, test, lint, and containerize Go applications.
package main

import (
	"context"
	"dagger/go/internal/dagger"
	"fmt"
	"regexp"
)

var (
	goVersionRegex = regexp.MustCompile(`(?m)^go (\d+\.\d+(?:\.\d+)?)$`)
	validImageRef  = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-/]*(:[a-zA-Z0-9._\-]+)?$`)
)

const (
	mountPath        = "/src"
	debugBase        = "alpine:latest"
	runtimeBase      = "gcr.io/distroless/static-debian13"
	golangciLintTag  = "v2.11.4"
)

type Go struct {
	Source *dagger.Directory
}

func New(
	// +optional
	// +defaultPath=.
	source *dagger.Directory,
) *Go {
	return &Go{Source: source}
}

// GoVersion extracts the Go version from go.mod.
func (g *Go) GoVersion(ctx context.Context) (string, error) {
	contents, err := g.Source.File("go.mod").Contents(ctx)
	if err != nil {
		return "", fmt.Errorf("reading go.mod: %w", err)
	}
	matches := goVersionRegex.FindStringSubmatch(contents)
	if len(matches) != 2 {
		return "", fmt.Errorf("could not determine Go version from go.mod")
	}
	return matches[1], nil
}

// Base returns a container with Go and the source mounted.
func (g *Go) Base(ctx context.Context) (*dagger.Container, error) {
	version, err := g.GoVersion(ctx)
	if err != nil {
		return nil, err
	}
	return dag.Container().
		From("golang:"+version).
		WithDirectory(mountPath, g.Source).
		WithWorkdir(mountPath), nil
}

// Download runs go mod download, running go mod tidy first if go.sum is missing.
func (g *Go) Download(ctx context.Context) (*dagger.Container, error) {
	base, err := g.Base(ctx)
	if err != nil {
		return nil, err
	}
	if exists, _ := g.Source.Exists(ctx, "go.sum"); !exists {
		base = base.WithExec([]string{"go", "mod", "tidy"})
	}
	return base.WithExec([]string{"go", "mod", "download"}), nil
}

// Lint runs golangci-lint.
func (g *Go) Lint(
	ctx context.Context,
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	cmd := []string{"golangci-lint", "run"}
	cmd = append(cmd, args...)
	return dag.Container().
		From("golangci/golangci-lint:"+golangciLintTag).
		WithDirectory(mountPath, g.Source).
		WithWorkdir(mountPath).
		WithExec(cmd).
		Stdout(ctx)
}

// Test runs go test ./... with optional args.
// +check
func (g *Go) Test(
	ctx context.Context,
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	ctr, err := g.Download(ctx)
	if err != nil {
		return "", err
	}
	cmd := []string{"go", "test"}
	cmd = append(cmd, args...)
	cmd = append(cmd, "./...")
	return ctr.WithExec(cmd).Stdout(ctx)
}

// Build compiles a statically-linked Linux binary.
func (g *Go) Build(
	ctx context.Context,
	// +optional
	// +default=.
	path string,
) (*dagger.Directory, error) {
	ctr, err := g.Download(ctx)
	if err != nil {
		return nil, err
	}
	return ctr.
		WithEnvVariable("CGO_ENABLED", "0").
		WithEnvVariable("GOOS", "linux").
		WithExec([]string{"go", "build", "-ldflags=-w -s", "-o", "/out/app", path}).
		Directory("/out"), nil
}

// Container builds the app into a distroless production container.
func (g *Go) Container(
	ctx context.Context,
	// +optional
	// +default=.
	path string,
) (*dagger.Container, error) {
	out, err := g.Build(ctx, path)
	if err != nil {
		return nil, err
	}
	return dag.Container().
		From(runtimeBase).
		WithFile("/app", out.File("app")).
		WithEntrypoint([]string{"/app"}), nil
}

// DebugContainer builds the app into an Alpine-based container for debugging.
func (g *Go) DebugContainer(
	ctx context.Context,
	// +optional
	// +default=.
	path string,
) (*dagger.Container, error) {
	out, err := g.Build(ctx, path)
	if err != nil {
		return nil, err
	}
	return dag.Container().
		From(debugBase).
		WithFile("/app", out.File("app")).
		WithEntrypoint([]string{"/app"}), nil
}

// Publish builds and pushes the container to a registry, returning published references.
func (g *Go) Publish(
	ctx context.Context,
	// +optional
	// +default=.
	path string,
	// Image names including tags (e.g., "myapp:latest").
	// +required
	imageName []string,
	// +optional
	// +default=docker.io
	registry string,
	// +optional
	username string,
	// +optional
	password *dagger.Secret,
) ([]string, error) {
	if len(imageName) == 0 {
		return nil, fmt.Errorf("at least one imageName is required")
	}
	for _, name := range imageName {
		if !validImageRef.MatchString(name) {
			return nil, fmt.Errorf("invalid image reference: %q", name)
		}
	}

	ctr, err := g.Container(ctx, path)
	if err != nil {
		return nil, err
	}
	ctr = ctr.WithRegistryAuth(registry, username, password)

	var refs []string
	for _, name := range imageName {
		ref, err := ctr.Publish(ctx, fmt.Sprintf("%s/%s", registry, name))
		if err != nil {
			return refs, fmt.Errorf("publishing %s: %w", name, err)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}
