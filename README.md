# CSM

English | [简体中文](./README.zh-CN.md)

`CSM` stands for `codex-session-manager`.

Its core purpose is simple:

> when people switch between multiple Codex accounts, old sessions become hard to find again. `CSM` is built to solve that specific problem.

![CSM Dashboard](./csm.png)

`CSM` is a lightweight local tool for scanning and browsing Codex sessions. It focuses on a few practical workflows:

- help users recover old sessions after switching across multiple Codex accounts
- list all sessions with a single command
- browse sessions in a local dashboard
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
- Manual cluster operations: `merge`, `split`, `tag`, `reset`
- Session titles prefer native Codex naming:
  - `thread_name` from `~/.codex/session_index.jsonl`
  - `thread_name_updated` from the session rollout file

## Main entry points

```bash
./dist/csm
./dist/csm dashboard
```

- `csm`: scan and print session list
- `csm dashboard`: open the local web dashboard

## Quick Start

### 1. Initialize

```bash
go run ./cmd/csm init
```

### 2. Add a Codex source

```bash
go run ./cmd/csm source add ~/.codex
go run ./cmd/csm source list
```

### 3. Scan sessions

```bash
go run ./cmd/csm scan
```

### 4. Print sessions directly

```bash
go run ./cmd/csm
go run ./cmd/csm -n 20
go run ./cmd/csm --verbose -n 1
go run ./cmd/csm --json -n 10
```

### 5. Start the dashboard

```bash
go run ./cmd/csm dashboard
go run ./cmd/csm dashboard --no-open
go run ./cmd/csm dashboard --addr 127.0.0.1:7788
```

### 6. Search and cluster operations

```bash
go run ./cmd/csm find session
go run ./cmd/csm cluster rebuild
go run ./cmd/csm cluster list -n 20
go run ./cmd/csm show <cluster-id>
go run ./cmd/csm tag set <cluster-id> my-cluster
go run ./cmd/csm cluster merge <target-cluster-id> <source-cluster-id...>
go run ./cmd/csm cluster split <source-cluster-id> <session-id...>
go run ./cmd/csm cluster reset <cluster-id>
```

## Build

```bash
make test
make build
```

After building:

```bash
./dist/csm --help
./dist/csm dashboard
```

Cross-platform binaries:

```bash
make build-all
```

Build output:

```text
dist/
```

## Local data

Default working directory:

```text
~/.config/csm
```

Override with:

```bash
CSM_HOME=/path/to/csm-home ./dist/csm
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

No license file is included yet. Add `MIT` or `Apache-2.0` before publishing publicly.
