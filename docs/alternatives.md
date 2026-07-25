# When to Use Automation Instead of gh-sweep

The README covers TUI peers (gh-dash, watchgha, gh-poi). This page covers the automation categories gh-sweep deliberately stays out of. The rule: if a task recurs on a schedule, automate it; if it needs human judgment, use gh-sweep for interactive exploration, cross-repo visibility, and one-off bulk operations.

## Dependency updates

Use [Renovate](https://github.com/renovatebot/renovate) for multi-repo setups, grouping, and scheduling, or [Dependabot](https://github.com/dependabot) for simple GitHub-native updates. gh-sweep does not touch dependencies.

## Repository settings as code

Use [Pulumi](https://www.pulumi.com/registry/packages/github/) or [Terraform](https://registry.terraform.io/providers/integrations/github/) to declare settings and protection rules. gh-sweep detects drift from a baseline repo and suits interactive exploration before you write the IaC.

## Stale issue and PR cleanup

Use [actions/stale](https://github.com/actions/stale) for scheduled labeling and closing. gh-sweep's orphan sweep is one-time, previewed cleanup of branches (not issues), driven by merged-PR and staleness detection.

## Flaky test detection

Use [BuildPulse](https://buildpulse.io/) or [Trunk](https://docs.trunk.io/flaky-tests/) for ML-based detection and quarantine, or [get-flakes](https://github.com/treebeardtech/get-flakes) for a free option. gh-sweep's gha-perf command sticks to duration statistics and regression flagging.

## Release automation

Use [semantic-release](https://github.com/semantic-release/semantic-release) for automated version bumps and changelogs, or [release-it](https://github.com/release-it/release-it) for interactive control. gh-sweep's releases view only reads release state across repos.

## Audit logging

Use the native [GitHub Audit Log](https://docs.github.com/en/organizations/keeping-your-organization-secure/managing-security-settings-for-your-organization/reviewing-the-audit-log-for-your-organization) for retention and compliance. gh-sweep suits quick interactive queries within GitHub's retention window.
