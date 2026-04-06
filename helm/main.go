// Manage Helm chart lifecycle: template, lint, package, push, install, upgrade, rollback, and uninstall.
package main

import (
	"context"
	"dagger/helm/internal/dagger"
	"fmt"
	"regexp"
)

var (
	validReleaseName = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,51}[a-z0-9])?$`)
	validNamespace   = regexp.MustCompile(`^[a-z0-9]([a-z0-9\-]{0,61}[a-z0-9])?$`)
	validOCIRef      = regexp.MustCompile(`^oci://[a-zA-Z0-9][a-zA-Z0-9._\-/:]+$`)
)

const (
	helmImage = "alpine/helm:3.17.3"
	chartPath = "/src"
	kubeFile  = "/root/.kube/config"
)

// Helm manages Helm chart operations.
type Helm struct {
	// +optional
	Source *dagger.Directory
	// +optional
	Kubeconfig *dagger.Secret
	// +optional
	RegistryUsername string
	// +optional
	RegistryPassword *dagger.Secret
}

func New(
	// Helm chart source directory.
	// +optional
	// +defaultPath=.
	source *dagger.Directory,
	// Kubeconfig for cluster operations.
	// +optional
	kubeconfig *dagger.Secret,
	// OCI registry username.
	// +optional
	registryUsername string,
	// OCI registry password.
	// +optional
	registryPassword *dagger.Secret,
) *Helm {
	return &Helm{
		Source:           source,
		Kubeconfig:       kubeconfig,
		RegistryUsername: registryUsername,
		RegistryPassword: registryPassword,
	}
}

func (h *Helm) base() *dagger.Container {
	return dag.Container().
		From(helmImage).
		WithDirectory(chartPath, h.Source).
		WithWorkdir(chartPath)
}

func (h *Helm) clusterContainer() (*dagger.Container, error) {
	if h.Kubeconfig == nil {
		return nil, fmt.Errorf("kubeconfig is required for cluster operations")
	}
	return h.base().WithMountedSecret(kubeFile, h.Kubeconfig), nil
}

func mountValuesFiles(ctr *dagger.Container, values []*dagger.File) (*dagger.Container, []string) {
	var flags []string
	for i, f := range values {
		path := fmt.Sprintf("/tmp/values-%d.yaml", i)
		ctr = ctr.WithMountedFile(path, f)
		flags = append(flags, "--values", path)
	}
	return ctr, flags
}

func validateReleaseName(name string) error {
	if !validReleaseName.MatchString(name) {
		return fmt.Errorf("invalid release name %q: must be lowercase alphanumeric and dashes, 1-53 chars", name)
	}
	return nil
}

func validateNamespace(ns string) error {
	if ns != "" && !validNamespace.MatchString(ns) {
		return fmt.Errorf("invalid namespace %q: must be lowercase alphanumeric and dashes, 1-63 chars", ns)
	}
	return nil
}

func appendCommonFlags(cmd []string, set []string, namespace string) []string {
	if namespace != "" {
		cmd = append(cmd, "--namespace", namespace)
	}
	for _, s := range set {
		cmd = append(cmd, "--set", s)
	}
	return cmd
}

// Template renders chart templates and returns the YAML output.
func (h *Helm) Template(
	ctx context.Context,
	// Release name for the template.
	// +optional
	// +default="release"
	releaseName string,
	// Custom values files.
	// +optional
	values []*dagger.File,
	// Value overrides (key=value format).
	// +optional
	// +default=[]
	set []string,
	// Kubernetes namespace.
	// +optional
	namespace string,
	// Additional helm template arguments.
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	if err := validateReleaseName(releaseName); err != nil {
		return "", err
	}
	if err := validateNamespace(namespace); err != nil {
		return "", err
	}

	ctr := h.base()
	ctr, valFlags := mountValuesFiles(ctr, values)

	cmd := []string{"helm", "template", releaseName, "."}
	cmd = appendCommonFlags(cmd, set, namespace)
	cmd = append(cmd, valFlags...)
	cmd = append(cmd, args...)

	return ctr.WithExec(cmd).Stdout(ctx)
}

// Lint validates the chart for possible issues.
func (h *Helm) Lint(
	ctx context.Context,
	// Custom values files.
	// +optional
	values []*dagger.File,
	// Value overrides (key=value format).
	// +optional
	// +default=[]
	set []string,
	// Additional helm lint arguments.
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	ctr := h.base()
	ctr, valFlags := mountValuesFiles(ctr, values)

	cmd := []string{"helm", "lint", "."}
	for _, s := range set {
		cmd = append(cmd, "--set", s)
	}
	cmd = append(cmd, valFlags...)
	cmd = append(cmd, args...)

	return ctr.WithExec(cmd).Stdout(ctx)
}

// Package creates a versioned chart archive (.tgz).
func (h *Helm) Package(
	ctx context.Context,
	// Override chart version.
	// +optional
	version string,
	// Override appVersion.
	// +optional
	appVersion string,
	// Additional helm package arguments.
	// +optional
	// +default=[]
	args []string,
) (*dagger.File, error) {
	cmd := []string{"helm", "package", ".", "--destination", "/out"}
	if version != "" {
		cmd = append(cmd, "--version", version)
	}
	if appVersion != "" {
		cmd = append(cmd, "--app-version", appVersion)
	}
	cmd = append(cmd, args...)

	ctr := h.base().WithExec(cmd)

	entries, err := ctr.Directory("/out").Entries(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading package output: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("helm package produced no output")
	}

	return ctr.Directory("/out").File(entries[0]), nil
}

// Push packages and pushes the chart to an OCI registry.
func (h *Helm) Push(
	ctx context.Context,
	// OCI registry URL (e.g., "oci://registry.example.com/charts").
	// +required
	registry string,
	// Override chart version.
	// +optional
	version string,
	// Override appVersion.
	// +optional
	appVersion string,
) (string, error) {
	if !validOCIRef.MatchString(registry) {
		return "", fmt.Errorf("invalid OCI registry reference %q: must start with oci://", registry)
	}
	if h.RegistryPassword == nil {
		return "", fmt.Errorf("registryPassword is required for push operations")
	}

	pkg, err := h.Package(ctx, version, appVersion, nil)
	if err != nil {
		return "", err
	}

	// Extract registry host for login (strip oci:// prefix and path after host)
	registryHost := registry[6:] // strip "oci://"
	if idx := len(registryHost); idx > 0 {
		// Find first slash after host to get just the host part
		for i, c := range registryHost {
			if c == '/' {
				registryHost = registryHost[:i]
				break
			}
		}
	}

	ctr := h.base().
		WithSecretVariable("HELM_REGISTRY_PASSWORD", h.RegistryPassword).
		WithMountedFile("/tmp/chart.tgz", pkg).
		WithExec([]string{
			"sh", "-c",
			fmt.Sprintf("echo \"$HELM_REGISTRY_PASSWORD\" | helm registry login %s --username %s --password-stdin",
				registryHost, h.RegistryUsername),
		}).
		WithExec([]string{"helm", "push", "/tmp/chart.tgz", registry})

	return ctr.Stdout(ctx)
}

// Install installs a Helm chart to a Kubernetes cluster.
func (h *Helm) Install(
	ctx context.Context,
	// Release name.
	// +required
	releaseName string,
	// Custom values files.
	// +optional
	values []*dagger.File,
	// Value overrides (key=value format).
	// +optional
	// +default=[]
	set []string,
	// Kubernetes namespace.
	// +optional
	namespace string,
	// Create namespace if it does not exist.
	// +optional
	// +default=false
	createNamespace bool,
	// Additional helm install arguments.
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	if err := validateReleaseName(releaseName); err != nil {
		return "", err
	}
	if err := validateNamespace(namespace); err != nil {
		return "", err
	}

	ctr, err := h.clusterContainer()
	if err != nil {
		return "", err
	}
	ctr, valFlags := mountValuesFiles(ctr, values)

	cmd := []string{"helm", "install", releaseName, "."}
	cmd = appendCommonFlags(cmd, set, namespace)
	cmd = append(cmd, valFlags...)
	if createNamespace {
		cmd = append(cmd, "--create-namespace")
	}
	cmd = append(cmd, args...)

	return ctr.WithExec(cmd).Stdout(ctx)
}

// Upgrade upgrades a Helm release, optionally installing it if it does not exist.
func (h *Helm) Upgrade(
	ctx context.Context,
	// Release name.
	// +required
	releaseName string,
	// Custom values files.
	// +optional
	values []*dagger.File,
	// Value overrides (key=value format).
	// +optional
	// +default=[]
	set []string,
	// Kubernetes namespace.
	// +optional
	namespace string,
	// Install the release if it does not exist.
	// +optional
	// +default=false
	install bool,
	// Create namespace if it does not exist.
	// +optional
	// +default=false
	createNamespace bool,
	// Additional helm upgrade arguments.
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	if err := validateReleaseName(releaseName); err != nil {
		return "", err
	}
	if err := validateNamespace(namespace); err != nil {
		return "", err
	}

	ctr, err := h.clusterContainer()
	if err != nil {
		return "", err
	}
	ctr, valFlags := mountValuesFiles(ctr, values)

	cmd := []string{"helm", "upgrade", releaseName, "."}
	cmd = appendCommonFlags(cmd, set, namespace)
	cmd = append(cmd, valFlags...)
	if install {
		cmd = append(cmd, "--install")
	}
	if createNamespace {
		cmd = append(cmd, "--create-namespace")
	}
	cmd = append(cmd, args...)

	return ctr.WithExec(cmd).Stdout(ctx)
}

// Rollback rolls back a release to a previous revision.
func (h *Helm) Rollback(
	ctx context.Context,
	// Release name.
	// +required
	releaseName string,
	// Revision number to roll back to (0 for previous).
	// +optional
	// +default=0
	revision int,
	// Kubernetes namespace.
	// +optional
	namespace string,
	// Additional helm rollback arguments.
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	if err := validateReleaseName(releaseName); err != nil {
		return "", err
	}
	if err := validateNamespace(namespace); err != nil {
		return "", err
	}

	ctr, err := h.clusterContainer()
	if err != nil {
		return "", err
	}

	cmd := []string{"helm", "rollback", releaseName}
	if revision > 0 {
		cmd = append(cmd, fmt.Sprintf("%d", revision))
	}
	if namespace != "" {
		cmd = append(cmd, "--namespace", namespace)
	}
	cmd = append(cmd, args...)

	return ctr.WithExec(cmd).Stdout(ctx)
}

// Uninstall removes a Helm release from the cluster.
func (h *Helm) Uninstall(
	ctx context.Context,
	// Release name.
	// +required
	releaseName string,
	// Kubernetes namespace.
	// +optional
	namespace string,
	// Additional helm uninstall arguments.
	// +optional
	// +default=[]
	args []string,
) (string, error) {
	if err := validateReleaseName(releaseName); err != nil {
		return "", err
	}
	if err := validateNamespace(namespace); err != nil {
		return "", err
	}

	ctr, err := h.clusterContainer()
	if err != nil {
		return "", err
	}

	cmd := []string{"helm", "uninstall", releaseName}
	if namespace != "" {
		cmd = append(cmd, "--namespace", namespace)
	}
	cmd = append(cmd, args...)

	return ctr.WithExec(cmd).Stdout(ctx)
}
