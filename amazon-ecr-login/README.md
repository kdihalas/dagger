# Amazon ECR Login Module

Log in to Amazon ECR private registries and ECR Public (public.ecr.aws) from Dagger pipelines.

## Requirements

- Dagger v0.20.3 or later
- AWS credentials with ECR permissions (`ecr:GetAuthorizationToken` for private, `ecr-public:GetAuthorizationToken` + `sts:GetServiceBearerToken` for public)

## Functions

| Function | Description | Key Parameters |
|----------|-------------|----------------|
| `Login` | Get credentials for private ECR registries | `registries` (string, optional, comma-separated account IDs) |
| `LoginPublic` | Get credentials for ECR Public | -- |
| `WithRegistryAuth` | Apply private ECR auth to a container | `ctr` (Container, required), `registries` (optional) |
| `WithPublicRegistryAuth` | Apply ECR Public auth to a container | `ctr` (Container, required) |

## Constructor Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `accessKeyId` | Secret | no | -- | AWS access key ID |
| `secretAccessKey` | Secret | no | -- | AWS secret access key |
| `sessionToken` | Secret | no | -- | AWS session token |
| `region` | string | no | `us-east-1` | AWS region |

## Return Types

`Login` and `LoginPublic` return `RegistryCredentials`:

| Field | Type | Description |
|-------|------|-------------|
| `registry` | string | Registry URL (e.g., `123456789012.dkr.ecr.us-east-1.amazonaws.com`) |
| `username` | string | Always `AWS` for ECR |
| `password` | Secret | ECR authorization token |

## Usage Examples

### Private ECR (default account)

```bash
dagger call -m amazon-ecr-login \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  --region us-east-1 \
  with-registry-auth --ctr alpine:latest
```

### Private ECR (multiple accounts)

```bash
dagger call -m amazon-ecr-login \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  --region us-east-1 \
  with-registry-auth \
    --ctr alpine:latest \
    --registries "123456789012,998877665544"
```

### ECR Public

```bash
dagger call -m amazon-ecr-login \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  with-public-registry-auth --ctr alpine:latest
```

### Get raw credentials

```bash
dagger call -m amazon-ecr-login \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  --region us-east-1 \
  login
```

## GitHub Actions

### Private ECR

```yaml
name: Push to ECR

on:
  push:
    branches: [main]

jobs:
  push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Login and push to ECR
        run: dagger -m amazon-ecr-login \
          --access-key-id ${{ secrets.AWS_ACCESS_KEY_ID }} \
          --secret-access-key ${{ secrets.AWS_SECRET_ACCESS_KEY }} \
          --region us-east-1 \
          call with-registry-auth \
            --ctr alpine:latest
```

### ECR Public

```yaml
name: Push to ECR Public

on:
  push:
    branches: [main]

jobs:
  push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Login to ECR Public
        run: dagger -m amazon-ecr-login \
          --access-key-id ${{ secrets.AWS_ACCESS_KEY_ID }} \
          --secret-access-key ${{ secrets.AWS_SECRET_ACCESS_KEY }} \
          call with-public-registry-auth \
            --ctr alpine:latest
```

### With aws-config Module (Role Assumption)

Use the `aws-config` Dagger module to assume an IAM role, then pass the temporary credentials to ECR login:

```bash
# Step 1: Assume role with aws-config
dagger call -m configure-aws-credentials \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  --region us-east-1 \
  assume-role \
    --role-arn "arn:aws:iam::123456789012:role/ECRPushRole" \
    --session-name "ecr-push"

# Step 2: Use the assumed credentials with amazon-ecr-login
dagger call -m amazon-ecr-login \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  --session-token env:AWS_SESSION_TOKEN \
  --region us-east-1 \
  with-registry-auth --ctr alpine:latest
```

### With aws-config Module (GitHub OIDC)

Use the `aws-config` module's OIDC web identity support to assume a role without static credentials, then log in to ECR:

```bash
dagger call -m configure-aws-credentials \
  --region us-east-1 \
  assume-role-with-web-identity \
    --role-arn "arn:aws:iam::123456789012:role/GHActionsECRRole" \
    --web-identity-token env:ACTIONS_ID_TOKEN

# Then use the exported credentials with amazon-ecr-login
dagger call -m amazon-ecr-login \
  --access-key-id env:AWS_ACCESS_KEY_ID \
  --secret-access-key env:AWS_SECRET_ACCESS_KEY \
  --session-token env:AWS_SESSION_TOKEN \
  --region us-east-1 \
  with-registry-auth --ctr alpine:latest
```

### GitHub Actions with aws-config + ECR Login

Full workflow using both Dagger modules together with GitHub OIDC -- no static AWS secrets needed:

```yaml
name: Push to ECR (OIDC + Dagger Modules)

on:
  push:
    branches: [main]

permissions:
  id-token: write
  contents: read

jobs:
  push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Get OIDC token
        id: oidc
        run: |
          TOKEN=$(curl -s -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
            "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=sts.amazonaws.com" | jq -r '.value')
          echo "token=$TOKEN" >> "$GITHUB_OUTPUT"
      - name: Assume role via aws-config
        run: dagger -m configure-aws-credentials \
          --region us-east-1 \
          call assume-role-with-web-identity \
            --role-arn "arn:aws:iam::123456789012:role/GHActionsECRRole" \
            --web-identity-token "${{ steps.oidc.outputs.token }}"
      - name: Login to ECR and push
        run: dagger -m amazon-ecr-login \
          --access-key-id env:AWS_ACCESS_KEY_ID \
          --secret-access-key env:AWS_SECRET_ACCESS_KEY \
          --session-token env:AWS_SESSION_TOKEN \
          --region us-east-1 \
          call with-registry-auth \
            --ctr alpine:latest
```
