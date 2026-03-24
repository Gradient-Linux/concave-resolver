# Contributing to concave-resolver

Contributions are welcome for daemon behavior, socket contracts, snapshot handling, tests, and documentation. Keep changes scoped to environment intelligence. Docker lifecycle, GPU setup, and suite management belong elsewhere in the Gradient Linux stack.

## Before you start

Read [README.md](README.md) and [docs/README.md](docs/README.md) before changing runtime behavior or the socket protocol.

## Development setup

Use Ubuntu 24.04 with Go 1.25 or newer.

```bash
git clone <repo-url>
cd concave-resolver
go build -o gradient-resolver .
go test ./...
./gradient-resolver help
```

If you want to exercise the live scanner, run the daemon against a machine that already has Docker and Gradient Linux suites available.

## Making changes

### Branching

Use one of these branch prefixes:

- `feat/<slug>`
- `fix/<slug>`
- `docs/<slug>`

### Commit messages

Format commits as `<type>(<scope>): <summary>`.

Use these types:

- `feat`
- `fix`
- `refactor`
- `test`
- `docs`
- `chore`

Keep the summary under 72 characters.

Examples:

- `feat(resolver): add snapshot pruning guard`
- `fix(socket): return unavailable on missing daemon`
- `docs(readme): clarify development preview status`

### Tests

- Add or update unit tests for any new function or behavior change.
- Run `go test ./...` before opening a pull request.
- Run `go test -race ./...` when you touch concurrency, socket handling, or state mutation.
- Keep tests hermetic. Unit tests must not require live Docker, Poetry, or pip.

### Pull requests

- Keep each pull request to one logical change.
- Explain what changed, why it changed, and how you verified it.
- Make API or socket changes explicit in the pull request description.

## Code conventions

- Keep the resolver focused on environment intelligence.
- Preserve `/run/gradient/resolver.sock` as the default socket path unless there is a documented compatibility reason to change it.
- Preserve `~/gradient/config/env-snapshots/` as the default snapshot root.
- Keep drift classification pure and easy to test.
- Use injectable command runners for container scanning so tests stay local.
- Return structured errors with context instead of swallowing failures.

## What we don't accept

- Dependencies added without prior discussion in an issue.
- Code that writes outside the Gradient workspace without explicit operator intent.
- Direct HTTP servers added to the resolver daemon.
- Shell string interpolation with user-controlled input.

## License

By contributing, you agree that your contributions will be released under the repository license when one is published.
