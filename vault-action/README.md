# Vault Action

Retrieve secrets from HashiCorp Vault KV v2 engine.

Supports two authentication methods:
- **Token**: provide `--token` directly
- **GitHub OIDC**: provide `--github-token` and `--role`

## Requirements

- Dagger v0.20.3 or later
- A running HashiCorp Vault instance
- Either a Vault token or a GitHub OIDC token with a configured JWT auth backend

## Constructor Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `url` | `string` | yes | — | Vault server URL (e.g., `https://vault.example.com:8200`) |
| `token` | `Secret` | no* | — | Vault authentication token |
| `github-token` | `Secret` | no* | — | GitHub OIDC token (JWT) |
| `role` | `string` | no* | — | Vault JWT auth role name (required with `github-token`) |
| `auth-mount` | `string` | no | `jwt` | Vault JWT auth mount path |
| `namespace` | `string` | no | `""` | Vault namespace (for Vault Enterprise) |

\* Provide either `--token` or `--github-token` + `--role`.

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

### Token Auth

```bash
dagger call -m github.com/kdihalas/dagger/vault-action \
  --url https://vault.example.com:8200 \
  --token env:VAULT_TOKEN \
  get-secret --mount secret --path myapp/config --key api-key
```

### GitHub OIDC Auth

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
  --token env:VAULT_TOKEN \
  get-secret-json --mount secret --path myapp/config
```

### Vault Enterprise namespaces

```bash
dagger call -m github.com/kdihalas/dagger/vault-action \
  --url https://vault.example.com:8200 \
  --token env:VAULT_TOKEN \
  --namespace admin/team-a \
  get-secret --mount secret --path myapp/config --key db-password
```

### Go SDK

```go
// Token auth
secret := dag.VaultAction(
    "https://vault.example.com:8200",
    dagger.VaultActionOpts{
        Token: dag.SetSecret("vault-token", os.Getenv("VAULT_TOKEN")),
    },
).GetSecret("secret", "myapp/config", "db-password")

// OIDC auth
secret := dag.VaultAction(
    "https://vault.example.com:8200",
    dagger.VaultActionOpts{
        GithubToken: dag.SetSecret("oidc-token", os.Getenv("GITHUB_OIDC_TOKEN")),
        Role:        "my-github-role",
    },
).GetSecret("secret", "myapp/config", "db-password")
```

## GitHub Actions

### With Token

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Get secret from Vault
        run: |
          dagger call -m github.com/kdihalas/dagger/vault-action \
            --url ${{ secrets.VAULT_URL }} \
            --token env:VAULT_TOKEN \
            get-secret --mount secret --path myapp/config --key api-key
        env:
          VAULT_TOKEN: ${{ secrets.VAULT_TOKEN }}
```

### With GitHub OIDC

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

## Vault OIDC Configuration

To use GitHub OIDC, configure Vault's JWT auth method:

```bash
vault auth enable jwt

vault write auth/jwt/config \
  oidc_discovery_url="https://token.actions.githubusercontent.com" \
  bound_issuer="https://token.actions.githubusercontent.com"

vault write auth/jwt/role/my-github-role \
  bound_audiences="https://vault.example.com:8200" \
  bound_claims_type="glob" \
  bound_claims='{"sub": "repo:myorg/myrepo:*"}' \
  user_claim="actor" \
  role_type="jwt" \
  token_policies="my-policy" \
  token_ttl="10m"
```
