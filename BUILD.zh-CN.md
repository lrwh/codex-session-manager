# 构建 CSM

[English](./BUILD.md)

这个文档专门放源码运行、编译和发布打包相关内容。

## 环境要求

- Go 1.24+
- `make`

## 直接从源码运行

```bash
go run ./cmd/csm --help
go run ./cmd/csm dashboard
```

## 编译本地二进制

```bash
make test
make build
./dist/csm --version
```

输出：

```text
dist/csm
```

## 跨平台构建

```bash
make clean
make build-all
```

输出文件：

```text
dist/csm-linux-amd64
dist/csm-darwin-amd64
dist/csm-darwin-arm64
dist/csm-windows-amd64.exe
```

## 版本化发布包

当前发布包是在跨平台二进制基础上继续打包生成：

```bash
tar -C dist -czf dist/csm-linux-amd64-0.1.0.tar.gz csm-linux-amd64
tar -C dist -czf dist/csm-darwin-amd64-0.1.0.tar.gz csm-darwin-amd64
tar -C dist -czf dist/csm-darwin-arm64-0.1.0.tar.gz csm-darwin-arm64
(cd dist && zip -q csm-windows-amd64-0.1.0.zip csm-windows-amd64.exe)
```

## 版本注入

当前版本号在 `Makefile` 中注入：

```makefile
VERSION ?= 0.1.0
LDFLAGS := -s -w -X main.version=$(VERSION)
```

## 发布检查清单

```bash
make test clean build-all
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
gh release create v0.1.0 ...
```
