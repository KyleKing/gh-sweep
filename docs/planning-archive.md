# Planning Archive

This is a condensed archive of the original phased plan (`.phases/`, phases 1 through 5 plus the anti-phases and implementation plan). It keeps the unique, still-relevant ideas as input for a future ROADMAP.md rewrite. Items the implementation plan marked complete (flaky test detection, error log extraction, workflow debugging, settings templates and drift detection, secrets workflow scanning and usage tracking, Linear PR-issue linking and sync status) are omitted or trimmed to their unbuilt remainder.

## Feature ideas

### Branch management
- Category-level batch actions (delete, skip, review) with safe confirmations and progress tracking
- Multi-select input supporting comma-separated IDs, ranges like "1-10", and "all"
- Tree visualization of branch hierarchy with color-coded ahead/behind counts
- Pairwise comparison matrix for up to 10 selected branches
- Stacked PR workflow: detect dependencies via merge-base, sort by distance from default branch, find the closest parent within the selection, create PR chains in order with linked descriptions
- Open question: how to resolve ambiguous parents (merge-base vs commit-date heuristics, multiple candidate parents)

### PR comments review
- List unresolved comments across repos with GitHub Search DSL filters (repo, author, date range, PR state) plus custom filters (comment author, PR number, participants, fuzzy text search)
- Preview comment context with surrounding code, open in browser, mark resolved from the TUI
- SQLite cache with configurable TTL (about 1 hour), ETag storage, and offline browsing
- Heuristic resolution detection: explicit resolve, conclusive replies ("done", "fixed"), or merged PR with the comment absent from the latest diff

### Branch protection rules
- View and visually diff protection rules across repos, detect org-wide inconsistencies
- Template-based rule application with preview, confirmation, and per-repo success reporting
- Export and import rule configurations
- Open question: template format (YAML vs JSON), merge vs replace semantics for partial application

### Actions and CI analytics
- Run history dashboard filtered by status, workflow, branch, and date range, with duration trends and regression highlighting (over 20 percent slower than average)
- Cost estimation: Actions minutes consumed, cost with paid runners, top cost drivers by repo and workflow
- CI runs per file or path: identify hot paths that trigger frequent rebuilds to tune workflow triggers
- Queue time vs run time split to spot runner bottlenecks
- Most expensive workflows by duration times frequency
- Flaky test threshold tuning is open (10 percent failure rate and more than 2 flips was the starting point) as is parsing varied test output formats (JUnit, pytest, go test)

### Settings, webhooks, access, and releases
- Interactive settings sync with change preview, selective apply, bulk apply, and rollback from stored prior state
- Git-backed, versioned, shareable settings templates covering merge strategies, branch protection, Actions permissions, and security settings
- Org-wide webhook inventory grouped by target URL, duplicate detection, delivery success rates, failure pattern detection, ping testing, redelivery, bulk enable or disable, and export of delivery logs
- Time-boxed collaborator grants stored in SQLite with expiration reminders, optional auto-revoke, bulk onboarding and offboarding, cloning one user's access pattern to another, and an exportable access audit log
- Secrets compliance extras: naming convention regex checks, flagging secrets shared across many repos, and inventory export for audits
- Multi-repo release dashboard: latest release per repo, version grouping (all repos on v1.x vs v2.x), repos with no release in 90 days, semver compliance flags, aggregated release notes export, and tag listing with convention comparison

### Watching and orphans
- Both shipped (watch-status enforcement and orphan branch sweep exist as commands); remaining ideas from the plan are covered under branch management above

### Integrations
- Linear: workflow automation insights (which Linear states map to which GitHub events), broken automation detection (issue stuck in progress with a merged PR), PRs missing issue links as policy violations, and cross-repo issue aggregation by project or cycle
- mani: import `mani.yaml` as the repo source, export a generated `mani.yaml` from org repos, run mani tasks from the TUI, and aggregate `git status` output across repos
- Other local tools (ghq, myrepos, gita, meta): auto-detect which is in use, adapt through a unified read-only interface, and never write their configs
- Open questions: precedence when multiple tools are detected, Linear multi-workspace support, and Linear API rate limits

### Analytics
- AI review metrics: count bot reviews (Copilot, Renovate, Dependabot, CodeRabbit, user-defined patterns), AI vs human review ratio, and PRs merged with only AI review flagged as risk
- Comment and activity metrics: comments per repo, PR, and user, per-PR activity (commits, reviews, participants, time to first response, time to merge), and PRs with no activity in N days
- Delay to first review with median, p90, and p95 per repo, plus slowest PRs
- Contributor metrics: active contributor counts over time, first-time vs returning contributors, and bus factor analysis (files touched by only one person)
- Merge behavior: merge vs squash vs rebase counts, PR size distribution, review coverage (PRs with 0, 1, or 2 plus reviews), and approval-to-merge timing
- Activity heatmap (date by repository, intensity by PR or comment volume)
- Export all metrics to CSV, JSON, and Markdown for external BI tools

### Dependency visibility (read-only)
- Show which repos have Renovate or Dependabot configured and compare dependency versions across repos

## Anti-goals

- No automated dependency updates, vulnerability patching, or license enforcement; use Renovate (or Dependabot for simple setups). gh-sweep only visualizes configuration and versions
- No declarative repo configuration or drift auto-correction; use Pulumi or Terraform. gh-sweep detects drift, makes one-off changes, and could export current settings as IaC templates
- No scheduled stale issue or PR automation; use actions/stale. gh-sweep does previewed one-time cleanup
- No ML-based flaky test detection, quarantine, or root cause analysis; use BuildPulse, Trunk, or get-flakes. gh-sweep keeps the same-commit flip heuristic and error extraction
- No audit log retention, SIEM integration, or compliance reporting; use the native GitHub Audit Log. gh-sweep does quick interactive queries within GitHub's retention window
- No automated version bumping, changelog generation, or release creation; use semantic-release or release-it. gh-sweep views and compares releases
- No real-time CI monitoring or live run streaming; use watchgha. gh-sweep does historical analysis only

The positioning rule: if it can be automated, use automation; if it needs human judgment, use gh-sweep (interactive exploration, cross-repo visibility, one-off bulk operations, debugging).

## Alternatives guidance

docs/alternatives.md already covers the per-category tool comparisons and stays as is. Points from the plan not captured there:

- The niche map against TUI peers: gh-dash is a PR and issue dashboard, gh-poi is single-repo PR and issue focused, gh-enhance is CLI enhancements, watchgha is real-time CI. gh-sweep's distinct ground is cross-repo management, branch operations, protection rules, and settings sync
- Prior art worth revisiting when building comment review: pr-comments-cli (basic Python CLI) and the GitHub PR Comment Tracker browser extension
- Prior art for protection rules: gh-branch-rules (CLI only, not interactive)
- Additional flaky test options beyond the doc: Deflaker (multi-platform) and AWS stale-issue-cleanup for label-driven stale handling
