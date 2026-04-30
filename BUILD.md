# Build CSM

This document covers source build, local run, and release packaging.

## Requirements

- Go 1.24+
- `make`

## Run From Source

```bash
go run ./cmd/csm --help
go run ./cmd/csm dashboard
```

## Build A Local Binary

```bash
make test
make build
./dist/csm --version
```

Output:

```text
dist/csm
```

## Cross-Platform Builds

```bash
make clean
make build-all
```

Outputs:

```text
dist/csm-linux-amd64
dist/csm-darwin-amd64
dist/csm-darwin-arm64
dist/csm-windows-amd64.exe
```

## Versioned Release Packages

Current release packages are built from the cross-platform binaries:

```bash
tar -C dist -czf dist/csm-linux-amd64-0.2.1.tar.gz csm-linux-amd64
tar -C dist -czf dist/csm-darwin-amd64-0.2.1.tar.gz csm-darwin-amd64
tar -C dist -czf dist/csm-darwin-arm64-0.2.1.tar.gz csm-darwin-arm64
(cd dist && zip -q csm-windows-amd64-0.2.1.zip csm-windows-amd64.exe)
```

## Version Injection

The current version is injected at build time from `Makefile`:

```makefile
VERSION ?= 0.2.1
LDFLAGS := -s -w -X main.version=$(VERSION)
```

## Release Checklist

```bash
make test clean build-all
git tag -a v0.2.1 -m "v0.2.1"
git push origin v0.2.1
gh release create v0.2.1 ...
```
