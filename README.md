# concave-resolver

Environment intelligence for Gradient Linux, delivered as a local Unix-socket daemon.

## What it does

`concave-resolver` tracks Python package drift inside Gradient Linux workloads without taking over Docker lifecycle or GPU management. It scans container environments, stores snapshot history under the Gradient workspace, classifies package changes by drift tier, and exposes resolver state to `concave` over `/run/gradient/resolver.sock`.

## Requirements

- Ubuntu 24.04 LTS
- Go 1.25+
- Docker Engine 26+
- Access to the Gradient workspace at `~/gradient/`

## Status

`concave-resolver` is in development for Gradient Linux v0.2. The repository already contains the daemon loop, socket protocol, snapshot store, and drift classification logic. Release packaging and full production wiring are still in progress.

## Configuration

The daemon reads and writes inside the standard Gradient workspace:

- Snapshots: `~/gradient/config/env-snapshots/`
- Default socket: `/run/gradient/resolver.sock`

The CLI also accepts runtime flags for local development:

- `--workspace`
- `--socket`
- `--interval`
- `--target group:container`

## Architecture

`concave-resolver` owns the Python environment layer only. It does not install Docker images, manage GPU drivers, or change suite topology. `concave` remains the control plane and queries resolver state over the local socket when environment status or drift reports are requested.

## Development

### Prerequisites

Install Go 1.25 or newer and ensure Docker is available if you want to scan live containers.

### Build

```bash
go build -o gradient-resolver .
```

### Test

```bash
go test ./...
```

### Run locally

```bash
./gradient-resolver run --target research:gradient-neural-torch
./gradient-resolver status
./gradient-resolver scan --group research --container gradient-neural-torch
```

### Repo layout

```text
concave-resolver/
  internal/resolver/   daemon loop, socket protocol, scanner, drift logic
  scripts/             systemd unit file
  docs/                repository docs
  main.go              CLI entrypoint
```

## Roadmap

The current line focuses on v0.2 environment drift detection, snapshot history, and resolver status reporting. Later work extends rollback depth, richer policy handling, and tighter integration with team-aware environment baselines.

## License

License terms have not been published in this repository yet.
