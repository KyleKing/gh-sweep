# Phase 2: GitHub Actions & Cross-Repo Settings

**Status:** Planned
**Dependencies:** Phase 1 complete
**Goal:** Add GitHub Actions metadata extraction and cross-repository settings management.

## Features

### 2.1 GitHub Actions Metadata Dashboard

**Niche:** While [watchgha](https://github.com/nedbat/watchgha) does real-time monitoring and commercial tools ([BuildPulse](https://buildpulse.io/), [Trunk](https://docs.trunk.io/flaky-tests/)) provide ML-based analytics, there's a gap for:
- Historical analysis with simple statistics
- AI-friendly error log extraction (last N lines of failures)
- Performance regression detection (comparing run times)
- Flaky test identification (basic pattern matching)

**Capabilities:**
- **Run History Dashboard:**
  - View workflow runs across multiple repos
  - Filter by status, workflow name, branch, date range
  - Display duration trends (chart or table)
  - Highlight performance regressions (>20% slower than average)

- **Flaky Test Detection:**
  - Identify tests that pass/fail inconsistently
  - Track failure rate by test name
  - Show first failure date and frequency
  - Simple heuristic: test failed then passed without code changes

- **Error Log Extraction:**
  - Extract last N lines (default: 100) of failed jobs
  - Format for AI consumption (JSON, Markdown)
  - Include context: workflow name, job name, step, timestamp
  - Filter noise (e.g., remove timestamps, stack traces)

- **Metadata Analytics:**
  - Jobs per repository (daily, weekly, monthly)
  - Success/failure rates
  - Average run duration by workflow
  - Most expensive workflows (by duration × frequency)
  - Queue time analysis

**Implementation Notes:**
- Use GitHub Actions API: `/repos/{owner}/{repo}/actions/runs`
- Cache workflow run data (24h TTL)
- Store in SQLite: `workflow_runs` table with indexed timestamps
- For flaky tests: parse test output (requires standardized format like JUnit XML)

### 2.2 Cross-Repo Settings Comparison

**Gap:** Existing tools like [github-settings-sync](https://danklco.com/posts/2025-06-easily-configure-github-repositories/) are CLI-only and non-interactive.

**Capabilities:**
- **Visual Settings Diff:**
  - Compare settings across repositories:
    - Branch protection rules
    - Webhook configurations
    - GitHub Actions permissions
    - Security settings (Dependabot, code scanning)
    - Merge strategies (squash, rebase, merge)
    - Default branch
    - Visibility (public/private)
  - Side-by-side comparison view
  - Highlight differences in red/green

- **Drift Detection:**
  - Define "baseline" repository or template
  - Identify repos that deviate from baseline
  - Categorize drift severity (critical, warning, info)
  - Example critical drift: branch protection disabled on main

- **Interactive Sync:**
  - Preview changes before applying
  - Select which settings to sync
  - Apply to single repo or bulk
  - Rollback capability (store previous state)

- **Template Management:**
  - Save common configurations as templates
  - Apply templates to new repositories
  - Version templates (git-backed YAML files)
  - Share templates across team

**Settings Coverage:**
```yaml
# Example settings template
repository:
  default_branch: main
  allow_merge_commit: false
  allow_squash_merge: true
  allow_rebase_merge: true
  delete_branch_on_merge: true

branch_protection:
  main:
    required_reviews: 2
    require_code_owner_reviews: true
    dismiss_stale_reviews: true
    require_status_checks:
      - "ci/test"
      - "ci/lint"
    enforce_admins: false

actions:
  enabled: true
  allowed_actions: "selected"
  selected_actions:
    - "actions/*"

security:
  dependabot_alerts: true
  dependabot_security_updates: true
  secret_scanning: true
  secret_scanning_push_protection: true
```

### 2.3 Webhook Overview & Debugging

**Gap:** [GitHub's UI](https://docs.github.com/en/webhooks/testing-and-troubleshooting-webhooks/viewing-webhook-deliveries) shows deliveries but lacks org-wide overview.

**Capabilities:**
- **Org-Wide Webhook Inventory:**
  - List all webhooks across repositories
  - Group by URL/target
  - Show event types per webhook
  - Identify duplicate webhooks (same URL, different repos)

- **Delivery Status Dashboard:**
  - Recent deliveries (last 72 hours per GitHub API limit)
  - Success/failure rates per webhook
  - Failure pattern detection (consistent timeouts, 4xx/5xx errors)
  - Alert on high failure rates (>10%)

- **Interactive Management:**
  - Test webhook (send ping event)
  - View delivery payload + response
  - Redeliver failed webhooks
  - Enable/disable webhooks
  - Bulk operations (disable all webhooks for a URL)

- **Debugging Helpers:**
  - Show recent error messages
  - Highlight SSL verification issues
  - Detect common problems (unreachable URL, timeout)
  - Export delivery logs for external analysis

**Implementation:**
- Use `/repos/{owner}/{repo}/hooks` API
- Use `/repos/{owner}/{repo}/hooks/{hook_id}/deliveries` for history
- Cache webhook configs (1h TTL)
- Delivery status: real-time (no caching)

## Architecture Changes

### New Packages
```
internal/
├── github/
│   ├── actions.go         # Actions API client
│   ├── settings.go        # Repository settings API
│   └── webhooks.go        # Webhooks API
├── tui/components/
│   ├── actions/           # Actions dashboard UI
│   │   ├── runs.go
│   │   ├── flaky.go
│   │   └── errors.go
│   ├── settings/          # Settings comparison UI
│   │   ├── diff.go
│   │   ├── template.go
│   │   └── sync.go
│   └── webhooks/          # Webhook management UI
│       ├── list.go
│       ├── deliveries.go
│       └── debug.go
├── cache/
│   ├── actions.go         # Actions-specific caching
│   └── settings.go        # Settings-specific caching
└── analytics/
    ├── flaky.go           # Flaky test detection
    ├── trends.go          # Performance trend analysis
    └── errors.go          # Error log extraction
```

## Implementation Logic

### 2.1 Flaky Test Detection

```go
type TestResult struct {
    Name       string
    Status     string  // pass, fail, skip
    Duration   time.Duration
    CommitSHA  string
    Timestamp  time.Time
}

type FlakyTest struct {
    Name           string
    FailureRate    float64  // 0.0 - 1.0
    FirstFailure   time.Time
    FailureCount   int
    TotalRuns      int
    LastFlipDate   time.Time  // Last time status changed
}

func DetectFlakyTests(results []TestResult) []FlakyTest {
    // 1. Group results by test name
    // 2. Sort by timestamp
    // 3. Identify "flips" (fail -> pass or pass -> fail)
    // 4. Calculate failure rate
    // 5. Flag tests with >10% failure rate and >2 flips
}

// Heuristic: Flaky if test failed then passed without code changes
func IsFlakyPattern(results []TestResult) bool {
    for i := 1; i < len(results); i++ {
        prev, curr := results[i-1], results[i]
        if prev.Status == "fail" && curr.Status == "pass" {
            if prev.CommitSHA == curr.CommitSHA {
                return true  // Same commit, different result = flaky
            }
        }
    }
    return false
}
```

### 2.2 Settings Comparison

```go
type RepoSettings struct {
    Repository           string
    DefaultBranch        string
    AllowMergeCommit     bool
    AllowSquashMerge     bool
    AllowRebaseMerge     bool
    DeleteBranchOnMerge  bool
    // ... other settings
}

type SettingsDiff struct {
    Field      string
    Repos      map[string]interface{}  // repo -> value
    Baseline   interface{}             // expected value
    Severity   string                  // critical, warning, info
}

func CompareSettings(repos []string, baseline RepoSettings) []SettingsDiff {
    // 1. Fetch settings for each repo
    // 2. Compare each field to baseline
    // 3. Identify differences
    // 4. Categorize severity:
    //    - critical: security settings, branch protection
    //    - warning: merge strategies, default branch
    //    - info: description, homepage
}

func ApplySettingTemplate(template RepoSettings, repos []string, dryRun bool) ([]ChangePreview, error) {
    // 1. Fetch current settings
    // 2. Calculate diff
    // 3. If dryRun: return preview
    // 4. Else: apply changes via PATCH API
    // 5. Store rollback data
}
```

### 2.3 Webhook Debugging

```go
type WebhookDelivery struct {
    ID           int
    WebhookID    int
    Event        string
    Status       int  // HTTP status code
    Duration     time.Duration
    Timestamp    time.Time
    RequestBody  string
    ResponseBody string
    Error        string
}

func AnalyzeWebhookHealth(deliveries []WebhookDelivery) WebhookHealth {
    successCount := 0
    failureCount := 0
    timeouts := 0

    for _, d := range deliveries {
        if d.Status >= 200 && d.Status < 300 {
            successCount++
        } else {
            failureCount++
            if d.Duration > 10*time.Second {
                timeouts++
            }
        }
    }

    successRate := float64(successCount) / float64(len(deliveries))

    return WebhookHealth{
        SuccessRate:  successRate,
        TotalDeliveries: len(deliveries),
        Timeouts:     timeouts,
        Alert:        successRate < 0.9,  // Alert if <90% success
    }
}
```

## Open Questions

1. **Flaky Test Detection:**
   - How to parse different test output formats (JUnit, pytest, go test)?
   - Should we support custom regex patterns for test identification?
   - What threshold for "flaky" (10% failure rate? 5%)?

2. **Settings Sync:**
   - Should we validate settings before applying (e.g., required status checks exist)?
   - How to handle settings that can't be synced (e.g., repo name, visibility)?
   - Should we support partial template application?

3. **Error Log Extraction:**
   - How many lines constitute "useful context" (50? 100? 200)?
   - Should we support custom filters (e.g., exclude stack traces)?
   - What format is best for AI consumption (JSON, Markdown, plain text)?

4. **Rate Limiting:**
   - Actions API has lower rate limits than REST API
   - How to handle large orgs with 100+ repos?
   - Should we implement exponential backoff or queueing?

## Test Cases

### 2.1 Actions Metadata Tests

**Unit Tests:**
- `TestDetectFlakyTests`: Verify flaky test detection algorithm
- `TestExtractErrorLogs`: Validate log extraction and filtering
- `TestCalculateTrends`: Performance regression detection
- `TestParseTestResults`: Different test output formats

**Integration Tests:**
- `TestFetchWorkflowRuns`: Real API calls (mocked)
- `TestCacheActionRuns`: Validate caching behavior

**TUI Tests:**
- `TestActionsRunsView`: Render workflow runs table
- `TestFlakyTestsView`: Display flaky test list

### 2.2 Settings Comparison Tests

**Unit Tests:**
- `TestCompareSettings`: Diff calculation logic
- `TestApplyTemplate`: Template application with validation
- `TestDetectDrift`: Baseline comparison
- `TestRollback`: Restore previous settings

**Integration Tests:**
- `TestFetchRepoSettings`: API calls for multiple repos
- `TestBulkSettingsUpdate`: Apply changes to multiple repos

**TUI Tests:**
- `TestSettingsDiffView`: Visual diff rendering
- `TestTemplateEditor`: Interactive template editing

### 2.3 Webhook Tests

**Unit Tests:**
- `TestAnalyzeWebhookHealth`: Health metrics calculation
- `TestDetectFailurePatterns`: Pattern recognition
- `TestGroupWebhooks`: Group by URL logic

**Integration Tests:**
- `TestFetchWebhooks`: Org-wide webhook discovery
- `TestFetchDeliveries`: Recent delivery history

**TUI Tests:**
- `TestWebhookListView`: Webhook inventory display
- `TestDeliveryDebugView`: Payload/response viewer

## Success Criteria

- [ ] Actions dashboard shows runs with performance trends
- [ ] Flaky test detection identifies >90% of known flaky tests
- [ ] Error log extraction produces AI-friendly output
- [ ] Settings comparison works for 50+ repos in <10s
- [ ] Webhook health alerts trigger on high failure rates
- [ ] Test coverage >80%
- [ ] All features have demo videos

## Performance Targets

- Actions dashboard: <5s to load 30 days of history (100 repos)
- Settings comparison: <10s for 50 repos
- Webhook inventory: <3s for 100 webhooks
- Flaky test detection: <2s for 1000 test results

## Related Documentation

- See Phase 1 for foundational features
- See Phase 3 for access management and release features
- See `anti-phases.md` for features explicitly NOT in scope
