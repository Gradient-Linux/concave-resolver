# concave-resolver

`concave-resolver` is the environment intelligence daemon for Gradient Linux.
It watches the Python package layer, stores environment snapshots, and exposes
its state over a local Unix socket at `/run/gradient/resolver.sock`.

## What is here

- `internal/resolver/types.go` - shared contracts for snapshots, drift, and daemon status
- `internal/resolver/diff.go` - pure drift classification and snapshot diff logic
- `internal/resolver/store.go` - snapshot pathing, persistence, and lookup helpers
- `internal/resolver/scanner.go` - mockable container scanner shell
- `internal/resolver/socket.go` - Unix socket server and client helpers
- `internal/resolver/service.go` - daemon loop skeleton
- `scripts/gradient-resolver.service` - systemd unit file

## Build

```bash
go test ./...
go build ./...
```

## Run

```bash
go run . run
```

The daemon starts a Unix socket server and performs periodic scan cycles for
configured targets. Targets can be added on the command line for manual runs.

## Snapshot store

Snapshots are written under:

`~/gradient/config/env-snapshots/`

File names follow:

`<group>.<RFC3339 timestamp>.lock`

The `default` group is used when no group name is provided.

