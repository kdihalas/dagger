// Retrieve secrets from HashiCorp Vault KV v2 engine using GitHub OIDC authentication.
package main

import (
	"context"
	"dagger/vault-action/internal/dagger"
	"fmt"
	"regexp"
	"strings"
)

var (
	validURL       = regexp.MustCompile(`^https?://.+`)
	validMountPath = regexp.MustCompile(`^[a-zA-Z0-9_\-/]+$`)
	validPath      = regexp.MustCompile(`^[a-zA-Z0-9_\-/]+$`)
	validKey       = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+$`)
	validRole      = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
)

const vaultImage = "hashicorp/vault:1.19"

// VaultAction retrieves secrets from HashiCorp Vault using GitHub OIDC authentication.
type VaultAction struct {
	// Vault server URL (e.g., https://vault.example.com:8200).
	URL string
	// GitHub OIDC token (JWT) for Vault authentication.
	GithubToken *dagger.Secret
	// Vault JWT auth role name.
	Role string
	// Vault JWT auth mount path.
	// +optional
	// +default="jwt"
	AuthMount string
	// Vault namespace (for Vault Enterprise).
	// +optional
	Namespace string
}

func New(
	// Vault server URL (e.g., https://vault.example.com:8200).
	url string,
	// GitHub OIDC token (JWT) for Vault authentication.
	githubToken *dagger.Secret,
	// Vault JWT auth role name.
	role string,
	// Vault JWT auth mount path.
	// +optional
	// +default="jwt"
	authMount string,
	// Vault namespace (for Vault Enterprise).
	// +optional
	namespace string,
) *VaultAction {
	if authMount == "" {
		authMount = "jwt"
	}
	return &VaultAction{
		URL:         url,
		GithubToken: githubToken,
		Role:        role,
		AuthMount:   authMount,
		Namespace:   namespace,
	}
}

func (v *VaultAction) base() *dagger.Container {
	ctr := dag.Container().
		From(vaultImage).
		WithEnvVariable("VAULT_ADDR", v.URL).
		WithSecretVariable("GITHUB_OIDC_TOKEN", v.GithubToken)
	if v.Namespace != "" {
		ctr = ctr.WithEnvVariable("VAULT_NAMESPACE", v.Namespace)
	}
	// Authenticate with Vault using the GitHub OIDC JWT and export the resulting token.
	ctr = ctr.WithExec([]string{
		"sh", "-c",
		fmt.Sprintf(
			`export VAULT_TOKEN=$(vault write -field=token auth/%s/login role=%s jwt="$GITHUB_OIDC_TOKEN") && echo "$VAULT_TOKEN" > /tmp/.vault-token`,
			v.AuthMount, v.Role,
		),
	}).WithEnvVariable("VAULT_TOKEN", "").
		WithExec([]string{"sh", "-c", `export VAULT_TOKEN=$(cat /tmp/.vault-token) && echo "$VAULT_TOKEN" > /dev/null`})

	return ctr
}

// wrapCmd wraps a vault command so it runs with the OIDC-derived token.
func (v *VaultAction) wrapCmd(args string) []string {
	return []string{
		"sh", "-c",
		fmt.Sprintf(`export VAULT_TOKEN=$(cat /tmp/.vault-token) && %s`, args),
	}
}

// GetSecret reads a single field from a KV v2 secret and returns it as a Dagger secret.
func (v *VaultAction) GetSecret(
	ctx context.Context,
	// KV v2 mount path (e.g., "secret").
	mount string,
	// Secret path within the mount (e.g., "data/myapp/config").
	path string,
	// Field name to retrieve from the secret.
	key string,
	// Name for the returned Dagger secret.
	// +optional
	// +default="vault-secret"
	name string,
) (*dagger.Secret, error) {
	if err := v.validate(); err != nil {
		return nil, err
	}
	if !validMountPath.MatchString(mount) {
		return nil, fmt.Errorf("invalid mount path: %q", mount)
	}
	if !validPath.MatchString(path) {
		return nil, fmt.Errorf("invalid secret path: %q", path)
	}
	if !validKey.MatchString(key) {
		return nil, fmt.Errorf("invalid key: %q", key)
	}

	out, err := v.base().
		WithExec(v.wrapCmd(fmt.Sprintf(
			`vault kv get -mount=%s -field=%s %s`,
			mount, key, path,
		))).
		Stdout(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading secret %s/%s field %s: %w", mount, path, key, err)
	}

	return dag.SetSecret(name, strings.TrimSpace(out)), nil
}

// GetSecretJSON reads all fields from a KV v2 secret and returns the JSON as a string.
func (v *VaultAction) GetSecretJSON(
	ctx context.Context,
	// KV v2 mount path (e.g., "secret").
	mount string,
	// Secret path within the mount (e.g., "data/myapp/config").
	path string,
) (string, error) {
	if err := v.validate(); err != nil {
		return "", err
	}
	if !validMountPath.MatchString(mount) {
		return "", fmt.Errorf("invalid mount path: %q", mount)
	}
	if !validPath.MatchString(path) {
		return "", fmt.Errorf("invalid secret path: %q", path)
	}

	out, err := v.base().
		WithExec(v.wrapCmd(fmt.Sprintf(
			`vault kv get -mount=%s -format=json %s`,
			mount, path,
		))).
		Stdout(ctx)
	if err != nil {
		return "", fmt.Errorf("reading secret %s/%s: %w", mount, path, err)
	}

	return strings.TrimSpace(out), nil
}

func (v *VaultAction) validate() error {
	if !validURL.MatchString(v.URL) {
		return fmt.Errorf("invalid vault URL: %q", v.URL)
	}
	if !validRole.MatchString(v.Role) {
		return fmt.Errorf("invalid role: %q", v.Role)
	}
	if !validMountPath.MatchString(v.AuthMount) {
		return fmt.Errorf("invalid auth mount: %q", v.AuthMount)
	}
	return nil
}
