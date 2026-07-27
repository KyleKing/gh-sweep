## v0.4.2 (2026-07-27)

### Fix

- **ci**: skip the release step when commitizen cut no new tag

## v0.4.1 (2026-07-26)

### Fix

- **deps**: migrate to go-gh v2.11.1 for GHSA-55v3-xh23-96gh
- **deps**: bump golang.org/x/net to 0.55.0 for GHSA-4374-p667-p6c8 and six more advisories

## v0.4.0 (2026-07-26)

### Feat

- **tasks**: rename the run task to dev so the mise shorthand works

### Fix

- **lint**: drop the deprecated gomodguard linter, whose blocked list was empty
- **format**: pin golines and wrap at 120 to match the lll limit

### Refactor

- **mise**: move template-managed tool pins into conf.d

## v0.3.3 (2026-07-26)

### Fix

- track cmd/gh-sweep, which a bare gh-sweep gitignore entry had excluded

## v0.3.2 (2026-07-26)

### Fix

- **ci**: strip ANSI before teatest output matching and stop mise auto-installing unused tools

## v0.3.1 (2026-07-26)

### Fix

- **release**: ignore the commitizen body.md so goreleaser's dirty check passes

## v0.3.0 (2026-07-26)

### Feat

- **watching**: add Ignored tab and GraphQL-backed repo metadata

### Fix

- **ci**: repair golangci config for the v2 schema and bump the lint pin to 2.12.2

## v0.2.0 (2026-07-26)

### Feat

- switch to filtered menu

### Fix

- drop MISE_ENV gating for hk tool pins

## v0.1.0 (2026-07-25)

### Feat

- wire protection cli and replace fabricated analytics data
- real unresolved pr comment review via graphql review threads
- implement branch management with guarded batch delete
- wire config loading into the tui and cli
- analyze gha performance (TBD)
- implement GitHub Action Performance reporting
- implement orphan branch sweep
- ensure that repository 'watch' status is set as expected
- add comprehensive tests, debugging features, and functional enhancements
- implement Phase 2 & 3 TUI components (settings, webhooks, access, releases)
- complete MVP with Phase 5 TUI components and live API integration
- implement Phases 1-4 (branch management through integrations)
- add core infrastructure (config, cache, GitHub/Git clients)
- initial gh-sweep scaffolding with Bubble Tea TUI

### Fix

- populate branch commit dates in branch listings

### Refactor

- migrate to bubble tea v2 with semantic theme
- adopt go template tooling and restructure layout
