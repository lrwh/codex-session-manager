# CSM

English | [简体中文](./README.zh-CN.md)

`CSM` stands for `codex-session-manager`.

Current version: `0.2.1`

Its core purpose is simple:

> when people switch between multiple Codex accounts, old sessions become hard to find again. `CSM` is built to solve that specific problem.

![CSM Dashboard](./csm.png)

`CSM` is a lightweight local tool for scanning and browsing Codex sessions. It focuses on a few practical workflows:

- help users recover old sessions after switching across multiple Codex accounts
- list all sessions with a single command
- browse sessions in a local dashboard
- open a session detail page and inspect the full conversation timeline
- search inside a single session detail view
- search by title, session id, preview, cwd, or file path
- show a ready-to-copy resume command for each session
- keep cluster operations available without making cluster the primary entry point

## Features

- Written in Go
- Works on Windows, macOS, and Linux
- No database
- No external service dependency
- Local JSON / JSONL storage
- CLI and local dashboard modes
- Session detail page with full content timeline and in-page search
- Manual cluster operations: `merge`, `split`, `tag`, `reset`
- Session titles prefer native Codex naming:
  - `thread_name` from `~/.codex/session_index.jsonl`
  - `thread_name_updated` from the session rollout file

## Main entry points

```bash
csm
csm dashboard
```

- `csm`: scan and print session list
- `csm dashboard`: open the local web dashboard and auto-prepare data on first run

## Install

Download the latest package from GitHub Releases:

`https://github.com/lrwh/codex-session-manager/releases/latest`

Use the package that matches your platform, then install `csm` into your `PATH`.

### Linux

```bash
curl -L -o csm-linux-amd64.tar.gz https://github.com/lrwh/codex-session-manager/releases/latest/download/csm-linux-amd64-0.2.1.tar.gz
tar -xzf csm-linux-amd64.tar.gz
sudo install -m 755 csm-linux-amd64 /usr/local/bin/csm
csm --version
```

### macOS Intel

```bash
curl -L -o csm-darwin-amd64.tar.gz https://github.com/lrwh/codex-session-manager/releases/latest/download/csm-darwin-amd64-0.2.1.tar.gz
tar -xzf csm-darwin-amd64.tar.gz
sudo install -m 755 csm-darwin-amd64 /usr/local/bin/csm
csm --version
```

### macOS Apple Silicon

```bash
curl -L -o csm-darwin-arm64.tar.gz https://github.com/lrwh/codex-session-manager/releases/latest/download/csm-darwin-arm64-0.2.1.tar.gz
tar -xzf csm-darwin-arm64.tar.gz
sudo install -m 755 csm-darwin-arm64 /usr/local/bin/csm
csm --version
```

### Windows

1. Download `csm-windows-amd64-0.2.1.zip` from Releases.
2. Unzip it and rename `csm-windows-amd64.exe` to `csm.exe`.
3. Move it to a stable directory such as `C:\Tools\csm\`.
4. Add that directory to `PATH`.
5. Open a new terminal and run `csm --version`.

## Update

After installation, update with:

```bash
csm update
```

Notes:

- Linux and macOS can replace the current executable automatically
- Windows currently downloads the latest package and prompts for manual replacement

## Quick Start

### 1. Fastest start

If this is your first run, you can start directly with:

```bash
csm dashboard
```

On first run, `csm dashboard` will automatically:

- create the local CSM home
- create `config.json` and `sources.json`
- add `~/.codex` as the default source when available
- scan sessions and rebuild cluster data

### 2. Initialize manually if you want explicit control

```bash
csm init
```

`csm init` is optional. It only creates the local working files explicitly.

### 3. Add a Codex source

```bash
csm source add ~/.codex
csm source list
```

### 4. Scan sessions

```bash
csm scan
```

### 5. Print sessions directly

```bash
csm
csm -n 20
csm --verbose -n 1
csm --json -n 10
```

### 6. Start the dashboard

```bash
csm dashboard
csm dashboard --no-open
csm dashboard --addr 127.0.0.1:7788
```

### 7. Search and cluster operations

```bash
csm find session
csm cluster rebuild
csm cluster list -n 20
csm show <cluster-id>
csm tag set <cluster-id> my-cluster
csm cluster merge <target-cluster-id> <source-cluster-id...>
csm cluster split <source-cluster-id> <session-id...>
csm cluster reset <cluster-id>
```

## Build From Source

For source build and packaging, see [BUILD.md](./BUILD.md).

## Local data

Default working directory:

```text
~/.config/csm
```

Override with:

```bash
CSM_HOME=/path/to/csm-home csm
```

Main local files:

- `config.json`
- `sources.json`
- `session_index.jsonl`
- `clusters.json`
- `tags.json`

## Command overview

```bash
csm
csm sessions
csm init
csm source add <path>
csm source list
csm scan
csm find <query>
csm dashboard
csm update
csm cluster rebuild
csm cluster list
csm cluster merge
csm cluster split
csm cluster reset
csm show <cluster-id>
csm tag set <cluster-id> <name>
csm tag remove <cluster-id>
```

## Notes

- `CSM` is primarily a session recovery product, not a cluster-heavy platform
- `CSM` is session-first; cluster is secondary
- Session titles only use native Codex rename sources
- If a session has no native rename metadata, `CSM` falls back to a derived summary title

## License

Licensed under `Apache-2.0`. See [LICENSE](./LICENSE).
