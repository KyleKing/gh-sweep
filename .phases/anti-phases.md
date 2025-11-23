# Anti-Phases: What's NOT in Scope

This document clearly defines what `gh-sweep` will **NOT** do and recommends better alternatives for those use cases.

## Philosophy

`gh-sweep` is designed for **interactive exploration, visualization, and targeted operations** across GitHub repositories. It complements existing automation tools rather than replacing them.

Use `gh-sweep` for:
- ✅ Interactive TUI workflows
- ✅ Cross-repository visibility and comparison
- ✅ One-off bulk operations
- ✅ Exploration and debugging

Do NOT use `gh-sweep` for:
- ❌ Automated background tasks (use GitHub Actions)
- ❌ Infrastructure as Code (use Pulumi/Terraform)
- ❌ Full-featured automation (use specialized tools)
- ❌ Enterprise-grade monitoring (use commercial platforms)

---

## Anti-Phase 1: Dependency Management

### What We're NOT Building
- Automated dependency updates (like Renovate/Dependabot)
- Vulnerability scanning and patching
- Dependency graph visualization
- License compliance enforcement
- Version pinning automation

### Why Not
[Renovate](https://github.com/renovatebot/renovate) and [Dependabot](https://github.com/dependabot) are purpose-built for this and do it excellently.

### Use Instead

#### **Renovate** (Recommended for most teams)

**Why:** Superior multi-repo and monorepo support, 30+ package managers, intelligent grouping.

**Installation:**
```bash
# Install Renovate GitHub App
# Visit: https://github.com/apps/renovate
```

**Example `renovate.json`:**
```json
{
  "extends": ["config:recommended"],
  "packageRules": [
    {
      "matchUpdateTypes": ["minor", "patch"],
      "groupName": "all non-major dependencies",
      "groupSlug": "all-minor-patch"
    }
  ],
  "schedule": ["before 5am on Monday"]
}
```

**Resources:**
- [Renovate Docs](https://docs.renovatebot.com/)
- [Renovate vs Dependabot](https://docs.renovatebot.com/bot-comparison/)

#### **Dependabot** (Good for simple setups)

**Why:** Built into GitHub, zero setup for basic use cases.

**Example `.github/dependabot.yml`:**
```yaml
version: 2
updates:
  - package-ecosystem: "npm"
    directory: "/"
    schedule:
      interval: "weekly"
    reviewers:
      - "your-team"
```

**Resources:**
- [Dependabot Docs](https://docs.github.com/en/code-security/dependabot)

#### **What gh-sweep CAN do:**
- ✅ Visualize dependency health across repos (read-only)
- ✅ Show which repos have Renovate/Dependabot configured
- ✅ Compare dependency versions across repos

---

## Anti-Phase 2: Infrastructure as Code (IaC)

### What We're NOT Building
- Declarative repository configuration
- Terraform/Pulumi-style state management
- GitOps workflows for org settings
- Automated drift correction
- Configuration version control

### Why Not
IaC tools (Pulumi, Terraform) provide version control, rollback, and compliance features that are critical for org-wide settings.

### Use Instead

#### **Pulumi** (Recommended for TypeScript/Python users)

**Why:** Use real programming languages, native GitHub provider, excellent for teams already using Pulumi.

**Example:**
```typescript
import * as github from "@pulumi/github";

const repo = new github.Repository("my-repo", {
    name: "my-repo",
    visibility: "private",
    hasIssues: true,
    hasProjects: false,
    deleteBranchOnMerge: true,
});

const protection = new github.BranchProtection("main-protection", {
    repositoryId: repo.nodeId,
    pattern: "main",
    requiredPullRequestReviews: [{
        requiredApprovingReviewCount: 2,
        requireCodeOwnerReviews: true,
    }],
});
```

**Resources:**
- [Pulumi GitHub Provider](https://www.pulumi.com/registry/packages/github/)
- [Pulumi at GitHub blog](https://www.pulumi.com/blog/managing-github-with-pulumi/)

#### **Terraform** (Industry standard IaC)

**Why:** Declarative HCL, mature ecosystem, widely adopted.

**Example:**
```hcl
resource "github_repository" "my_repo" {
  name               = "my-repo"
  visibility         = "private"
  has_issues         = true
  has_projects       = false
  delete_branch_on_merge = true
}

resource "github_branch_protection" "main" {
  repository_id = github_repository.my_repo.node_id
  pattern       = "main"

  required_pull_request_reviews {
    required_approving_review_count = 2
    require_code_owner_reviews      = true
  }
}
```

**Resources:**
- [Terraform GitHub Provider](https://registry.terraform.io/providers/integrations/github/latest/docs)

#### **When to use what:**
- **Pulumi:** TypeScript/Python teams, complex logic, existing Pulumi usage
- **Terraform:** HCL preference, multi-cloud, Terraform ecosystem
- **gh-sweep:** Interactive exploration, drift detection, one-off changes

#### **What gh-sweep CAN do:**
- ✅ Compare settings across repos (detect drift from IaC)
- ✅ Preview IaC changes before applying
- ✅ Export current settings as Pulumi/Terraform templates

---

## Anti-Phase 3: Stale Issue/PR Automation

### What We're NOT Building
- Automated stale labeling
- Scheduled issue/PR closing
- Comment-based triggers
- Auto-labeling based on activity
- Stale bot configuration

### Why Not
This is a "set and forget" automation problem, best solved by GitHub Actions.

### Use Instead

#### **GitHub Actions: `actions/stale`** (Official)

**Why:** Well-maintained, configurable, runs automatically.

**Example `.github/workflows/stale.yml`:**
```yaml
name: 'Close stale issues and PRs'
on:
  schedule:
    - cron: '0 0 * * *'  # Daily

jobs:
  stale:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/stale@v9
        with:
          stale-issue-message: 'This issue is stale. Will close in 7 days.'
          stale-pr-message: 'This PR is stale. Will close in 7 days.'
          days-before-stale: 60
          days-before-close: 7
          exempt-issue-labels: 'work-in-progress,blocked'
          exempt-pr-labels: 'work-in-progress'
```

**Resources:**
- [actions/stale](https://github.com/marketplace/actions/process-stale-issues-and-prs)

#### **AWS Stale Issue Cleanup** (Advanced)

**Why:** Label-based triggering, more flexibility.

**Resources:**
- [aws-actions/stale-issue-cleanup](https://github.com/aws-actions/stale-issue-cleanup)

#### **What gh-sweep CAN do:**
- ✅ One-time bulk cleanup (preview before closing)
- ✅ Show which issues/PRs WOULD be marked stale
- ✅ Interactive review before closing

---

## Anti-Phase 4: Enterprise Flaky Test Detection

### What We're NOT Building
- ML-based flaky test detection
- Root cause analysis
- Test quarantine systems
- CI failure categorization (flake vs real failure)
- Historical flakiness trends

### Why Not
Commercial tools have invested heavily in ML models and extensive analytics that we can't match.

### Use Instead

#### **BuildPulse** (Best for CI optimization)

**Why:** Identifies flaky tests, provides quarantine, root cause analysis. Optimizes CI workloads.

**Pricing:** Paid service

**Resources:**
- [BuildPulse](https://buildpulse.io/)

#### **Trunk Flaky Tests** (Good CI analytics)

**Why:** Detects flakiness, historical views, workflow comparisons.

**Resources:**
- [Trunk Flaky Tests](https://docs.trunk.io/flaky-tests/)

#### **Deflaker** (Multi-platform)

**Why:** Works with GitHub Actions, GitLab, Jenkins.

**Resources:**
- [Deflaker](https://deflaker.com/)

#### **Open-Source: get-flakes** (Basic detection)

**Why:** Free, simple, good for small teams.

**Example:**
```bash
pip install get-flakes
get-flakes --repo owner/repo --days 7
```

**Resources:**
- [get-flakes](https://github.com/treebeardtech/get-flakes)

#### **What gh-sweep CAN do:**
- ✅ Simple heuristics (test failed then passed on same commit)
- ✅ Extract error logs for AI-assisted debugging
- ✅ Show failure frequency (basic statistics)

---

## Anti-Phase 5: Comprehensive Audit Logging

### What We're NOT Building
- Full audit log retention (beyond GitHub's 90/180 days)
- Compliance report generation
- SIEM integration
- Real-time security monitoring
- Audit log alerting

### Why Not
GitHub's native audit log + Actions provide enterprise-grade capabilities.

### Use Instead

#### **GitHub Audit Log** (Native)

**Why:** Built-in, comprehensive, GraphQL API access.

**Access:**
```bash
# View org audit log
gh api graphql -f query='
{
  organization(login: "your-org") {
    auditLog(first: 100) {
      edges {
        node {
          action
          actorLogin
          createdAt
        }
      }
    }
  }
}'
```

**Resources:**
- [Audit Log Docs](https://docs.github.com/en/organizations/keeping-your-organization-secure/managing-security-settings-for-your-organization/reviewing-the-audit-log-for-your-organization)

#### **org-audit-action** (Automated exports)

**Why:** Export audit logs to CSV/JSON for compliance.

**Resources:**
- [GitHub Actions for Security Compliance](https://github.blog/enterprise-software/github-actions-for-security-compliance/)

#### **What gh-sweep CAN do:**
- ✅ Quick interactive queries (last 90 days)
- ✅ Specific incident investigation
- ✅ Visual timeline of events

---

## Anti-Phase 6: Full Release Automation

### What We're NOT Building
- Automated version bumping
- Changelog generation
- Git tag creation based on commits
- Conventional commit parsing
- Automated GitHub release creation

### Why Not
Specialized tools handle conventional commits, semver, and changelog generation better.

### Use Instead

#### **semantic-release** (Full automation)

**Why:** Analyzes commits, determines version, generates changelog, creates release.

**Example `.releaserc.json`:**
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

**Resources:**
- [semantic-release](https://github.com/semantic-release/semantic-release)

#### **release-it** (Interactive releases)

**Why:** More manual control, interactive prompts.

**Resources:**
- [release-it](https://github.com/release-it/release-it)

#### **git-semver** (Simple tagging)

**Why:** Lightweight, just handles version tags.

**Resources:**
- [git-semver](https://github.com/mdomke/git-semver)

#### **What gh-sweep CAN do:**
- ✅ View releases across repos
- ✅ Compare versions
- ✅ Export aggregated release notes
- ✅ Validate semver compliance

---

## Anti-Phase 7: Real-Time CI Monitoring

### What We're NOT Building
- Live workflow run streaming
- Real-time failure notifications
- CI runner health monitoring
- Workflow queue visualization
- Build time predictions

### Why Not
[watchgha](https://github.com/nedbat/watchgha) does real-time monitoring excellently.

### Use Instead

#### **watchgha** (Real-time TUI)

**Why:** Live updates, branch-specific, lightweight.

**Installation:**
```bash
pip install watchgha
watch_gha_runs
```

**Resources:**
- [watchgha](https://github.com/nedbat/watchgha)

#### **What gh-sweep CAN do:**
- ✅ Historical analysis (past runs, not live)
- ✅ Performance trends
- ✅ Error log extraction

---

## Summary Table

| Feature | Use Instead | When to Use gh-sweep |
|---------|-------------|---------------------|
| **Dependency Updates** | Renovate, Dependabot | Visualize health, compare versions |
| **Repo Settings IaC** | Pulumi, Terraform | Detect drift, one-off changes, export templates |
| **Stale Automation** | GitHub Actions (actions/stale) | One-time cleanup, preview |
| **Flaky Test Detection** | BuildPulse, Trunk, Deflaker | Simple stats, error extraction |
| **Audit Logging** | GitHub Audit Log + Actions | Quick queries, incident investigation |
| **Release Automation** | semantic-release, release-it | View releases, compare versions |
| **Real-Time CI** | watchgha | Historical analysis, trends |

---

## Positioning gh-sweep

`gh-sweep` is designed to **complement** the GitHub ecosystem:

```
┌─────────────────────────────────────────┐
│         GitHub Ecosystem                │
├─────────────────────────────────────────┤
│  Automation: Renovate, Actions          │
│  IaC: Pulumi, Terraform                 │
│  Monitoring: BuildPulse, watchgha       │
└─────────────────────────────────────────┘
                   ↕
┌─────────────────────────────────────────┐
│           gh-sweep                      │
├─────────────────────────────────────────┤
│  Interactive TUI for:                   │
│  - Exploration & visualization          │
│  - One-off bulk operations              │
│  - Debugging & investigation            │
│  - Cross-repo comparison                │
└─────────────────────────────────────────┘
```

**Golden Rule:** If it can be automated, use automation. If it needs human judgment, use `gh-sweep`.
