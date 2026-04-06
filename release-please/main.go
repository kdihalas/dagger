// Automate GitHub releases with release-please.
package main

import (
	"context"
	"dagger/release-please/internal/dagger"
	"fmt"
	"regexp"
	"strings"
)

const releasePleaseVersion = "16.17.1"

var (
	validRepoURL = regexp.MustCompile(`^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`)

	allowedReleaseTypes = map[string]bool{
		"dart":           true,
		"elixir":         true,
		"go":             true,
		"helm":           true,
		"java":           true,
		"krm-blueprint":  true,
		"maven":          true,
		"node":           true,
		"ocaml":          true,
		"php":            true,
		"python":         true,
		"ruby":           true,
		"rust":           true,
		"salesforce":     true,
		"simple":         true,
		"terraform-module": true,
	}
)

type ReleasePlease struct {
	Token *dagger.Secret
}

func New(
	// GitHub token for authentication.
	// +required
	token *dagger.Secret,
) *ReleasePlease {
	return &ReleasePlease{Token: token}
}

// Container returns a container with release-please installed.
func (r *ReleasePlease) Container() *dagger.Container {
	return dag.Container().
		From("node:current-alpine3.23").
		WithExec([]string{"npm", "i", fmt.Sprintf("release-please@%s", releasePleaseVersion), "-g"})
}

func (r *ReleasePlease) validateInputs(releaseType, repoUrl string) error {
	if !allowedReleaseTypes[releaseType] {
		types := make([]string, 0, len(allowedReleaseTypes))
		for t := range allowedReleaseTypes {
			types = append(types, t)
		}
		return fmt.Errorf("invalid release type %q; allowed: %s", releaseType, strings.Join(types, ", "))
	}
	if !validRepoURL.MatchString(repoUrl) {
		return fmt.Errorf("invalid repo URL %q; expected format: github.com/owner/repo", repoUrl)
	}
	return nil
}

func (r *ReleasePlease) exec(ctx context.Context, args ...string) (string, error) {
	token, err := r.Token.Plaintext(ctx)
	if err != nil {
		return "", fmt.Errorf("reading token: %w", err)
	}
	cmd := append([]string{"release-please"}, args...)
	cmd = append(cmd, "--token", token)
	return r.Container().WithExec(cmd).Stdout(ctx)
}

// ReleasePr creates a release pull request.
func (r *ReleasePlease) ReleasePr(
	ctx context.Context,
	// Release type (e.g., "node", "go", "python").
	// +required
	releaseType string,
	// Repository URL (e.g., "github.com/owner/repo").
	// +required
	repoUrl string,
) (string, error) {
	if err := r.validateInputs(releaseType, repoUrl); err != nil {
		return "", err
	}
	return r.exec(ctx, "release-pr", "--repo-url", repoUrl, "--release-type", releaseType)
}

// GithubRelease creates a GitHub release from a merged release PR.
func (r *ReleasePlease) GithubRelease(
	ctx context.Context,
	// +required
	releaseType string,
	// +required
	repoUrl string,
) (string, error) {
	if err := r.validateInputs(releaseType, repoUrl); err != nil {
		return "", err
	}
	return r.exec(ctx, "github-release", "--repo-url", repoUrl, "--release-type", releaseType)
}

// Run creates a release PR then a GitHub release, returning combined output.
func (r *ReleasePlease) Run(
	ctx context.Context,
	// +required
	releaseType string,
	// +required
	repoUrl string,
) (string, error) {
	prOut, err := r.ReleasePr(ctx, releaseType, repoUrl)
	if err != nil {
		return "", err
	}
	relOut, err := r.GithubRelease(ctx, releaseType, repoUrl)
	if err != nil {
		return prOut, err
	}
	return prOut + "\n" + relOut, nil
}

// Bootstrap initializes release-please in a repository.
func (r *ReleasePlease) Bootstrap(
	ctx context.Context,
	// +required
	repoUrl string,
) (string, error) {
	if !validRepoURL.MatchString(repoUrl) {
		return "", fmt.Errorf("invalid repo URL %q; expected format: github.com/owner/repo", repoUrl)
	}
	return r.exec(ctx, "bootstrap", "--repo-url", repoUrl)
}
