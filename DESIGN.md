# Design

Project-specific architecture, design decisions, and domain context for gh-sweep. Generic Go and workflow conventions live in [AGENTS.md](AGENTS.md); setup and task commands live in [CONTRIBUTING.md](CONTRIBUTING.md).

## Overview

Bubble Tea TUI plus Cobra CLI for sweeping GitHub repositories: branch cleanup, protection drift, unresolved review threads, Actions performance, and a set of read-only audit views. The bare `gh-sweep` command opens a home menu with 13 views; each subcommand (`branches`, `comments`, `protection`, `gha-perf`, `orphans`, `watching`) opens its view directly or prints a table with `--list`.

- Framework: Bubble Tea v2 (`charm.land/bubbletea/v2`) with lipgloss/v2
- Theme: Catppuccin Latte/Macchiato via a hand-rolled semantic role palette
- GitHub access: REST and GraphQL through `cli/go-gh`, behind an injectable transport seam with a test-time mutation guard

## Architecture

```
├── cmd/gh-sweep/          # Entry point (main.go passes version to cli.Execute)
├── internal/
│   ├── cache/             # In-memory TTL cache + JSON gha-perf run cache
│   ├── cli/               # Cobra commands (root launches the TUI)
│   ├── config/            # YAML config loading and defaults
│   ├── git/               # Local git helpers
│   ├── github/            # REST/GraphQL client, per-domain API files,
│   │   └── transport.go   #   test transport seam and mutation guard
│   ├── models/            # Shared data structures
│   ├── orphans/           # Orphan detection (scanner, detector, types)
│   └── tui/
│       ├── main.go        # MainModel: home menu and view switching
│       ├── theme/         # Semantic color roles, background detection
│       └── components/    # One package per view, each with model.go
└── scripts/               # check-test-safety.sh
```

## GitHub Client

`internal/github` holds one file per API domain (`branches.go`, `protection.go`, `comments_graphql.go`, `gha_perf.go`, ...) hanging off a shared `Client`.

- `NewClient` builds REST and GraphQL clients through `clientOptions()`, the single place that decides transport behavior
- Under `go test` (`testing.Testing()`), `clientOptions()` pins a fake host and token to avoid keyring lookups, and injects either the transport registered via `SetTestTransport` or the default `safetyTransport`
- The mutation guard: `safetyTransport` panics on any DELETE/PATCH/POST/PUT during a test, so an unfaked test can never mutate real GitHub state. GETs pass through by design
- `NewClientWithTransport` takes an explicit `http.RoundTripper` for httptest-style tests. `NewClientWithToken` bypasses `clientOptions()` entirely (no guard); avoid it in anything a test can reach
- `scripts/check-test-safety.sh` statically rejects test files that call `gh.RESTClient`/`gh.HTTPClient`/`gh.GQLClient` directly and warns when a test constructs a client without a fake transport

Unresolved comment review uses GraphQL `reviewThreads` (REST cannot report resolution state); without `--pr` it scans the newest open PRs capped at `DefaultOpenPRCap` (20) to bound API cost.

Adding a new API surface:

1. Add a `internal/github/<domain>.go` file with methods on `*Client`
1. Route all requests through the client so the transport seam applies
1. Add a table-driven test with `NewClientWithTransport` or `SetTestTransport`
1. Run `mise run test:safety` to confirm the test never builds a raw client

## TUI Composition

`MainModel` (`internal/tui/main.go`) owns the terminal and a `ViewMode` enum. Each view is its own package under `internal/tui/components/` exposing a `Model` with value-receiver `Init`/`Update`/`View`. On a home-menu keypress the matching component is constructed fresh (`branches.NewModel(repo, base)`) and its `Init` command starts the data load; `esc` drops back to the home menu and discards the component. Async results arrive as typed messages private to each component (for example `branchesLoadedMsg`, `deleteResultMsg`) and `MainModel` forwards non-key messages to the active component only.

Adding a new TUI view:

1. Create `internal/tui/components/<name>/model.go` with `Model`, typed `...Msg` results, `NewModel`, `Init`, `Update`, `View`
1. Add the `View<Name>` const in `internal/tui/main.go`, a home-menu key case that constructs the model, and cases in `updateActive` and `renderContent`
1. Add a `golden_test.go` (build tag `golden`) with a loaded-state snapshot and a `model_internal_test.go` with state-transition tests
1. Document the keybinding in README.md (home menu table) and UX.md

## Configuration

`config.Load()` reads the first file found of `./.gh-sweep.yaml`, `~/.gh-sweep.yaml`, `~/.config/gh-sweep/config.yaml`; a missing file means `DefaultConfig()`. `.gh-sweep.yaml.example` mirrors the `Config` struct field for field. The persistent `--org` and `--repos` flags override config values, and per-command flags (for example `--stale-days`, `--days`, `--base-branch`) fall back to config only when not passed (`cmd.Flags().Changed`). `QualifiedRepos()` expands bare repo names with `default_org`.

## Caching

Two unrelated caches live in `internal/cache`:

- `memory.go`: generic in-process TTL cache
- `gha_perf_cache.go`: persistent JSON cache at `~/.cache/gh-sweep/gha-perf/<owner>_<repo>.json` storing fetched `RunTiming` records. `gha-perf` merges new runs into the cached set by run ID, so repeat invocations only fetch unseen runs; `--no-cache` skips it and `--cache-only` never fetches

## UI Design

Semantic roles defined in `internal/tui/theme`, with Catppuccin Latte for light terminals and Macchiato for dark. `theme.Detect()` honors `CATPPUCCIN_THEME` (`latte`/`light`, `macchiato`/`dark`) and otherwise reads the terminal background via lipgloss.

| Role      | Purpose                       | Latte              | Macchiato          |
| --------- | ----------------------------- | ------------------ | ------------------ |
| Primary   | Titles, focused elements      | Mauve `#8839ef`    | Mauve `#c6a0f6`    |
| Secondary | Selection backgrounds, chrome | Surface2 `#acb0be` | Surface2 `#5b6078` |
| Accent    | Selected items                | Teal `#179299`     | Teal `#8bd5ca`     |
| Muted     | Subtitles, help text          | Overlay2 `#7c7f93` | Overlay2 `#939ab7` |
| Text      | Normal text                   | `#4c4f69`          | `#cad3f5`          |
| Success   | Healthy states                | Green `#40a02b`    | Green `#a6da95`    |
| Warning   | Warnings, active highlights   | Yellow `#df8e1d`   | Yellow `#eed49f`   |
| Error     | Errors, overdue states        | Red `#d20f39`      | Red `#ed8796`      |

## Testing

Four layers:

- Unit tests: table-driven with `t.Parallel`, next to the code, the default landing spot for new tests
- Golden snapshots (`//go:build golden`, `mise run test:golden`, `-update` to refresh): a terminal-size matrix for the home view in `internal/tui/testdata` plus one loaded-state snapshot per component
- teatest (`internal/tui/teatest_internal_test.go`): boots the full program against a canned fake GitHub transport and drives navigation end to end
- Safety: the runtime mutation guard in `transport.go` plus the static `scripts/check-test-safety.sh` scan

Coverage targets by tier: critical paths (`internal/github`, `internal/config`, `internal/models`) above 80%, UI components above 60%. Current actuals (total ~50%): models 100%, tui 87%, github 77%, config 79% meet or near their bars; `internal/cli` (~26%, `--list` render paths untested) and the branches component (~12%) are the known gaps, tracked as ROADMAP M3. The `test:coverage-min` task gates at 70% total but is not yet wired into `ci`.

## External Dependencies

- `gh` CLI authentication is reused via `cli/go-gh` (v1); no token setup is needed when `gh auth login` has run
- `charm.land/bubbletea/v2` and `charm.land/lipgloss/v2` for the TUI; `charmbracelet/x/exp/golden` and `x/exp/teatest/v2` in tests
- `spf13/cobra` for the CLI, `gopkg.in/yaml.v3` for config

## Release Checklist

1. `mise run ci` (tests, golden pass, build)
1. Merge conventional commits to `main`; `bump_version.yml` runs commitizen to tag and goreleaser to publish binaries (`gh-sweep-<os>-<arch>`)
1. Update `Formula/gh-sweep.rb` version and SHA256 values (`mise run brew:sha` prints the steps)
1. Verify `gh extension install kyleking/gh-sweep` finds the binaries, which are downloaded by that exact naming
