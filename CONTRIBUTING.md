# Contributing

## Branching

- Work on `feature/resolver`.
- Keep changes limited to this repository.
- Do not revert unrelated edits if the tree is dirty.

## Development

- Keep runtime behavior real, but keep tests hermetic.
- Do not require live Docker, Poetry, or pip in unit tests.
- Preserve the socket path `/run/gradient/resolver.sock`.
- Preserve the snapshot path `~/gradient/config/env-snapshots/`.

## Verification

Run:

```bash
go test ./...
go build ./...
```

## Style

- Prefer small, focused functions.
- Keep exported identifiers documented.
- Keep the resolver package free of direct HTTP servers.

