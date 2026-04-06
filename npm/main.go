// Build, test, lint, and containerize Node.js applications.
package main

import (
	"context"
	"dagger/npm/internal/dagger"
	"encoding/json"
	"fmt"
	"regexp"
)

var validImageRef = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._\-/]*(:[a-zA-Z0-9._\-]+)?$`)

const (
	mountPath   = "/src"
	debugBase   = "node:current-alpine3.23"
	runtimeBase = "gcr.io/distroless/nodejs22-debian12"
)

type packageManager int

const (
	pmNpm packageManager = iota
	pmPnpm
	pmYarn
)

func (pm packageManager) cmd() string {
	switch pm {
	case pmPnpm:
		return "pnpm"
	case pmYarn:
		return "yarn"
	default:
		return "npm"
	}
}

type Npm struct {
	Source *dagger.Directory
}

func New(
	// +optional
	// +defaultPath=.
	source *dagger.Directory,
) *Npm {
	return &Npm{Source: source}
}

// NodeVersion extracts the Node.js version from package.json engines field.
func (n *Npm) NodeVersion(ctx context.Context) (string, error) {
	contents, err := n.Source.File("package.json").Contents(ctx)
	if err != nil {
		return "", fmt.Errorf("reading package.json: %w", err)
	}
	var pkg struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}
	if err := json.Unmarshal([]byte(contents), &pkg); err != nil {
		return "", fmt.Errorf("parsing package.json: %w", err)
	}
	if pkg.Engines.Node == "" {
		return "current", nil
	}
	return pkg.Engines.Node, nil
}

func (n *Npm) detectPackageManager(ctx context.Context) packageManager {
	if exists, _ := n.Source.Exists(ctx, "pnpm-lock.yaml"); exists {
		return pmPnpm
	}
	if exists, _ := n.Source.Exists(ctx, "yarn.lock"); exists {
		return pmYarn
	}
	return pmNpm
}

// Base returns a container with Node.js and the source mounted.
func (n *Npm) Base(ctx context.Context) (*dagger.Container, error) {
	version, err := n.NodeVersion(ctx)
	if err != nil {
		return nil, err
	}
	ctr := dag.Container().
		From("node:"+version+"-alpine").
		WithDirectory(mountPath, n.Source).
		WithWorkdir(mountPath)

	pm := n.detectPackageManager(ctx)
	if pm == pmPnpm {
		ctr = ctr.WithExec([]string{"npm", "i", "-g", "pnpm"})
	}
	return ctr, nil
}

// Install runs the package manager install command.
func (n *Npm) Install(ctx context.Context) (*dagger.Container, error) {
	base, err := n.Base(ctx)
	if err != nil {
		return nil, err
	}
	pm := n.detectPackageManager(ctx)
	switch pm {
	case pmPnpm:
		return base.WithExec([]string{"pnpm", "install", "--frozen-lockfile"}).
			WithExec([]string{"ls", "-la", "node_modules"}).
			Sync(ctx)
	case pmYarn:
		return base.WithExec([]string{"yarn", "install", "--frozen-lockfile"}).
			WithExec([]string{"ls", "-la", "node_modules"}).
			Sync(ctx)
	default:
		return base.WithExec([]string{"npm", "ci"}).
			WithExec([]string{"ls", "-la", "node_modules"}).
			Sync(ctx)
	}
}

// Lint runs the lint script defined in package.json.
func (n *Npm) Lint(
	ctx context.Context,
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	ctr, err := n.Install(ctx)
	if err != nil {
		return "", err
	}
	pm := n.detectPackageManager(ctx)
	cmd := []string{pm.cmd(), "run", "lint"}
	if pm == pmNpm && len(args) > 0 {
		cmd = append(cmd, "--")
	}
	cmd = append(cmd, args...)
	return ctr.WithExec(cmd).Stdout(ctx)
}

// Test runs the test script defined in package.json.
// +check
func (n *Npm) Test(
	ctx context.Context,
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	ctr, err := n.Install(ctx)
	if err != nil {
		return "", err
	}
	pm := n.detectPackageManager(ctx)
	cmd := []string{pm.cmd(), "run", "test"}
	if pm == pmNpm && len(args) > 0 {
		cmd = append(cmd, "--")
	}
	cmd = append(cmd, args...)
	return ctr.WithExec(cmd).Stdout(ctx)
}

// Build runs the build script defined in package.json and returns the output directory.
func (n *Npm) Build(
	ctx context.Context,
	// Output directory name (e.g., "dist", "build").
	// +optional
	// +default=dist
	outDir string,
) (*dagger.Directory, error) {
	ctr, err := n.Install(ctx)
	if err != nil {
		return nil, err
	}
	pm := n.detectPackageManager(ctx)
	built := ctr.WithExec([]string{pm.cmd(), "run", "build"})
	return built.Directory(mountPath + "/" + outDir), nil
}

// Container builds the app and creates a distroless production container.
func (n *Npm) Container(
	ctx context.Context,
	// Output directory name from the build step.
	// +optional
	// +default=dist
	outDir string,
	// Entrypoint file inside the output directory.
	// +optional
	// +default=index.js
	entrypoint string,
) (*dagger.Container, error) {
	out, err := n.Build(ctx, outDir)
	if err != nil {
		return nil, err
	}
	return dag.Container().
		From(runtimeBase).
		WithDirectory("/app", out).
		WithWorkdir("/app").
		WithEntrypoint([]string{entrypoint}), nil
}

// DebugContainer builds the app into an Alpine Node.js container for debugging.
func (n *Npm) DebugContainer(
	ctx context.Context,
	// +optional
	// +default=dist
	outDir string,
	// +optional
	// +default=index.js
	entrypoint string,
) (*dagger.Container, error) {
	out, err := n.Build(ctx, outDir)
	if err != nil {
		return nil, err
	}
	return dag.Container().
		From(debugBase).
		WithDirectory("/app", out).
		WithWorkdir("/app").
		WithEntrypoint([]string{"node", entrypoint}), nil
}

// Publish builds and pushes the container to a registry, returning published references.
func (n *Npm) Publish(
	ctx context.Context,
	// +optional
	// +default=dist
	outDir string,
	// +optional
	// +default=index.js
	entrypoint string,
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

	ctr, err := n.Container(ctx, outDir, entrypoint)
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
