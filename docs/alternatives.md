# When to Use Alternatives to gh-sweep

This guide helps you choose the right tool for your GitHub management needs. See [anti-phases.md](../.phases/anti-phases.md) for detailed comparisons.

## Quick Decision Matrix

| Task | Best Tool | When to Use gh-sweep Instead |
|------|-----------|------------------------------|
| **Automated dependency updates** | [Renovate](https://github.com/renovatebot/renovate) | Visualize dependency health, compare versions across repos |
| **Repository settings as code** | [Pulumi](https://www.pulumi.com/registry/packages/github/) or [Terraform](https://registry.terraform.io/providers/integrations/github/) | Detect drift, one-off changes, export current config |
| **Stale issue cleanup** | [actions/stale](https://github.com/actions/stale) | One-time bulk cleanup, preview before closing |
| **Flaky test detection** | [BuildPulse](https://buildpulse.io/), [Trunk](https://docs.trunk.io/flaky-tests/) | Simple statistics, error log extraction for AI |
| **Audit logging** | [GitHub Audit Log](https://docs.github.com/en/organizations/keeping-your-organization-secure/managing-security-settings-for-your-organization/reviewing-the-audit-log-for-your-organization) | Quick interactive queries, incident investigation |
| **Release automation** | [semantic-release](https://github.com/semantic-release/semantic-release) | View releases across repos, compare versions |
| **Real-time CI monitoring** | [watchgha](https://github.com/nedbat/watchgha) | Historical analysis, performance trends |
| **PR/Issue dashboard** | [gh-dash](https://github.com/dlvhdr/gh-dash) | Branch management, protection rules, comment search |

## Detailed Comparisons

### Dependency Management: Renovate vs Dependabot

**Use [Renovate](https://github.com/renovatebot/renovate) for:**
- Multi-repo or monorepo setups
- 30+ package managers (Go, npm, Python, Docker, etc.)
- Advanced grouping (e.g., all minor updates in one PR)
- Custom scheduling and automerge rules

**Example renovate.json:**
```json
{
  "extends": ["config:recommended"],
  "packageRules": [
    {
      "matchUpdateTypes": ["minor", "patch"],
      "groupName": "all non-major dependencies"
    }
  ],
  "schedule": ["before 5am on Monday"]
}
```

**Use [Dependabot](https://github.com/dependabot) for:**
- Simple repos with standard package managers
- GitHub-native integration (zero setup)
- Quick security updates

**Example .github/dependabot.yml:**
```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
```

**Use gh-sweep for:**
- ✅ Viewing which repos have Renovate/Dependabot configured
- ✅ Comparing dependency versions across repos
- ✅ Identifying outdated dependencies (read-only)

---

### Infrastructure as Code: Pulumi vs Terraform

**Use [Pulumi](https://www.pulumi.com/registry/packages/github/) for:**
- TypeScript/Python/Go preference
- Complex logic (conditionals, loops)
- Existing Pulumi infrastructure

**Example:**
```typescript
import * as github from "@pulumi/github";

const repos = ["repo1", "repo2", "repo3"];

repos.forEach(name => {
    const repo = new github.Repository(name, {
        name: name,
        visibility: "private",
        deleteBranchOnMerge: true,
    });

    new github.BranchProtection(`${name}-main`, {
        repositoryId: repo.nodeId,
        pattern: "main",
        requiredPullRequestReviews: [{
            requiredApprovingReviewCount: 2,
        }],
    });
});
```

**Use [Terraform](https://registry.terraform.io/providers/integrations/github/) for:**
- HCL preference
- Multi-cloud infrastructure
- Terraform ecosystem (state management, workspaces)

**Example:**
```hcl
resource "github_repository" "repos" {
  for_each = toset(["repo1", "repo2", "repo3"])

  name       = each.key
  visibility = "private"
  delete_branch_on_merge = true
}
```

**Use gh-sweep for:**
- ✅ Detecting drift from IaC-defined state
- ✅ Interactive exploration before writing IaC
- ✅ One-off emergency changes
- ✅ Exporting current config as Pulumi/Terraform templates

---

### Stale Issue/PR Automation

**Use [actions/stale](https://github.com/actions/stale) for:**
- Automated, scheduled cleanup
- Consistent labeling and closing
- Set-and-forget automation

**Example:**
```yaml
name: 'Close stale issues and PRs'
on:
  schedule:
    - cron: '0 0 * * *'

jobs:
  stale:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/stale@v9
        with:
          stale-issue-message: 'Stale. Closing in 7 days.'
          days-before-stale: 60
          days-before-close: 7
          exempt-issue-labels: 'work-in-progress'
```

**Use gh-sweep for:**
- ✅ One-time bulk cleanup
- ✅ Preview which issues/PRs would be closed
- ✅ Interactive review before closing

---

### Flaky Test Detection

**Use [BuildPulse](https://buildpulse.io/) for:**
- ML-based detection
- Root cause analysis
- Test quarantine
- CI optimization

**Use [Trunk Flaky Tests](https://docs.trunk.io/flaky-tests/) for:**
- Historical trending
- Workflow comparisons
- Automatic detection

**Use [get-flakes](https://github.com/treebeardtech/get-flakes) (open-source) for:**
- Simple detection
- Small teams
- Free solution

**Use gh-sweep for:**
- ✅ Simple heuristics (failed then passed on same commit)
- ✅ Extracting error logs for AI-assisted debugging
- ✅ Basic failure frequency statistics

---

### Release Automation

**Use [semantic-release](https://github.com/semantic-release/semantic-release) for:**
- Full automation
- Conventional commits
- Automated changelogs
- CI/CD integration

**Setup:**
```json
{
  "branches": ["main"],
  "plugins": [
    "@semantic-release/commit-analyzer",
    "@semantic-release/release-notes-generator",
    "@semantic-release/changelog",
    "@semantic-release/github"
  ]
}
```

**Use [release-it](https://github.com/release-it/release-it) for:**
- Interactive releases
- More manual control
- Custom workflows

**Use gh-sweep for:**
- ✅ Viewing releases across repos
- ✅ Comparing versions (semver compliance)
- ✅ Aggregating release notes
- ✅ Identifying outdated releases

---

### Real-Time CI Monitoring

**Use [watchgha](https://github.com/nedbat/watchgha) for:**
- Live workflow status
- Current branch monitoring
- Real-time updates

**Installation:**
```bash
pip install watchgha
watch_gha_runs
```

**Use gh-sweep for:**
- ✅ Historical analysis (past runs, not live)
- ✅ Performance trend analysis
- ✅ Error log extraction
- ✅ Flaky test identification

---

## Ecosystem Integration

gh-sweep is designed to **complement** the GitHub ecosystem:

```
┌─────────────────────────────────┐
│     Automation & IaC            │
│  ─────────────────────────      │
│  Renovate, Dependabot           │
│  Pulumi, Terraform              │
│  semantic-release               │
│  actions/stale                  │
└─────────────────────────────────┘
           ↕ (Manages)
┌─────────────────────────────────┐
│     GitHub Repositories         │
└─────────────────────────────────┘
           ↕ (Explores)
┌─────────────────────────────────┐
│         gh-sweep TUI            │
│  ─────────────────────────      │
│  Interactive exploration        │
│  Drift detection                │
│  One-off bulk operations        │
│  Debugging & investigation      │
└─────────────────────────────────┘
```

**Best Practice:** Use automation tools for recurring tasks, use gh-sweep for exploration and one-off operations.

## When to Choose gh-sweep

Choose gh-sweep when you need:

1. **Interactive Workflows:**
   - Visualizing branch relationships before creating stacked PRs
   - Previewing changes before bulk operations
   - Exploring unresolved PR comments with filtering

2. **Cross-Repo Visibility:**
   - Comparing settings across 50+ repositories
   - Identifying inconsistencies in branch protection
   - Viewing release status across projects

3. **Human Judgment:**
   - Deciding which branches to delete (dependency analysis)
   - Reviewing which comments need attention
   - Investigating CI failures with context

4. **One-Off Operations:**
   - Emergency access grants for contractors
   - Bulk webhook debugging after outage
   - Temporary settings sync before migration

**Golden Rule:** If it can be automated, use automation. If it needs human judgment, use gh-sweep.

## Further Reading

- [Phase Documentation](../.phases/) - Detailed feature plans
- [Anti-Phases](../.phases/anti-phases.md) - What we explicitly don't build
- [Contributing](../CONTRIBUTING.md) - Development guidelines
