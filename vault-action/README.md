# Vault Action

Retrieve secrets from HashiCorp Vault KV v2 engine using GitHub OIDC (JWT) authentication.

## Requirements

- Dagger v0.20.3 or later
- A running HashiCorp Vault instance with JWT auth method configured
- A GitHub Actions workflow with `id-token: write` permission

## Constructor Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | `string` | yes | — | Vault server URL (e.g., `https://vault.example.com:8200`) |
| `github-token` | `Secret` | yes | — | GitHub OIDC token (JWT) |
| `role` | `string` | yes | — | Vault JWT auth role name |
| `auth-mount` | `string` | no | `jwt` | Vault JWT auth mount path |
| `namespace` | `string` | no | `""` | Vault namespace (for Vault Enterprise) |

## Functions

| Function | Description |
|----------|-------------|
| `get-secret` | Read a single field from a KV v2 secret, returned as a Dagger secret |
| `get-secret-json` | Read all fields from a KV v2 secret, returned as JSON |

### get-secret

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `mount` | `string` | yes | — | KV v2 mount path (e.g., `secret`) |
| `path` | `string` | yes | — | Secret path within the mount |
| `key` | `string` | yes | — | Field name to retrieve |
| `name` | `string` | no | `vault-secret` | Name for the returned Dagger secret |

### get-secret-json

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `mount` | `string` | yes | — | KV v2 mount path (e.g., `secret`) |
| `path` | `string` | yes | — | Secret path within the mount |

## Usage

### Read a single secret field

```bash
dagger call -m github.com/kdihalas/dagger/vault-action \
  --url https://vault.example.com:8200 \
  --github-token env:GITHUB_OIDC_TOKEN \
  --role my-github-role \
  get-secret --mount secret --path myapp/config --key api-key
```

### Read all fields as JSON

```bash
dagger call -m github.com/kdihalas/dagger/vault-action \
  --url https://vault.example.com:8200 \
  --github-token env:GITHUB_OIDC_TOKEN \
  --role my-github-role \
  get-secret-json --mount secret --path myapp/config
```

### Use with Vault Enterprise namespaces

```bash
dagger call -m github.com/kdihalas/dagger/vault-action \
  --url https://vault.example.com:8200 \
  --github-token env:GITHUB_OIDC_TOKEN \
  --role my-github-role \
  --namespace admin/team-a \
  get-secret --mount secret --path myapp/config --key db-password
```

### Use in a Dagger pipeline (Go SDK)

```go
secret := dag.VaultAction(
    "https://vault.example.com:8200",
    dag.SetSecret("oidc-token", os.Getenv("GITHUB_OIDC_TOKEN")),
    "my-github-role",
).GetSecret("secret", "myapp/config", "db-password")

ctr := dag.Container().
    From("alpine:latest").
    WithSecretVariable("DB_PASSWORD", secret)
```

## GitHub Actions

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write
    steps:
      - uses: actions/checkout@v6
      - uses: actions/github-script@v7
        id: oidc
        with:
          script: return await core.getIDToken("https://vault.example.com:8200")
          result-encoding: string
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Get secret from Vault
        run: |
          dagger call -m github.com/kdihalas/dagger/vault-action \
            --url https://vault.example.com:8200 \
            --github-token env:GITHUB_OIDC_TOKEN \
            --role my-github-role \
            get-secret --mount secret --path myapp/config --key api-key
        env:
          GITHUB_OIDC_TOKEN: ${{ steps.oidc.outputs.result }}
```

## Vault Configuration

To use this module, configure Vault's JWT auth method to trust GitHub's OIDC provider:

```bash
# Enable JWT auth method
vault auth enable jwt

# Configure with GitHub's OIDC discovery
vault write auth/jwt/config \
  oidc_discovery_url="https://token.actions.githubusercontent.com" \
  bound_issuer="https://token.actions.githubusercontent.com"

# Create a role bound to your repository
vault write auth/jwt/role/my-github-role \
  bound_audiences="https://vault.example.com:8200" \
  bound_claims_type="glob" \
  bound_claims='{"sub": "repo:myorg/myrepo:*"}' \
  user_claim="actor" \
  role_type="jwt" \
  token_policies="my-policy" \
  token_ttl="10m"
```
