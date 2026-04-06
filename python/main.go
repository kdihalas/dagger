// Build, test, lint, and containerize Python applications.
package main

import (
	"context"
	"dagger/python/internal/dagger"
	"fmt"
	"regexp"
)

var validImageRef = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-/]*(:[a-zA-Z0-9._\-]+)?$`)

const (
	mountPath   = "/src"
	debugBase   = "python:3-alpine"
	runtimeBase = "gcr.io/distroless/python3-debian12"
)

type packageManager int

const (
	pmPip packageManager = iota
	pmUv
	pmPoetry
)

type Python struct {
	Source *dagger.Directory
}

func New(
	// +optional
	// +defaultPath=.
	source *dagger.Directory,
) *Python {
	return &Python{Source: source}
}

func (p *Python) detectPackageManager(ctx context.Context) packageManager {
	if exists, _ := p.Source.Exists(ctx, "uv.lock"); exists {
		return pmUv
	}
	if exists, _ := p.Source.Exists(ctx, "pyproject.toml"); exists {
		contents, err := p.Source.File("pyproject.toml").Contents(ctx)
		if err == nil && regexp.MustCompile(`\[tool\.poetry\]`).MatchString(contents) {
			return pmPoetry
		}
	}
	return pmPip
}

// PythonVersion extracts the Python version from .python-version file if present.
func (p *Python) PythonVersion(ctx context.Context) (string, error) {
	if exists, _ := p.Source.Exists(ctx, ".python-version"); exists {
		contents, err := p.Source.File(".python-version").Contents(ctx)
		if err != nil {
			return "", fmt.Errorf("reading .python-version: %w", err)
		}
		version := regexp.MustCompile(`\s+`).ReplaceAllString(contents, "")
		if version != "" {
			return version, nil
		}
	}
	return "3", nil
}

// Base returns a container with Python and the source mounted.
func (p *Python) Base(ctx context.Context) (*dagger.Container, error) {
	version, err := p.PythonVersion(ctx)
	if err != nil {
		return nil, err
	}
	ctr := dag.Container().
		From("python:"+version+"-slim").
		WithDirectory(mountPath, p.Source).
		WithWorkdir(mountPath)

	pm := p.detectPackageManager(ctx)
	switch pm {
	case pmUv:
		ctr = ctr.WithExec([]string{"pip", "install", "uv"})
	case pmPoetry:
		ctr = ctr.WithExec([]string{"pip", "install", "poetry"}).
			WithExec([]string{"poetry", "config", "virtualenvs.create", "false"})
	}
	return ctr, nil
}

// Install runs the package manager install command.
func (p *Python) Install(ctx context.Context) (*dagger.Container, error) {
	base, err := p.Base(ctx)
	if err != nil {
		return nil, err
	}
	pm := p.detectPackageManager(ctx)
	switch pm {
	case pmUv:
		return base.WithExec([]string{"uv", "sync", "--frozen"}).Sync(ctx)
	case pmPoetry:
		return base.WithExec([]string{"poetry", "install", "--no-interaction"}).Sync(ctx)
	default:
		if exists, _ := p.Source.Exists(ctx, "requirements.txt"); exists {
			return base.WithExec([]string{"pip", "install", "-r", "requirements.txt"}).Sync(ctx)
		}
		return base.WithExec([]string{"pip", "install", "."}).Sync(ctx)
	}
}

// Lint runs a Python linter. Uses ruff by default.
func (p *Python) Lint(
	ctx context.Context,
	// Linter command to run.
	// +optional
	// +default=ruff
	linter string,
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	ctr, err := p.Install(ctx)
	if err != nil {
		return "", err
	}
	ctr = ctr.WithExec([]string{"pip", "install", linter})
	cmd := []string{linter, "check", "."}
	cmd = append(cmd, args...)
	return ctr.WithExec(cmd).Stdout(ctx)
}

// Test runs pytest with optional args.
// +check
func (p *Python) Test(
	ctx context.Context,
	// Test runner command.
	// +optional
	// +default=pytest
	runner string,
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	ctr, err := p.Install(ctx)
	if err != nil {
		return "", err
	}
	ctr = ctr.WithExec([]string{"pip", "install", runner})
	cmd := []string{runner}
	cmd = append(cmd, args...)
	return ctr.WithExec(cmd).Stdout(ctx)
}

// Build runs pip wheel or poetry build and returns the dist directory.
func (p *Python) Build(
	ctx context.Context,
	// Output directory name.
	// +optional
	// +default=dist
	outDir string,
) (*dagger.Directory, error) {
	ctr, err := p.Install(ctx)
	if err != nil {
		return nil, err
	}
	pm := p.detectPackageManager(ctx)
	switch pm {
	case pmUv:
		ctr = ctr.WithExec([]string{"uv", "build", "--out-dir", outDir})
	case pmPoetry:
		ctr = ctr.WithExec([]string{"poetry", "build", "-o", outDir})
	default:
		ctr = ctr.WithExec([]string{"pip", "wheel", "--wheel-dir", outDir, "."})
	}
	return ctr.Directory(mountPath + "/" + outDir), nil
}

// Container creates a distroless production container with the application installed.
func (p *Python) Container(
	ctx context.Context,
	// Entrypoint command.
	// +optional
	// +default=python
	entrypoint string,
	// Arguments passed to the entrypoint.
	// +optional
	// +default=[]
	entrypointArgs []string,
) (*dagger.Container, error) {
	ctr, err := p.Install(ctx)
	if err != nil {
		return nil, err
	}
	// Copy installed site-packages from the build container
	sitePackages := ctr.Directory("/usr/local/lib")
	appDir := ctr.Directory(mountPath)

	ep := []string{entrypoint}
	ep = append(ep, entrypointArgs...)

	return dag.Container().
		From(runtimeBase).
		WithDirectory("/usr/local/lib", sitePackages).
		WithDirectory("/app", appDir).
		WithWorkdir("/app").
		WithEntrypoint(ep), nil
}

// DebugContainer creates an Alpine-based container for debugging.
func (p *Python) DebugContainer(
	ctx context.Context,
	// +optional
	// +default=python
	entrypoint string,
	// +optional
	// +default=[]
	entrypointArgs []string,
) (*dagger.Container, error) {
	ctr, err := p.Install(ctx)
	if err != nil {
		return nil, err
	}
	sitePackages := ctr.Directory("/usr/local/lib")
	appDir := ctr.Directory(mountPath)

	ep := []string{entrypoint}
	ep = append(ep, entrypointArgs...)

	return dag.Container().
		From(debugBase).
		WithDirectory("/usr/local/lib", sitePackages).
		WithDirectory("/app", appDir).
		WithWorkdir("/app").
		WithEntrypoint(ep), nil
}

// Publish builds and pushes the container to a registry, returning published references.
func (p *Python) Publish(
	ctx context.Context,
	// +optional
	// +default=python
	entrypoint string,
	// +optional
	// +default=[]
	entrypointArgs []string,
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

	ctr, err := p.Container(ctx, entrypoint, entrypointArgs)
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
