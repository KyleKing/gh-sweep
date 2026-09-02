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

## Needs your input

- GHA Performance view load time: 13-32+ seconds even on a full cache hit,
  found while scripting the demo recording and dropped from that recording
  rather than investigated. Worth profiling before the next person hits it
  live
- **Sharing code with aragonite/gh-repo-dashboard beyond the rendering packages.** Full
  investigation (access-strategy comparison, what's shared, why the
  PR/workflow-run models did not move, and a feature-level synergy survey
  between the two tools) lives in
  [aragonite's docs/gh-sweep.md](https://github.com/KyleKing/aragonite/blob/main/docs/gh-sweep.md).
  The transport-seam/mutation-guard extraction from that doc has landed:
  `internal/github/transport.go` now wraps aragonite's `transport` package,
  `go.mod` pins a released aragonite commit, a gitignored `go.work`
  overrides it for local dev against the sibling checkout, and
  `mise run verify-released` (already wired into `hk`'s `pre-push` hook)
  proves the pinned version builds with `GOWORK=off` before a push.
  Everything else in that doc waits for a concrete want or a third
  consumer. The `verify-released`/`go.work` pattern this repo hand-applied
  is now in my_go_template v0.12.1, so a `copier update` reconciles the
  hand-applied copy with the template's

## Pending, waiting on a decision

Built and verified, not yet run against coverbasedev. Neither has been applied.

- **Branch prune.** 56 branches across 6 repos (30 docs, 18 irm, 3 watch-doggo,
  5 irm stale), previewed with `policy --list`. Run with
  `policy --policy <file> --apply --prune`. Eight of the 56 are 0-5 days old,
  including three merged the same day, which is the declared rule working
- **Ruleset apply.** Four repos in the coverbase `rules.yaml`. Do watch-doggo
  first: it is the only one whose live ruleset carries bypass actors
  (OrganizationAdmin, RepositoryRole 5), so it is the one apply that exercises
  the `Unmanaged`/`BypassActors` round-trip against real data. Verify both
  actors survived before doing the other three

Two defaults chosen without an explicit answer, both toward the safer side and
both a one-line change to reverse:

- `--apply` reports branch drift and deletes nothing; deletion needs `--prune`.
  Every other domain changes settings the API can change back, whereas this one
  removes refs, so it must not ride along with a bare `--apply --yes` in a
  scheduled job
- Closed-PR branches now delete by default (`--exclude-closed-pr` opts out),
  reversing the previous opt-in. Justified by GitHub restoring a branch from
  its PR on request, but it is a behavior change for anyone else on gh-sweep

## Backport to aragonite

Rate limiting belongs in the shared library, not here. gh-sweep is the second
consumer of the transport seam, and the third would rebuild the same thing.

aragonite v0.11.0 already has half of it: `forge/github.Budgets` reads what is
left of each pool proactively, and the read is free because GitHub does not
charge a rate-limit request against the rate limit. It models core, graphql,
and search as separate allowances, which is the right shape.

What gh-sweep has that aragonite does not, and should move up:

- The reactive half. `internal/github/ratelimit.go` wraps the transport, turns
  an exhausted-quota response into a typed `RateLimitError` naming the local
  reset time, and refuses later requests to that pool until the window opens so
  a retry loop cannot spend a request per attempt relearning the same thing. It
  honors `Retry-After` for secondary limits, and passes a permissions 403
  through unchanged because that response carries no remaining-count header
- Per-pool blocking, keyed off the request path (`/graphql`, `/search/`, else
  core) rather than one global window. This matches the separation `Budget`
  already documents: a tool out of core can still make GraphQL calls
- Token resolution. Both repos call `api.NewRESTClient`/`NewGraphQLClient` with
  empty options to pick up gh CLI auth with a `GITHUB_TOKEN` fallback. That
  resolution, and the decision to prefer it over a hand-rolled token flag,
  should be stated once in aragonite rather than rediscovered per repo

Once it lands there, delete `internal/github/ratelimit.go` and pin the release.
Combining the two halves is the actual win: `Budgets` before a burst to decide
whether to start, the transport to fail usefully when a burst runs out anyway.

The pairing matters for this repo specifically, because `DomainBranches` and the
orphans scan both cost one request per branch and exhausted the 5000-request
core pool twice in one session.

## Deferred

Low priority; pick up when convenient.

- Generated usage docs: if a fixture DSL for scripted TUI sessions is
  adopted, generate `docs/USAGE.md` from the same fixtures so the docs
  cannot go stale
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
- `--domains` to scope a policy apply to named domains (`--apply --domains rulesets`), useful for rolling one domain out a repo at a time. `--prune` already gates the only destructive domain, so this is convenience rather than safety
- Read-only org dump (`gh sweep dump --org <org> --format json`, one blob per repo covering settings, protection, rulesets, security toggles, and default branch) so an agent or a script can read a whole org's configuration in one call instead of looping five `gh api` endpoints. Preferred over an MCP server: no long-running process, and the output is diffable
- Cheaper branch dating: `DomainBranches` and the orphans scan both cost one request per branch, because GitHub's branches endpoint omits commit dates. A GraphQL query returning refs with target commit dates would collapse that to one request per repo, and would move the cost onto the separate GraphQL pool
- Pre-flight budget check: with aragonite's `Budgets`, warn before a scan whose branch count exceeds what the core pool has left, rather than failing partway through having spent it
- Watch status beyond repos the viewer owns: the GraphQL query widens to `ORGANIZATION_MEMBER` only when an org is named, so a bare `gh sweep watching` still reports personal repos only. Querying the org directly would cover repos the viewer is not a member of but can see
- Analytics extras: AI-vs-human review ratios, contributor and bus-factor metrics, merge-behavior stats, review-delay percentiles, activity heatmap, CSV/JSON/markdown export
- Release extras: version grouping across repos, semver compliance flags, aggregated release-notes export
- Command grouping: `cobra.Group` is unused, so `--help` lists every subcommand in one flat alphabetical block while the home menu already groups the same views into Namespace Audit, Single Repo, Cross-Repo, and Policy. Mirroring the menu in `--help` is where a CLI conventionally signals which commands converge declared state (`policy`) and which answer a one-off question (`gha-perf`, `analytics`, `comments`)
- Read-only dependency visibility (which repos have Renovate or Dependabot, version comparison)
- Other local repo tools (ghq, myrepos, gita, meta) behind a unified read-only interface

## Parked

Captured from earlier planning; the TUI views exist but are read-only, and the actionable halves stay parked until wanted:

- Webhook debugging: the webhooks view lists hooks and health; ping testing, redelivery, bulk enable/disable, duplicate detection, and delivery-log export are not built
- Time-boxed collaborator grants: the collaborators view shows access by repo and user; expiring grants, auto-revoke, bulk on/offboarding, and access-pattern cloning are not built
- Secrets compliance extras: the secrets view inventories org and repo secrets; naming-convention checks, cross-repo sharing flags, and audit export are not built
- CI cost estimation (Actions minutes, cost drivers by repo and workflow) and queue-time vs run-time splits on top of the gha-perf data
