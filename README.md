# Dagger Modules Monorepo

Reusable Dagger modules for building, testing, and releasing software.

## Requirements

- Dagger v0.20.3 or later

## Modules

| Module | Description |
|--------|-------------|
| [amazon-ecr-login](./amazon-ecr-login) | Log in to Amazon ECR and ECR Public registries |
| [aws-config](./aws-config) | Configure AWS credentials (static, role assumption, OIDC) |
| [go](./go) | Build, test, lint, and containerize Go applications |
| [npm](./npm) | Build, test, lint, and containerize Node.js applications |
| [python](./python) | Build, test, lint, and containerize Python applications |
| [release-please](./release-please) | Automate GitHub releases with release-please |

## Quick Start

### Amazon ECR Login Module

```bash
dagger call -m amazon-ecr-login \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  --region us-east-1 \
  with-registry-auth --ctr alpine:latest
```

### AWS Config Module

```bash
dagger call -m aws-config \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  with-credentials --ctr alpine:latest
```

### Go Module

```bash
dagger call -m go --source . build
```

### NPM Module

```bash
dagger call -m npm --source . build --out-dir dist
```

### Python Module

```bash
dagger call -m python --source . test
```

### Release-Please Module

```bash
dagger call -m release-please --token env:GITHUB_TOKEN release-pr \
  --release-type go \
  --repo-url github.com/owner/repo
```

See each module's README for full documentation and examples.
