# Development Guide

## Prerequisites

- Go 1.24+
- `just` task runner

## Build & Test

```bash
# Verify formatting, run vet, and run tests
just check

# Build the Linux amd64 release binary
just build

# Compile locally for current architecture
go build -o differ .
```

## Release Workflow

Releases are automated via GitHub Actions using semantic version tags.
See [Delnegend/actions](https://github.com/Delnegend/actions/blob/main/README.md) for shared workflow mechanics.

```bash
# Bump version in VERSION file
just version set 1.2.0
```
