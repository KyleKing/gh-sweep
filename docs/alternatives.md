# Alternatives

Use gh-sweep for interactive exploration, cross-repo visibility, and one-off bulk
operations. Anything that recurs on a schedule belongs in automation instead.

## Other TUIs

[gh-dash](https://github.com/dlvhdr/gh-dash) is a personal PR and issue
dashboard: one view of the PRs and issues assigned to you across repos, with a
quick review workflow. Use it for daily triage of your own work, and gh-sweep for
repository administration across many repos.

[watchgha](https://github.com/nedbat/watchgha) tails GitHub Actions runs live
with streaming status updates. Use it while developing to watch CI in real time.
gh-sweep's `gha-perf` command covers the historical side: duration trends,
regressions, and flaky heuristics.

[gh-poi](https://github.com/seachicken/gh-poi) safely deletes local branches
whose PRs have merged, from a single checkout. Use it for local single-repo
hygiene. gh-sweep's `branches` and `orphans` commands work on remote branches
across many repos without a checkout.

## Automation gh-sweep stays out of

### Dependency updates

Use [Renovate](https://github.com/renovatebot/renovate) for multi-repo setups, grouping, and scheduling, or [Dependabot](https://github.com/dependabot) for simple GitHub-native updates. gh-sweep does not touch dependencies.

### Repository settings as code

Use [Pulumi](https://www.pulumi.com/registry/packages/github/) or [Terraform](https://registry.terraform.io/providers/integrations/github/) to declare settings and protection rules. gh-sweep detects drift from a baseline repo and suits interactive exploration before you write the IaC.

### Stale issue and PR cleanup

Use [actions/stale](https://github.com/actions/stale) for scheduled labeling and closing. gh-sweep's orphan sweep is one-time, previewed cleanup of branches (not issues), driven by merged-PR and staleness detection.

### Flaky test detection

Use [BuildPulse](https://buildpulse.io/) or [Trunk](https://docs.trunk.io/flaky-tests/) for ML-based detection and quarantine, or [get-flakes](https://github.com/treebeardtech/get-flakes) for a free option. gh-sweep's gha-perf command sticks to duration statistics and regression flagging.

### Release automation

Use [semantic-release](https://github.com/semantic-release/semantic-release) for automated version bumps and changelogs, or [release-it](https://github.com/release-it/release-it) for interactive control. gh-sweep's releases view only reads release state across repos.

### Audit logging

Use the native [GitHub Audit Log](https://docs.github.com/en/organizations/keeping-your-organization-secure/managing-security-settings-for-your-organization/reviewing-the-audit-log-for-your-organization) for retention and compliance. gh-sweep suits quick interactive queries within GitHub's retention window.
