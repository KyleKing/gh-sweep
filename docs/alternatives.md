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

gh-sweep's `policy` command declares repo settings, security & analysis,
release immutability, and a branch-protection baseline in a YAML file, diffs
it against live repos, and applies drift. That overlaps with what
[Pulumi](https://www.pulumi.com/registry/packages/github/) and
[Terraform](https://registry.terraform.io/providers/integrations/github/) do,
but the two are built for different scales.

Terraform and Pulumi own state: every managed resource lives in a state file
that has to stay in sync with reality, which is exactly what an org needs
when it has to express per-team, per-environment variation and prove who
changed what for an audit. `policy` has no state file. It reads live GitHub on
every run and diffs it against the YAML you wrote, so there is nothing to go
stale, drift-detect, or import. That trade only pays off at the scale
`policy` targets: one person keeping a personal or small-team set of repos
consistent, where the settings really are meant to be the same everywhere and
the state-file overhead buys nothing.

Reach for Terraform or Pulumi once you need per-environment configuration, a
change history independent of git blame on a YAML file, or org-wide policy
enforcement with approval workflows. Reach for `policy` when you want a repo
to look the way you declared it, fast, without adopting an IaC pipeline to
get there.

### Stale issue and PR cleanup

Use [actions/stale](https://github.com/actions/stale) for scheduled labeling and closing. gh-sweep's orphan sweep is one-time, previewed cleanup of branches (not issues), driven by merged-PR and staleness detection.

### Flaky test detection

Use [BuildPulse](https://buildpulse.io/) or [Trunk](https://docs.trunk.io/flaky-tests/) for ML-based detection and quarantine, or [get-flakes](https://github.com/treebeardtech/get-flakes) for a free option. gh-sweep's gha-perf command sticks to duration statistics and regression flagging.

### Release automation

Use [semantic-release](https://github.com/semantic-release/semantic-release) for automated version bumps and changelogs, or [release-it](https://github.com/release-it/release-it) for interactive control. gh-sweep's releases view only reads release state across repos.

### Audit logging

Use the native [GitHub Audit Log](https://docs.github.com/en/organizations/keeping-your-organization-secure/managing-security-settings-for-your-organization/reviewing-the-audit-log-for-your-organization) for retention and compliance. gh-sweep suits quick interactive queries within GitHub's retention window.
