# Troubleshooting

Toolchain failures that look like project bugs. Each entry names the symptom, the
check that tells you which cause you have, and the fix.

## Reinstall first

Most tool and hook problems clear with a forced reinstall:

```bash
mise install --force
hk install --mise --force
```

On git 2.55+, `hk install --mise` writes `hook.hk-*.command` entries into
`.git/config` and creates no `.git/hooks/pre-commit`. Check with
`git config --get-regexp '^hook\.'`, because a file-existence check reports a
false negative.

## `GOPROXY list is not the empty string, but contains no entries`

```
go: golang.org/x/tools/cmd/goimports@latest: GOPROXY list is not the empty string, but contains no entries
mise ERROR go failed
```

The mise-managed Go install is corrupt. Every mise Go install ships a `go.env`
holding the `GOPROXY`, `GOSUMDB`, and `GOTOOLCHAIN` defaults, and a truncated
install loses that file along with `src/`. An empty `GOPROXY` then breaks every
module download.

Confirm the install rather than the config:

```bash
mise exec -- go env GOPROXY GOSUMDB
# healthy:
# https://proxy.golang.org,direct
# sum.golang.org
```

An empty first line means the install is broken. The global config at
`~/Library/Application Support/go/env` (Linux: `~/.config/go/env`) is a separate
file and is usually fine.

Reinstall the version rather than pinning `GOPROXY` in `mise.toml`. Setting those
variables by hand papers over the corruption and hides it from the next person:

```bash
mise uninstall go@<version>
mise install go@<major.minor>
```

The same corruption also shows up as `"fmt" is not in std` or similar
"not in std" errors for standard library packages, because `src/` is missing.

## `go tool version does not match`

```
compile: version "go1.X" does not match go tool version "go1.Y"
```

Two Go toolchains are on `PATH` and `go` loaded a `compile` from the other tree.
In CI this means a job pairs `actions/setup-go` with `jdx/mise-action`. The
generated `ci.yml` keeps them in separate jobs on purpose: the `ci` and `hooks`
jobs use mise alone, and `lint` and `benchmark` use `setup-go` alone. Setting
`GOROOT: ""` does not fix it, because `mise run` re-exports `GOROOT`.

Locally, the same failure means a system Go install shadows the mise one. Check
with `which -a go` and `mise exec -- go version`.

## A pinned tool version does not exist

```
mise ERROR failed to install golangci-lint@<version>
```

Check the tag upstream before assuming a network problem. Pins live in
`.config/mise/conf.d/template.toml`, and the golangci-lint pin is duplicated in
the `lint` job of `.github/workflows/ci.yml`. Both have to move together.
`mise ls-remote golangci-lint` lists the versions mise will accept.

Tools installed from source (`go:` prefixed pins) build against the Go version
mise resolves, so a tool that requires a newer Go than the project pins makes
mise download a second Go release for that build. That is expected and not a
corruption.

## Golden fixtures rewritten on commit

Golden files are byte-exact snapshots. `hk.pkl` excludes `**/*.golden` from every
whitespace fixer, so a fixture stored under any other name is silently rewritten
on commit. Rename it to `*.golden` or add its glob to that exclude.

Regenerate with `go test -tags=golden -run TestGolden ./internal/tui/... -update`
and review the diff. Never hand-edit.

## Debugging a single test

```bash
go test -v -run TestName ./package
```

## gh-sweep runtime

Every view errors on authentication because gh-sweep reuses the `gh` CLI login.
Run `gh auth status`, then `gh auth login` if needed.

A view is empty because no repos are in scope. Pass `--org` or `--repos`, or set
`default_org` and `repositories` in the
[config file](./configuration.md).

Comments finds nothing on a busy repo because, without `--pr`, it only scans the
newest 20 open pull requests. Pass `--pr <number>` for an older one.

A repo shows as unwatched when GitHub says otherwise, because neither REST nor
GraphQL exposes the "Custom" per-notification-type watch setting. A repo left at
Custom reports the same state as one at the plain default. This is a GitHub API
limit, so gh-sweep says so rather than asserting a state it cannot confirm.

`gha-perf` reports stale numbers because it merges into a JSON cache under
`~/.cache/gh-sweep/gha-perf/`. Pass `--no-cache` to ignore it for one run.

`brew install --cask KyleKing/tap/gh-sweep` fails. The tap holds no gh-sweep cask
until the first release published after the `TAP_DEPLOY_KEY` secret was added on
2026-07-30; v0.5.0 predates it. Install through `gh extension install` or
`go install` until a later release appears in the tap.
