// Retrieve secrets from HashiCorp Vault KV v2 engine.
//
// Supports two authentication methods:
//   - Token: provide --token directly
//   - GitHub OIDC: provide --github-token and --role
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

// VaultAction retrieves secrets from HashiCorp Vault.
type VaultAction struct {
	// Vault server URL (e.g., https://vault.example.com:8200).
	URL string
	// Vault authentication token (for token-based auth).
	// +optional
	Token *dagger.Secret
	// GitHub OIDC token (JWT) for Vault JWT auth.
	// +optional
	GithubToken *dagger.Secret
	// Vault JWT auth role name (required with github-token).
	// +optional
	Role string
	// Vault JWT auth mount path (used with github-token).
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
	// Vault authentication token (for token-based auth).
	// +optional
	token *dagger.Secret,
	// GitHub OIDC token (JWT) for Vault JWT auth.
	// +optional
	githubToken *dagger.Secret,
	// Vault JWT auth role name (required with github-token).
	// +optional
	role string,
	// Vault JWT auth mount path (used with github-token).
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
		Token:       token,
		GithubToken: githubToken,
		Role:        role,
		AuthMount:   authMount,
		Namespace:   namespace,
	}
}

func (v *VaultAction) base() (*dagger.Container, error) {
	if err := v.validate(); err != nil {
		return nil, err
	}

	ctr := dag.Container().
		From(vaultImage).
		WithEnvVariable("VAULT_ADDR", v.URL)

	if v.Namespace != "" {
		ctr = ctr.WithEnvVariable("VAULT_NAMESPACE", v.Namespace)
	}

	if v.Token != nil {
		// Token-based auth: set VAULT_TOKEN directly.
		return ctr.WithSecretVariable("VAULT_TOKEN", v.Token), nil
	}

	// GitHub OIDC auth: login via JWT and persist the resulting token.
	ctr = ctr.
		WithSecretVariable("GITHUB_OIDC_TOKEN", v.GithubToken).
		WithExec([]string{
			"sh", "-c",
			fmt.Sprintf(
				`vault write -field=token auth/%s/login role=%s jwt="$GITHUB_OIDC_TOKEN" > /tmp/.vault-token`,
				v.AuthMount, v.Role,
			),
		})

	return ctr, nil
}

// execCmd returns the exec args for a vault command, handling the token source
// difference between token auth (VAULT_TOKEN env) and OIDC auth (file).
func (v *VaultAction) execCmd(args string) []string {
	if v.Token != nil {
		return []string{"sh", "-c", args}
	}
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
	if !validMountPath.MatchString(mount) {
		return nil, fmt.Errorf("invalid mount path: %q", mount)
	}
	if !validPath.MatchString(path) {
		return nil, fmt.Errorf("invalid secret path: %q", path)
	}
	if !validKey.MatchString(key) {
		return nil, fmt.Errorf("invalid key: %q", key)
	}

	ctr, err := v.base()
	if err != nil {
		return nil, err
	}

	out, err := ctr.
		WithExec(v.execCmd(fmt.Sprintf(
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
	if !validMountPath.MatchString(mount) {
		return "", fmt.Errorf("invalid mount path: %q", mount)
	}
	if !validPath.MatchString(path) {
		return "", fmt.Errorf("invalid secret path: %q", path)
	}

	ctr, err := v.base()
	if err != nil {
		return "", err
	}

	out, err := ctr.
		WithExec(v.execCmd(fmt.Sprintf(
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

	hasToken := v.Token != nil
	hasOIDC := v.GithubToken != nil

	if !hasToken && !hasOIDC {
		return fmt.Errorf("provide either --token or --github-token with --role")
	}
	if hasToken && hasOIDC {
		return fmt.Errorf("provide either --token or --github-token, not both")
	}
	if hasOIDC {
		if v.Role == "" {
			return fmt.Errorf("--role is required when using --github-token")
		}
		if !validRole.MatchString(v.Role) {
			return fmt.Errorf("invalid role: %q", v.Role)
		}
		if !validMountPath.MatchString(v.AuthMount) {
			return fmt.Errorf("invalid auth mount: %q", v.AuthMount)
		}
	}

	return nil
}
