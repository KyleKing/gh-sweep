# Contributing to gh-sweep

## Setup

Prerequisites: Go (see `go.mod`), [mise](https://mise.jdx.dev/), [hk](https://hk.jdx.dev/)

```bash
mise install
hk install --mise
mise run ci
```

## Tasks

Shared tasks live in `.config/mise/conf.d/template.toml` (managed by the copier template). Project-specific tasks go in additional `.config/mise/conf.d/*.toml` files.

| Command                       | Description                                                           |
| ----------------------------- | --------------------------------------------------------------------- |
| `mise run bench`              | Run benchmarks                                                        |
| `mise run brew:sha`           | Print the SHA256 steps for the Homebrew formula (run after a release) |
| `mise run build`              | Build binary                                                          |
| `mise run ci`                 | Full CI check (tests + golden tests + build)                          |
| `mise run clean`              | Clean build artifacts                                                 |
| `mise run demo`               | Generate VHS demo recordings                                          |
| `mise run format`             | Auto-fix lint and formatting                                          |
| `mise run hooks`              | Run git hooks                                                         |
| `mise run lint`               | Run linter                                                            |
| `mise run run`                | Run from source (`go run`, always reflects current code)              |
| `mise run test`               | Run tests with coverage                                               |
| `mise run test:coverage-min`  | Verify the 70% coverage threshold                                     |
| `mise run test:golden`        | Run golden snapshot tests                                             |
| `mise run test:golden-update` | Regenerate golden snapshots                                           |
| `mise run test:safety`        | Check tests for un-faked GitHub client construction                   |
| `mise run test:view-coverage` | View coverage report in browser                                       |
| `mise tasks`                  | List all available tasks                                              |

## Code Guidelines

Follow [AGENTS.md](AGENTS.md) for code organization, testing patterns, and error handling.

Linting is configured in `.golangci.toml`. Run `mise run format` to auto-fix. A legacy exclusion block suppresses the style and complexity linters across `internal/` until the component refactor (ROADMAP M3); correctness linters stay enforced everywhere.

## Git Workflow

Conventional commits enforced via [commitizen](https://commitizen-tools.github.io/commitizen/):

```
<type>(<scope>): <subject>
```

Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`

Git hooks run automatically via hk on commit and push.

## Development Install

Run straight from source with `go run`, which always reflects the current code, so there's no built binary or installed extension to go stale between edits:

```bash
go run ./cmd/gh-sweep [args]
# or
mise run run -- [args]
```

To test the actual `gh sweep` extension invocation or a Homebrew install, use the released version rather than installing from this checkout:

```bash
gh extension install kyleking/gh-sweep
# or
brew install --formula https://github.com/KyleKing/gh-sweep/raw/main/Formula/gh-sweep.rb
```

## Releases

Merging a releasable Conventional Commit to `main` triggers the `bump_version` workflow: commitizen bumps the version, tags, and updates the changelog, then goreleaser builds the binaries and publishes the GitHub release in the same workflow (tags pushed with the default `GITHUB_TOKEN` cannot trigger a second workflow). A manually pushed `v*` tag runs `release.yml` as a fallback.

After a release:

1. Verify the properly named binaries are attached (`gh-sweep-linux-amd64`, `gh-sweep-darwin-arm64`, etc.), since `gh extension install kyleking/gh-sweep` and the formula download them by that exact naming
1. Update the `version` and `sha256` values in `Formula/gh-sweep.rb`; `mise run brew:sha` prints the `shasum` commands

Users install from the repository formula:

```bash
brew install --formula https://github.com/KyleKing/gh-sweep/raw/main/Formula/gh-sweep.rb
```

## Troubleshooting

```bash
mise install --force   # Reinstall tools
hk install --mise --force  # Reinstall hooks
go test -v -run TestName ./package  # Debug specific test
go test -tags=golden -run TestGolden ./internal/tui/... -update  # Refresh stale goldens
```
