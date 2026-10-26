# Contributing to Rivet

Thanks for your interest in Rivet! This project follows the conventions used
across the [GrayCodeAI](https://github.com/GrayCodeAI) repositories.

## Quick start

1. Fork the repo and create a feature branch off `main`:

   ```bash
   git checkout -b feat/short-description
   ```

2. Make your changes in small, focused commits.
3. Run the local checks before pushing:

   ```bash
   gofmt -l .        # must print nothing
   go vet ./...
   go test -race ./...
   ```

4. Open a pull request. CI runs the same checks on every push and pull request.

## Build & test

Rivet is a Go module (`go 1.27.2`) with no third-party dependencies. Build the
CLI with:

```bash
go build -o rivet ./cmd/rivet
```

| Command                 | What it does                                    |
| ----------------------- | ----------------------------------------------- |
| `gofmt -l .`            | List files that need formatting (must be empty) |
| `go vet ./...`          | Run the standard Go analysers                   |
| `go test ./...`         | Run the unit tests                              |
| `go test -race ./...`   | Run the unit tests with the race detector       |
| `go build ./cmd/rivet` | Build the CLI                                   |

Tests use temporary projects; do not run block scripts from the repository
during tests. Keep receipts bound to the script, check, and manifest hashes.

## Commit messages

We use [Conventional Commits](https://www.conventionalcommits.org/). Keep the
subject line short and use the body to explain what changed and why.

```
<type>(<optional scope>): <short summary>
```

Common types: `feat:`, `fix:`, `perf:`, `refactor:`, `docs:`, `test:`,
`build:`, `ci:`, `chore:`, and `revert:`.

Mark a breaking change with `!` after the type or scope (for example
`refactor!: ...`) and call it out in `CHANGELOG.md`.

## Pull request checklist

Before requesting review:

- [ ] `gofmt -l .` prints nothing.
- [ ] `go vet ./...` and `go test -race ./...` pass.
- [ ] New behaviour has tests; bug fixes have a regression test.
- [ ] User-visible changes are listed under `## [Unreleased]` in
      `CHANGELOG.md` (Keep a Changelog: Added / Changed / Fixed / Removed /
      Security).
- [ ] Documentation and the README are updated when behaviour changes.
- [ ] No secrets, API keys, or personal data in code, tests, fixtures, or logs.

## Code review etiquette

- Reviewers focus on correctness, design, and tests; formatting is enforced by
  tooling.
- Authors respond to every comment (resolved, addressed, or declined with a
  reason).
- Squash-merge by default; the PR title becomes the commit, so it must be a
  valid Conventional Commit message.
- One maintainer approval is required before merging.

## Reporting bugs

Open an issue with the commit you tested (`git rev-parse HEAD`), reproduction
steps, expected behaviour, and actual behaviour. Rivet has not cut a tagged
release yet, so the commit SHA is the version.

## Reporting security issues

**Do not open a public issue.** See [SECURITY.md](./SECURITY.md) for private
reporting channels.

## License

By contributing, you agree that your contributions are licensed under the MIT
License (see [LICENSE](./LICENSE)).
