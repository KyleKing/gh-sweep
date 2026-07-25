# Roadmap

Phased, shippable milestones for gh-sweep. Each milestone stands on its own and can be released independently; the roadmap can stop at any point without leaving the app half-migrated.

Architecture and domain context live in [DESIGN.md](DESIGN.md); Go and workflow conventions live in [AGENTS.md](AGENTS.md).

## Vision

A cross-repo sweep and audit tool: point it at an org or a repo list and find what needs attention (dead and orphaned branches, unresolved review threads, protection and settings drift, slow or flaky workflows, risky configuration) then act on it interactively with previews and confirmations. Automation tools own recurring tasks; gh-sweep owns the one-off passes that need human judgment (see [docs/alternatives.md](docs/alternatives.md)).

## Testing strategy

Four layers, detailed in [DESIGN.md](DESIGN.md#testing): table-driven unit tests as the base, golden snapshots behind a build tag for visual regression, teatest end-to-end runs against a fake GitHub transport, and a two-part safety net (runtime mutation guard plus a static scan) so no test can mutate real GitHub state. New features land with unit tests first; golden snapshots stay deliberately few to avoid brittle churn.

## Shipped

- Layout restructure onto the Go template tooling (52f100f)
- Config loading wired into TUI and CLI with flag precedence (521b4c1)
- Bubble Tea v2 migration with the semantic theme (10adee3)
- Branch management with guarded batch delete (ba2ff65)
- Real unresolved-comment review via GraphQL review threads (707df75)
- Protection CLI wiring and real analytics data (fcb59dc)
- Orphan branch sweep (e2e40a1) and watch-status enforcement (bbe525b)
- GHA performance reporting with cache and CSV export (04628bd)
- Release plumbing: goreleaser, commitizen bump, brew formula (f54b1ab)
- Test suite: transport seam, mutation guard, golden and teatest layers (cd206cc)

## M1: Pages CNAME subdomain-takeover audit

A new `gh-sweep pages` command that cross-checks GitHub Pages custom domains against live DNS, in both directions.

Scope:

- For each repo, query the Pages API (`GET /repos/{o}/{r}/pages` returns `cname`, verification state, and HTTPS enforcement) and read the repo's `CNAME` file
- Resolve DNS for each custom domain (CNAME/A/ALIAS records) and flag: domains that no longer resolve to `<user>.github.io` (dangling), Pages disabled while DNS still points at GitHub (takeover risk, since anyone can claim the subdomain on their own Pages site), and unverified domains
- Reverse check: given a list of DNS-configured subdomains, verify each has a live Pages site backing it
- Needs a domain list input: a `pages_domains` config key first, a zone file or DNS-provider API later

Starting points: `internal/github/client.go` (transport seam), `internal/cli/watching.go` (simplest command shape to copy), a new `internal/dns` package around `net.Resolver`.

## M2: Terminal plotting for gha-perf

Port the duration and trend charts the deleted Python `gha_perf.py` prototype rendered with plotext: per-workflow duration over time, branch comparison bars, and regression markers, inside the ghaperf TUI view.

Starting points: `internal/tui/components/ghaperf/model.go`, possibly [ntcharts](https://github.com/NimbleMarkets/ntcharts) for Bubble Tea-native plots.

## M3: Coverage to 70% and component refactor

Raise total coverage to the 70% bar the `test:coverage-min` task already defines, and remove the `.golangci.toml` legacy exclusion block (30 linters suppressed across all of `internal/`, marked `TODO(v1)`).

Scope:

- `internal/cli` (~26%, 57 functions): test the `--list` render paths through fake transports
- `internal/tui/components/branches` (~12%): state-transition tests for selection, delete flow, and error states
- Wire `test:coverage-min` and `test:safety` into the `ci` task once green
- Refactor components as needed so the excluded linters pass, then delete the exclusion rule

## M4: One GHA analytics path and live unused-secrets detection

`internal/github/actions.go` and `gha_perf.go` model the same `actions/runs` payload twice (`WorkflowRun` vs `RunTiming`, `WorkflowRunStats` vs `WorkflowStats`) with split consumers (analytics vs ghaperf/CLI). Merge them onto one model. Separately, `DetectUnusedSecrets` and its helpers in `internal/github/secrets.go` have no production callers, so the secrets view's Unused tab always renders empty; wire workflow scanning into the secrets component.

Starting points: `internal/github/actions.go`, `internal/github/gha_perf.go`, `internal/github/secrets.go`, `internal/tui/components/secrets/model.go`.

## M5: Demo GIF and generated usage docs

Record a demo with VHS (`.github/assets/demo.tape`, following the gh-repo-dashboard pattern) and embed it at the top of the README. If a fixture DSL for scripted TUI sessions is adopted, generate `docs/USAGE.md` from the same fixtures so the docs cannot go stale.

## Deferred

Low priority; pick up when convenient.

- Stacked-PR creation from selected branches (dependency detection via merge-base, PR chains with linked descriptions); parent-resolution heuristics are the open question
- Protection apply/template: template-based rule application with preview and per-repo reporting, export/import; template format and merge-vs-replace semantics undecided
- Linear and mani integrations: the packages were removed because both are large surface areas better served by their own tools until a concrete need returns
- SQLite cache with TTL and ETags for offline comment browsing; the JSON gha-perf cache covers the current need
- Real-time watching of runs or events; watchgha owns live monitoring
- Branch extras: multi-select by ranges ("1-10", "all"), tree visualization of branch hierarchy, pairwise comparison matrix
- Comment extras: surrounding-code context preview, open in browser, mark resolved from the TUI, heuristic resolution detection ("done"/"fixed" replies, merged PR with comment absent from the final diff)
- Settings sync: change preview, selective and bulk apply, rollback from stored prior state, git-backed shareable templates
- Analytics extras: AI-vs-human review ratios, contributor and bus-factor metrics, merge-behavior stats, review-delay percentiles, activity heatmap, CSV/JSON/markdown export
- Release extras: version grouping across repos, semver compliance flags, aggregated release-notes export
- Read-only dependency visibility (which repos have Renovate or Dependabot, version comparison)
- Other local repo tools (ghq, myrepos, gita, meta) behind a unified read-only interface

## Parked

Captured from earlier planning; the TUI views exist but are read-only, and the actionable halves stay parked until wanted:

- Webhook debugging: the webhooks view lists hooks and health; ping testing, redelivery, bulk enable/disable, duplicate detection, and delivery-log export are not built
- Time-boxed collaborator grants: the collaborators view shows access by repo and user; expiring grants, auto-revoke, bulk on/offboarding, and access-pattern cloning are not built
- Secrets compliance extras: the secrets view inventories org and repo secrets; naming-convention checks, cross-repo sharing flags, and audit export are not built
- CI cost estimation (Actions minutes, cost drivers by repo and workflow) and queue-time vs run-time splits on top of the gha-perf data
