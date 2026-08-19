# Roadmap

Phased, shippable milestones for gh-sweep. Each milestone stands on its own and can be released independently; the roadmap can stop at any point without leaving the app half-migrated.

Architecture and domain context live in [DESIGN.md](DESIGN.md); Go and workflow conventions live in [AGENTS.md](AGENTS.md).

## Vision

A cross-repo sweep and audit tool: point it at an org or a repo list and find what needs attention (dead and orphaned branches, unresolved review threads, protection and settings drift, slow or flaky workflows, risky configuration) then act on it interactively with previews and confirmations. Automation tools own recurring tasks; gh-sweep owns the one-off passes that need human judgment (see [docs/alternatives.md](docs/alternatives.md)).

## Testing strategy

Four layers, detailed in [DESIGN.md](DESIGN.md#testing): table-driven unit tests as the base, golden snapshots behind a build tag for visual regression, teatest end-to-end runs against a fake GitHub transport, and a two-part safety net (runtime mutation guard plus a static scan) so no test can mutate real GitHub state. New features land with unit tests first; golden snapshots stay deliberately few to avoid brittle churn.

Shipped work is not tracked here. `CHANGELOG.md` and `git log` are the record.

## M0: TUI watching toggle confirmation (deferred)

The TUI watching component's `w`/`u`/`i` keys mutate a single repo's
subscription with no prompt, unlike the CLI's `orphans --cleanup` and
`watching --watch-all`, which now gate behind a typed `yes`. Left as is: each
key press is a single reversible toggle (press again to undo), not a batch
operation, so the stakes don't match the CLI paths. Revisit only if it turns
out to bite in practice.

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

- Pages audit: detect a domain proxied through Cloudflare or another CDN (its
  A records fall outside GitHub's documented Pages ranges) and report it as
  unverifiable rather than a false "dangling" finding
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
