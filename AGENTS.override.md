# concave-resolver override

This repository is owned by the Resolver Agent.

- Branch: `feature/resolver`
- Scope: `internal/resolver/*`, repo root docs, and `scripts/gradient-resolver.service`
- Keep tests hermetic. Do not require a running Docker daemon, Poetry, or pip in unit tests.
- Preserve the Unix socket contract at `/run/gradient/resolver.sock`.
- Preserve the snapshot store contract under `~/gradient/config/env-snapshots/`.

