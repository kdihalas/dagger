// A generated module for ReleasePlease functions
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
	"dagger/release-please/internal/dagger"
	"strings"
)

type ReleasePlease struct {
	// Token is a secret containing the GitHub token to use for authentication with the GitHub API.
	Token *dagger.Secret
}

func New(
	// The GitHub token to use for authentication with the GitHub API.
	// +required
	token *dagger.Secret,
) *ReleasePlease {
	return &ReleasePlease{Token: token}
}

// Container returns a Dagger container configured with the release-please tool installed.
// This container can be used to run release-please commands in a consistent environment.
func (r *ReleasePlease) Container(ctx context.Context) *dagger.Container {
	return dag.Container().From("node:current-alpine3.23").WithExec([]string{
		"npm", "i", "release-please", "-g",
	})
}

// ReleasePr creates a release pull request using the release-please tool and returns the URL of the created pull request.
func (r *ReleasePlease) ReleasePr(
	ctx context.Context,
	// args are the arguments to pass to the release-please command. For example, you might include "--repo-url=github.com/owner/repo" and "--package-name=my-package".
	// release-type is the type of release to create, such as "node" or "python". This will determine the format of the release notes and the versioning scheme used by release-please.
	// +required
	releaseType string,
	// repo-url is the URL of the GitHub repository to create the release in, such as "github.com/owner/repo".
	// +required
	repoUrl string,
) (string, error) {
	return r.Container(ctx).WithSecretVariable("GITHUB_TOKEN", r.Token).WithExec([]string{
		"sh", "-c", `release-please release-pr --token $GITHUB_TOKEN --release-type ` + releaseType + ` --repo-url ` + repoUrl,
	}).Stdout(ctx)
}

// GithubRelease creates a GitHub release using the release-please tool and returns the URL of the created release.
func (r *ReleasePlease) GithubRelease(
	ctx context.Context,
	// args are the arguments to pass to the release-please command. For example, you might include "--repo-url=github.com/owner/repo" and "--package-name=my-package".
	// release-type is the type of release to create, such as "node" or "python". This will determine the format of the release notes and the versioning scheme used by release-please.
	// +required
	releaseType string,
	// repo-url is the URL of the GitHub repository to create the release in, such as "github.com/owner/repo".
	// +required
	repoUrl string,
) (string, error) {
	return r.Container(ctx).WithSecretVariable("GITHUB_TOKEN", r.Token).WithExec([]string{
		"sh", "-c", `release-please github-release --token $GITHUB_TOKEN --release-type ` + releaseType + ` --repo-url ` + repoUrl,
	}).Stdout(ctx)
}

// Run executes the release-please commands to create a release pull request and a GitHub release, and returns the combined output of both operations.
func (r *ReleasePlease) Run(
	ctx context.Context,
	// release-type is the type of release to create, such as "node" or "python". This will determine the format of the release notes and the versioning scheme used by release-please.
	// +required
	releaseType string,
	// repo-url is the URL of the GitHub repository to create the release in, such as "github.com/owner/repo".
	// +required
	repoUrl string,
) (string, error) {
	ghOut, err := r.GithubRelease(ctx, releaseType, repoUrl)
	if err != nil {
		return ghOut, err
	}
	rpOut, err := r.ReleasePr(ctx, releaseType, repoUrl)
	if err != nil {
		return rpOut, err
	}

	return strings.Join([]string{ghOut, rpOut}, "\n"), nil
}

// Bootstrap runs the `release-please bootstrap` command to set up release-please in the repository.
func (r *ReleasePlease) Bootstrap(
	ctx context.Context,
	// repo-url is the URL of the GitHub repository to bootstrap release-please in, such as "github.com/owner/repo".
	// +required
	repoUrl string,
) (string, error) {
	return r.Container(ctx).WithSecretVariable("GITHUB_TOKEN", r.Token).WithExec([]string{
		"sh", "-c", `release-please bootstrap --token $GITHUB_TOKEN --repo-url ` + repoUrl,
	}).Stdout(ctx)
}
