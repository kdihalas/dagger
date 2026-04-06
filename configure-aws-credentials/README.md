# Configure AWS Credentials Module

Configure AWS credentials for Dagger pipelines. Supports static credentials, IAM role assumption, and OIDC web identity tokens.

## Requirements

- Dagger v0.20.3 or later
- AWS credentials or an OIDC identity provider configured

## Functions

| Function | Description | Key Parameters |
|----------|-------------|----------------|
| `WithCredentials` | Apply AWS credentials to any container as env vars | `ctr` (Container, required) |
| `AssumeRole` | Assume an IAM role via STS and return new credentials | `roleArn` (required), `sessionName`, `duration`, `externalId`, `policy` |
| `AssumeRoleWithWebIdentity` | Assume a role via OIDC web identity token | `roleArn` (required), `webIdentityToken` (Secret, required), `sessionName`, `duration`, `policy` |

## Constructor Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `accessKeyId` | Secret | no | -- | AWS access key ID |
| `secretAccessKey` | Secret | no | -- | AWS secret access key |
| `sessionToken` | Secret | no | -- | AWS session token |
| `region` | string | no | `us-east-1` | AWS region |

## Usage Examples

### Static Credentials

Apply credentials to a container:

```bash
dagger call -m configure-aws-credentials \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  --region us-west-2 \
  with-credentials --ctr alpine:latest
```

### Assume an IAM Role

```bash
dagger call -m configure-aws-credentials \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  assume-role \
    --role-arn "arn:aws:iam::123456789012:role/MyRole" \
    --session-name "my-pipeline" \
    --duration 3600 \
  with-credentials --ctr alpine:latest
```

### Assume Role with External ID

```bash
dagger call -m configure-aws-credentials \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  assume-role \
    --role-arn "arn:aws:iam::123456789012:role/CrossAccountRole" \
    --external-id "my-external-id" \
  with-credentials --ctr alpine:latest
```

### OIDC Web Identity (GitHub Actions)

```bash
dagger call -m configure-aws-credentials \
  --region us-east-1 \
  assume-role-with-web-identity \
    --role-arn "arn:aws:iam::123456789012:role/GHActionsRole" \
    --web-identity-token env:ACTIONS_ID_TOKEN \
  with-credentials --ctr alpine:latest
```

### Scope Down Permissions with Inline Policy

```bash
dagger call -m configure-aws-credentials \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  assume-role \
    --role-arn "arn:aws:iam::123456789012:role/MyRole" \
    --policy '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}' \
  with-credentials --ctr alpine:latest
```

## Environment Variables Set

`WithCredentials` sets the following on the target container:

| Variable | Source | Type |
|----------|--------|------|
| `AWS_ACCESS_KEY_ID` | Secret | `WithSecretVariable` |
| `AWS_SECRET_ACCESS_KEY` | Secret | `WithSecretVariable` |
| `AWS_SESSION_TOKEN` | Secret (if present) | `WithSecretVariable` |
| `AWS_REGION` | Plain | `WithEnvVariable` |
| `AWS_DEFAULT_REGION` | Plain | `WithEnvVariable` |

All credential values are handled as Dagger secrets and are never exposed in logs or build cache.

## GitHub Actions

Use the configure-aws-credentials module in GitHub Actions with static credentials or OIDC:

### Static Credentials

```yaml
name: AWS Pipeline (Static)

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Configure AWS and deploy
        run: dagger -m configure-aws-credentials \
          --access-key-id ${{ secrets.AWS_ACCESS_KEY_ID }} \
          --secret-access-key ${{ secrets.AWS_SECRET_ACCESS_KEY }} \
          --region us-east-1 \
          call with-credentials \
          --ctr alpine:latest
```

### OIDC Web Identity

```yaml
name: AWS Pipeline (OIDC)

on:
  push:
    branches: [main]

permissions:
  id-token: write

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Get OIDC token
        run: echo "ACTIONS_ID_TOKEN=$(curl $ACTIONS_ID_TOKEN_REQUEST_URL)" >> $GITHUB_ENV
      - name: Assume role and deploy
        run: dagger -m configure-aws-credentials \
          --region us-east-1 \
          call assume-role-with-web-identity \
          --role-arn "arn:aws:iam::123456789012:role/GHActionsRole" \
          --web-identity-token ${{ env.ACTIONS_ID_TOKEN }} \
          | dagger call with-credentials --ctr alpine:latest
```
