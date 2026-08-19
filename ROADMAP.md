# Roadmap

Phased, shippable milestones for gh-sweep. Each milestone stands on its own and can be released independently; the roadmap can stop at any point without leaving the app half-migrated.

Architecture and domain context live in [DESIGN.md](DESIGN.md); Go and workflow conventions live in [AGENTS.md](AGENTS.md).

## Vision

A cross-repo sweep and audit tool: point it at an org or a repo list and find what needs attention (dead and orphaned branches, unresolved review threads, protection and settings drift, slow or flaky workflows, risky configuration) then act on it interactively with previews and confirmations. Automation tools own recurring tasks; gh-sweep owns the one-off passes that need human judgment (see [docs/alternatives.md](docs/alternatives.md)).

## Testing strategy

Four layers, detailed in [DESIGN.md](DESIGN.md#testing): table-driven unit tests as the base, golden snapshots behind a build tag for visual regression, teatest end-to-end runs against a fake GitHub transport, and a two-part safety net (runtime mutation guard plus a static scan) so no test can mutate real GitHub state. New features land with unit tests first; golden snapshots stay deliberately few to avoid brittle churn.

Shipped work is not tracked here. `CHANGELOG.md` and `git log` are the record.

## M0: Guardrails on the destructive commands

The highest-priority milestone. `orphans --cleanup` is currently an unprompted,
unlimited delete, and all four gaps below are present as of v0.5.0.

- `runCleanup` in `internal/cli/orphans.go` deletes every classified branch
  immediately. `--dry-run` previews, but `orphans --cleanup` without it deletes
  with no prompt, no count summary, and no limit. With `--org` and `--namespace`
  both omitted the namespace resolves to the authenticated user and the scan
  covers every non-archived repo you own, so a bare `orphans --cleanup` is an
  account-wide branch delete. Add an interactive confirmation before the delete
  loop: print the count and the full list, then require a typed `yes`, with a
  `--yes`/`--force` flag to skip it for automation. The TUI orphans component
  already gates deletion behind a `y/N` screen, so the CLI is the asymmetric path
- `internal/orphans/detector.go:70` classifies a branch whose PR closed without
  merging as `OrphanTypeClosedPR` and returns it as deletable, which can erase
  abandoned-for-now work. Exclude that class from `--cleanup` by default behind
  an explicit opt-in flag, or at minimum list it separately in the confirmation
  so it is never deleted silently alongside merged branches
- `DefaultScanOptions` in `internal/orphans/types.go:94` sets
  `StaleDaysThreshold` to 7. A branch with no PR whose last commit is a week old
  is classified stale and deletable, which is short for personal repos. Either
  raise the default to 30 or make the confirmation non-negotiable for the stale
  class
- `watching --watch-all` has no confirmation and no dry-run, and the TUI `w`/`u`
  keys mutate subscriptions with no prompt. Lower stakes (notifications only) but
  the same pattern would make the tool consistent

Coverage on exactly this code is the thinnest in the repo, so M3 below and this
milestone reinforce each other. Test the orphan detector and the CLI cleanup path
first.

## Open questions from Kyle

From a desktop screenshot review, still unresolved:

1. Want to "open" the repo and make edits
1. Remove excess metadata (fork is useful, but star/follower count is not)
1. Are the watching categories right? There seems to be overlap
1. "Unwatched" is a misleading default tab name (Kyle prefers "All activity")
1. `Custom` notification setting isn't supported: GitHub's API can't see or set
   per-notification-type settings, so a repo configured that way (e.g. releases-only)
   shows as default instead

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
- Linear and mani integrations: the packages were removed because both are large surface areas better served by their own tools until a concrete need returns
- SQLite cache with TTL and ETags for offline comment browsing; the JSON gha-perf cache covers the current need
- Real-time watching of runs or events; watchgha owns live monitoring
- Branch extras: multi-select by ranges ("1-10", "all"), tree visualization of branch hierarchy, pairwise comparison matrix
- Comment extras: surrounding-code context preview, open in browser, mark resolved from the TUI, heuristic resolution detection ("done"/"fixed" replies, merged PR with comment absent from the final diff)
- Policy extras: a richer change-preview format than the current table/JSON/markdown diff output; git-backed shareable policy templates across users; org rulesets and custom-property coverage reporting alongside the per-repo diff
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
