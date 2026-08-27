## v0.8.0 (2026-08-27)

### Feat

- **github**: share the transport test seam and mutation guard via aragonite
- **tui**: reorganize home menu by scope with letter shortcuts and scrolling

## v0.7.0 (2026-08-20)

### Feat

- **tui**: scroll the secrets lists instead of overflowing the terminal
- **tui**: scroll the watching list instead of overflowing the terminal
- **tui**: scroll the collaborators list instead of overflowing the terminal
- **tui**: scroll the protection list instead of overflowing the terminal
- **tui**: scroll the comments list instead of overflowing the terminal
- **tui**: scroll the releases list instead of overflowing the terminal
- **tui**: scroll the ghaperf tables instead of overflowing the terminal
- **tui**: scroll the settings overview instead of overflowing the terminal
- **tui**: scroll the webhooks list instead of overflowing the terminal
- **tui**: scroll the policy diff list instead of overflowing the terminal
- **tui**: scroll the branches list instead of overflowing the terminal
- **tui**: scroll the orphans list instead of overflowing the terminal
- add read-only policy-check workflow and shared scroll windowing
- **secrets**: wire live unused-secrets detection into the TUI
- **ghaperf**: add duration trend sparklines, branch bars, regression markers
- **pages**: add gh-sweep pages command to audit Pages domains against DNS
- **watching**: open repo in browser, trim per-row metadata, rename default tab
- **orphans,watching**: guard destructive cleanup and watch-all behind confirmation
- add reusable policy-apply workflow with environment approval gate
- add ? help legend and g/G jump to the remaining tabbed views
- add ? help legend, / search, and g/G jump to list views
- add inverse selection to branches, orphans, and watching

### Fix

- bootstrap branch protection instead of failing on unprotected repos

### Refactor

- **github,tui**: merge WorkflowRun into RunTiming for one GHA analytics model
- **tui**: fully clear internal/tui of the legacy lint exclusion
- **tui**: clean up watching lint issues
- **tui**: clean up orphans lint issues
- **tui**: clean up ghaperf lint issues
- **tui**: clean up policy lint issues
- **tui**: clean up branches lint issues
- **tui**: clean up secrets lint issues
- **tui**: clean up collaborators lint issues
- **tui**: drop stale gocritic nolint from analytics cleanup
- **tui**: clean up analytics lint issues
- **tui**: clean up comments lint issues
- **tui**: clean up webhooks lint issues
- **tui**: clean up releases lint issues
- **tui**: clean up protection lint issues
- **tui**: clean up settings lint issues
- **github**: fully clear internal/github of the legacy lint exclusion
- **cli**: fully clear internal/cli of the legacy lint exclusion
- **orphans**: fully clear internal/orphans of the legacy lint exclusion
- **models**: fully clear internal/models of the legacy lint exclusion
- **cache**: fully clear internal/cache of the legacy lint exclusion
- **config**: fully clear internal/config of the legacy lint exclusion
- **git**: fully clear internal/git of the legacy lint exclusion
- **config,github,models,policy,orphans,pages**: fix tagliatelle, musttag, containedctx, named returns
- **cli,github,orphans,tui**: wrap external errors, fix line length, add a scoped gochecknoinits exception

## v0.6.0 (2026-08-05)

### Feat

- add gh-sweep policy command for declarative repo settings sync

## v0.5.0 (2026-07-30)

### Feat

- **release**: publish a Homebrew cask from goreleaser

## v0.4.4 (2026-07-30)

### Fix

- **release**: build each target into its own dist path

## v0.4.3 (2026-07-27)

### Refactor

- rename camelCase Go files to snake_case

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
