# concave-resolver docs

This directory tracks the public repository documentation for `concave-resolver`.

## Start here

- [README.md](../README.md) explains the daemon boundary, runtime paths, and local commands.
- [CONTRIBUTING.md](../CONTRIBUTING.md) covers build, test, and review expectations.

## Runtime contract

- Default socket: `/run/gradient/resolver.sock`
- Snapshot root: `~/gradient/config/env-snapshots/`
- CLI entrypoints: `run`, `status`, `scan`

## Scope

`concave-resolver` owns environment snapshots and drift reporting. It does not manage Docker images, workspace layout, or GPU drivers.
