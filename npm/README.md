# NPM Module

Build, test, lint, and containerize Node.js applications with Dagger. Auto-detects npm, pnpm, or yarn.

## Requirements

- Dagger v0.20.3 or later
- Node.js v16+ (version specified in `package.json` engines field)

## Functions

| Function | Description | Key Parameters |
|----------|-------------|-----------------|
| `NodeVersion` | Extract Node.js version from package.json | — |
| `Base` | Container with Node.js and source mounted | — |
| `Install` | Run package manager install (`npm ci`, `pnpm install`, or `yarn install`) | — |
| `Lint` | Run lint script from package.json | `args` ([]string, optional) |
| `Test` | Run test script from package.json | `args` ([]string, optional) |
| `Build` | Run build script and return output directory | `outDir` (string, optional, default: `dist`) |
| `Container` | Build production container (distroless Node) | `outDir`, `entrypoint` (default: `index.js`) |
| `DebugContainer` | Build Alpine Node.js debug container | `outDir`, `entrypoint` (default: `index.js`) |
| `Publish` | Build and push container to registry | `outDir`, `entrypoint`, `imageName` (required), `registry`, `username`, `password` |

## Usage Examples

Extract Node version:

```bash
dagger call -m npm --source . node-version
```

Run tests:

```bash
dagger call -m npm --source . test
```

Run tests with arguments:

```bash
dagger call -m npm --source . test --args "--coverage"
```

Run linter:

```bash
dagger call -m npm --source . lint
```

Build application:

```bash
dagger call -m npm --source . build --out-dir dist
```

Publish container:

```bash
dagger call -m npm --source . publish \
  --out-dir dist \
  --entrypoint index.js \
  --image-name myapp:latest \
  --registry docker.io \
  --username myuser \
  --password env:DOCKER_PASSWORD
```

Publish multiple tags:

```bash
dagger call -m npm --source . publish \
  --out-dir dist \
  --entrypoint server.js \
  --image-name myapp:latest,myapp:v1.0.0 \
  --registry docker.io \
  --username myuser \
  --password env:DOCKER_PASSWORD
```

Debug container (Alpine with shell):

```bash
dagger call -m npm --source . debug-container
```

## GitHub Actions

Use the NPM module in GitHub Actions to test, lint, and containerize your Node.js applications:

```yaml
name: Build Node App

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test-and-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: dagger/dagger-for-github@v8.4.1
        with:
          version: "0.20.3"
      - name: Run tests
        run: dagger -m npm call --source . test
      - name: Run linter
        run: dagger -m npm call --source . lint
      - name: Build application
        run: dagger -m npm call --source . build --out-dir dist
      - name: Build container
        run: dagger -m npm call --source . container \
          --out-dir dist \
          --entrypoint index.js
      - name: Publish container
        run: dagger -m npm call --source . publish \
          --out-dir dist \
          --entrypoint index.js \
          --image-name myapp:latest \
          --registry docker.io \
          --username ${{ secrets.DOCKER_USERNAME }} \
          --password ${{ secrets.DOCKER_PASSWORD }}
```
