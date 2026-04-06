# Helm Module

Manage Helm chart lifecycle: template, lint, package, push, install, upgrade, rollback, and uninstall.

## Requirements

- Dagger v0.20.3 or later
- Kubeconfig secret for cluster operations (install, upgrade, rollback, uninstall)
- OCI registry credentials for push operations

## Functions

| Function | Description | Requires Kubeconfig |
|----------|-------------|---------------------|
| `Template` | Render chart templates, return YAML | no |
| `Lint` | Validate chart structure | no |
| `Package` | Create .tgz chart archive | no |
| `Push` | Push chart to OCI registry | no |
| `Install` | Install a release to a cluster | yes |
| `Upgrade` | Upgrade a release (with --install option) | yes |
| `Rollback` | Roll back to a previous revision | yes |
| `Uninstall` | Remove a release | yes |

## Constructor Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `source` | Directory | no | `.` | Helm chart source directory |
| `kubeconfig` | Secret | no | -- | Kubeconfig for cluster operations |
| `registryUsername` | string | no | -- | OCI registry username |
| `registryPassword` | Secret | no | -- | OCI registry password |

## Common Parameters

Most functions accept these optional parameters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `values` | []File | Custom values files |
| `set` | []string | Value overrides (key=value) |
| `namespace` | string | Kubernetes namespace |
| `args` | []string | Additional helm arguments |

## Usage Examples

### Lint a chart

```bash
dagger call -m helm --source ./my-chart lint
```

### Render templates

```bash
dagger call -m github.com/kdihalas/dagger/helm --source ./my-chart template --release-name my-release
```

### Render with custom values

```bash
dagger call -m github.com/kdihalas/dagger/helm --source ./my-chart template \
  --release-name my-release \
  --values ./values-prod.yaml \
  --set "image.tag=v1.2.3"
```

### Package a chart

```bash
dagger call -m github.com/kdihalas/dagger/helm --source ./my-chart package --version 1.0.0
```

### Push to OCI registry

```bash
dagger call -m github.com/kdihalas/dagger/helm \
  --source ./my-chart \
  --registry-username myuser \
  --registry-password env:REGISTRY_PASSWORD \
  push --registry "oci://registry.example.com/charts"
```

### Install a release

```bash
dagger call -m github.com/kdihalas/dagger/helm \
  --source ./my-chart \
  --kubeconfig env:KUBECONFIG \
  install \
    --release-name my-app \
    --namespace production \
    --create-namespace
```

### Upgrade with install fallback

```bash
dagger call -m github.com/kdihalas/dagger/helm \
  --source ./my-chart \
  --kubeconfig env:KUBECONFIG \
  upgrade \
    --release-name my-app \
    --namespace production \
    --install \
    --set "image.tag=v1.3.0"
```

### Rollback to previous revision

```bash
dagger call -m github.com/kdihalas/dagger/helm \
  --kubeconfig env:KUBECONFIG \
  rollback \
    --release-name my-app \
    --namespace production
```

### Rollback to specific revision

```bash
dagger call -m github.com/kdihalas/dagger/helm \
  --kubeconfig env:KUBECONFIG \
  rollback \
    --release-name my-app \
    --revision 3 \
    --namespace production
```

### Uninstall a release

```bash
dagger call -m github.com/kdihalas/dagger/helm \
  --kubeconfig env:KUBECONFIG \
  uninstall \
    --release-name my-app \
    --namespace production
```

## GitHub Actions

### Lint and Package

```yaml
name: Helm CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  helm:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Lint chart
        run: dagger -m github.com/kdihalas/dagger/helm call --source ./my-chart lint
      - name: Package chart
        run: dagger -m github.com/kdihalas/dagger/helm call --source ./my-chart package --version 1.0.0
```

### Push to OCI Registry

```yaml
name: Helm Push

on:
  push:
    tags: ["v*"]

jobs:
  push:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Push chart
        run: dagger -m github.com/kdihalas/dagger/helm \
          --source ./my-chart \
          --registry-username ${{ secrets.REGISTRY_USERNAME }} \
          --registry-password ${{ secrets.REGISTRY_PASSWORD }} \
          call push --registry "oci://registry.example.com/charts"
```

### Deploy to Cluster

```yaml
name: Helm Deploy

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
      - name: Deploy
        run: dagger -m github.com/kdihalas/dagger/helm \
          --source ./my-chart \
          --kubeconfig ${{ secrets.KUBECONFIG }} \
          call upgrade \
            --release-name my-app \
            --namespace production \
            --install \
            --create-namespace \
            --set "image.tag=${{ github.sha }}"
```

### Push to ECR with configure-aws-credentials

Use the `configure-aws-credentials` and `amazon-ecr-login` modules to authenticate, then push the chart to ECR:

```yaml
name: Helm Push to ECR (OIDC)

on:
  push:
    tags: ["v*"]

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
      - name: Push chart to ECR
        run: dagger -m github.com/kdihalas/dagger/helm \
          --source ./my-chart \
          --registry-username AWS \
          --registry-password env:ECR_PASSWORD \
          call push --registry "oci://123456789012.dkr.ecr.us-east-1.amazonaws.com/charts"
```
