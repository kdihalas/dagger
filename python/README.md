# Python Module

Build, test, lint, and containerize Python applications with Dagger. Auto-detects pip, uv, or poetry.

## Requirements

- Dagger v0.20.3 or later
- Python 3.x (version from `.python-version` file, defaults to `3`)

## Functions

| Function | Description | Key Parameters |
|----------|-------------|----------------|
| `PythonVersion` | Extract Python version from `.python-version` | -- |
| `Base` | Container with Python and source mounted | -- |
| `Install` | Run package manager install (pip, uv, or poetry) | -- |
| `Lint` | Run a Python linter | `linter` (string, optional, default: `ruff`), `args` ([]string, optional) |
| `Test` | Run tests | `runner` (string, optional, default: `pytest`), `args` ([]string, optional) |
| `Build` | Build wheel/sdist and return output directory | `outDir` (string, optional, default: `dist`) |
| `Container` | Build production container (distroless) | `entrypoint` (string, optional, default: `python`), `entrypointArgs` ([]string, optional) |
| `DebugContainer` | Build Alpine-based debug container | `entrypoint`, `entrypointArgs` |
| `Publish` | Build and push container to registry | `entrypoint`, `entrypointArgs`, `imageName` ([]string, required), `registry`, `username`, `password` |

## Package Manager Detection

The module auto-detects the package manager:

| Lockfile/Config | Package Manager | Install Command |
|-----------------|-----------------|-----------------|
| `uv.lock` | uv | `uv sync --frozen` |
| `pyproject.toml` with `[tool.poetry]` | poetry | `poetry install` |
| `requirements.txt` | pip | `pip install -r requirements.txt` |
| Otherwise | pip | `pip install .` |

## Usage Examples

Extract Python version:

```bash
dagger call -m github.com/kdihalas/dagger/python --source . python-version
```

Run tests:

```bash
dagger call -m github.com/kdihalas/dagger/python --source . test
```

Run tests with arguments:

```bash
dagger call -m github.com/kdihalas/dagger/python --source . test --args "-v" --args "--tb=short"
```

Use a different test runner:

```bash
dagger call -m github.com/kdihalas/dagger/python --source . test --runner unittest
```

Run linter:

```bash
dagger call -m github.com/kdihalas/dagger/python --source . lint
```

Use a different linter:

```bash
dagger call -m github.com/kdihalas/dagger/python --source . lint --linter flake8
```

Build distribution:

```bash
dagger call -m github.com/kdihalas/dagger/python --source . build
```

Build a production container:

```bash
dagger call -m github.com/kdihalas/dagger/python --source . container --entrypoint python --entrypoint-args "-m,myapp"
```

Build a debug container:

```bash
dagger call -m github.com/kdihalas/dagger/python --source . debug-container
```

Publish to a registry:

```bash
dagger call -m github.com/kdihalas/dagger/python --source . publish \
  --entrypoint python \
  --entrypoint-args "-m,myapp" \
  --image-name myapp:latest \
  --registry docker.io \
  --username myuser \
  --password env:DOCKER_PASSWORD
```

## GitHub Actions

Use the Python module in GitHub Actions to test, lint, and containerize your Python applications:

```yaml
name: Build Python App

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
        run: dagger -m github.com/kdihalas/dagger/python call --source . test
      - name: Run linter
        run: dagger -m github.com/kdihalas/dagger/python call --source . lint
      - name: Build distribution
        run: dagger -m github.com/kdihalas/dagger/python call --source . build --out-dir dist
      - name: Build container
        run: dagger -m python call --source . container \
          --entrypoint github.com/kdihalas/dagger/python \
          --entrypoint-args "-m,myapp"
      - name: Publish container
        run: dagger -m github.com/kdihalas/dagger/python call --source . publish \
          --entrypoint python \
          --entrypoint-args "-m,myapp" \
          --image-name myapp:latest \
          --registry docker.io \
          --username ${{ secrets.DOCKER_USERNAME }} \
          --password ${{ secrets.DOCKER_PASSWORD }}
```
