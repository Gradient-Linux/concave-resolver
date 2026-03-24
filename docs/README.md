# Documentation

This repository contains the Phase 11 resolver scaffold for Gradient Linux.

## Key contracts

- Unix socket: `/run/gradient/resolver.sock`
- Snapshot store: `~/gradient/config/env-snapshots/`
- Service entrypoint: `gradient-resolver run`

## Implementation notes

- Drift classification is pure and lives in `internal/resolver/diff.go`.
- Snapshot persistence is atomic and lives in `internal/resolver/store.go`.
- The scanner is intentionally mockable so tests do not need Docker.
- The socket server speaks newline-delimited JSON over a Unix domain socket.

