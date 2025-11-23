# Phase 5: Analytics & Statistics

**Status:** Planned
**Dependencies:** Phase 1, Phase 2 complete (Phase 3-4 optional)
**Goal:** Provide comprehensive analytics and statistics for repository activity, CI performance, and collaboration metrics.

## Features

### 5.1 CI/CD Performance Analytics

**Capabilities:**
- **CI Runs Per Repository:**
  - Count workflow runs per day/week/month
  - Group by repository
  - Show success/failure rates
  - Display as bar chart or table

- **CI Runs Per File/Path:**
  - Track which files trigger the most CI runs
  - Identify "hot paths" that cause frequent rebuilds
  - Useful for optimizing CI triggers (e.g., exclude docs from tests)
  - Example: "Changes to `src/api/*.go` triggered 50 runs this week"

- **Workflow Duration Trends:**
  - Average duration per workflow over time
  - Detect performance regressions (>20% slower)
  - Compare duration across branches (main vs feature)
  - Show slowest workflows

- **Queue Time Analysis:**
  - Time waiting for runner (queue time)
  - Time spent executing (run time)
  - Identify bottlenecks (long queue times = need more runners)

- **Cost Estimation:**
  - Estimate GitHub Actions minutes consumed
  - Calculate cost (if using paid runners)
  - Show top cost drivers (repo + workflow)

**Implementation:**
- Use Actions API: `/repos/{owner}/{repo}/actions/runs`
- Parse workflow run logs for file triggers (requires git diff comparison)
- Cache run data (7 days retention, configurable)
- Store in SQLite: `workflow_runs` table with indexed timestamps

### 5.2 AI Review Metrics

**Context:** Track usage of AI code review tools (GitHub Copilot PR summaries, third-party bots).

**Capabilities:**
- **AI Review Count:**
  - Count AI-generated PR comments/reviews
  - Identify which repos use AI reviews most
  - Track over time (weekly/monthly)

- **Detection Heuristics:**
  - GitHub Copilot: Look for `copilot` bot user
  - Other bots: Detect by username pattern (e.g., `*[bot]`)
  - Custom: Allow user-defined bot patterns

- **AI vs Human Review Ratio:**
  - Compare AI review count to human review count
  - Show as percentage: "30% of reviews are AI-generated"
  - Highlight repos over-reliant on AI (>50%)

- **Review Quality Proxy:**
  - Track PRs with only AI reviews (no human review)
  - Flag as potential risk
  - Show PRs merged without human review

**Implementation:**
- Use `/repos/{owner}/{repo}/pulls/{number}/reviews` API
- Filter reviews by user type: `Bot`
- Match username patterns: `copilot`, `renovate`, `dependabot`, etc.

### 5.3 Comment & Activity Analytics

**Capabilities:**
- **Number of Comments:**
  - Per repository: Total comments on PRs/issues
  - Per PR: Comment count (identify highly-discussed PRs)
  - Per user: Top commenters
  - Over time: Comment trends (increasing/decreasing)

- **Activity Per PR:**
  - Commits count
  - Comments count
  - Reviews count
  - Participants count
  - Time to first response
  - Time to merge

- **PRs Without Recent Activity:**
  - List PRs with no activity in last N days (default: 7)
  - Show days since last activity
  - Flag stale PRs for closure or follow-up

- **Delay to First Code Review:**
  - Time from PR opened to first review
  - Average per repository
  - Percentiles (p50, p90, p95)
  - Identify repos with slow review cycles

**Activity Heatmap:**
- Visualize PR activity over time
- X-axis: Date, Y-axis: Repository
- Color intensity: Number of PRs/comments

**Implementation:**
- Use PRs API + Comments API + Reviews API
- Calculate metrics from timestamps
- Cache PR metadata (1h TTL)

### 5.4 Contributor Analytics

**Capabilities:**
- **Active Contributors:**
  - Count unique contributors per repo
  - Track over time (weekly/monthly)
  - Identify growing/shrinking contributor base

- **Contributor Churn:**
  - Track first-time contributors
  - Identify returning vs one-time contributors
  - Show retention rate

- **Contribution Patterns:**
  - PR size distribution (small/medium/large)
  - Review participation (who reviews whom)
  - External vs internal contributors

- **Bus Factor Analysis:**
  - Identify knowledge silos (files touched by only 1 person)
  - Show distribution of file ownership
  - Recommend cross-training areas

**Implementation:**
- Use Commits API + PRs API
- Analyze authorship patterns
- Parse `git log --numstat` for file ownership

### 5.5 Merge Behavior Analytics

**Capabilities:**
- **Merge Strategies:**
  - Count merge commits vs squash vs rebase
  - Show per repository
  - Identify inconsistent patterns

- **PR Size Distribution:**
  - Small (<10 lines), Medium (10-100), Large (>100)
  - Encourage smaller PRs (show median size)

- **Review Coverage:**
  - Percentage of PRs with 0, 1, 2+ reviews
  - Flag PRs merged without review

- **Approval Patterns:**
  - Time from approval to merge
  - Number of approvals before merge
  - Re-approval after new commits

**Implementation:**
- Use PRs API: `merged_by`, `merge_commit_sha`
- Compare commit count: rebase (1 commit), merge (>1), squash (1 squashed)

## Architecture Changes

### New Packages
```
internal/
├── analytics/
│   ├── ci.go              # CI/CD analytics
│   ├── ai.go              # AI review metrics
│   ├── comments.go        # Comment analytics
│   ├── contributors.go    # Contributor analytics
│   ├── merge.go           # Merge behavior
│   └── aggregator.go      # Aggregate metrics
├── tui/components/
│   ├── analytics/
│   │   ├── dashboard.go   # Main analytics dashboard
│   │   ├── ci.go          # CI metrics view
│   │   ├── ai.go          # AI review view
│   │   ├── comments.go    # Comment stats view
│   │   └── charts.go      # Chart rendering (bar, line)
├── cache/
│   └── metrics.go         # Metrics caching
└── export/
    ├── csv.go             # Export to CSV
    ├── json.go            # Export to JSON
    └── markdown.go        # Export to Markdown
```

## Implementation Logic

### 5.1 CI Runs Per File

```go
type FileTrigger struct {
    Path         string
    TriggerCount int
    LastTriggered time.Time
}

func AnalyzeCITriggersPerFile(runs []WorkflowRun, repo string) ([]FileTrigger, error) {
    fileTriggers := make(map[string]*FileTrigger)

    for _, run := range runs {
        // Get files changed in the commit that triggered the run
        files, err := getChangedFiles(repo, run.HeadSHA)
        if err != nil {
            continue
        }

        for _, file := range files {
            if trigger, ok := fileTriggers[file]; ok {
                trigger.TriggerCount++
                if run.CreatedAt.After(trigger.LastTriggered) {
                    trigger.LastTriggered = run.CreatedAt
                }
            } else {
                fileTriggers[file] = &FileTrigger{
                    Path:          file,
                    TriggerCount:  1,
                    LastTriggered: run.CreatedAt,
                }
            }
        }
    }

    // Convert to slice and sort by count
    var result []FileTrigger
    for _, trigger := range fileTriggers {
        result = append(result, *trigger)
    }

    sort.Slice(result, func(i, j int) bool {
        return result[i].TriggerCount > result[j].TriggerCount
    })

    return result, nil
}
```

### 5.2 AI Review Detection

```go
type ReviewStats struct {
    Repository      string
    TotalReviews    int
    AIReviews       int
    HumanReviews    int
    AIPercentage    float64
    PRsOnlyAI       int
}

func AnalyzeAIReviews(prs []PullRequest, reviews []Review) ReviewStats {
    stats := ReviewStats{
        Repository: prs[0].Repository,
    }

    aiUsernames := []string{"copilot", "renovate", "dependabot"}
    prAIReviews := make(map[int]int)     // PR number -> AI review count
    prHumanReviews := make(map[int]int)  // PR number -> human review count

    for _, review := range reviews {
        stats.TotalReviews++

        isAI := false
        for _, bot := range aiUsernames {
            if strings.Contains(strings.ToLower(review.User), bot) || review.UserType == "Bot" {
                isAI = true
                break
            }
        }

        if isAI {
            stats.AIReviews++
            prAIReviews[review.PRNumber]++
        } else {
            stats.HumanReviews++
            prHumanReviews[review.PRNumber]++
        }
    }

    // Count PRs with only AI reviews
    for prNum, aiCount := range prAIReviews {
        humanCount := prHumanReviews[prNum]
        if aiCount > 0 && humanCount == 0 {
            stats.PRsOnlyAI++
        }
    }

    if stats.TotalReviews > 0 {
        stats.AIPercentage = float64(stats.AIReviews) / float64(stats.TotalReviews) * 100
    }

    return stats
}
```

### 5.3 Delay to First Review

```go
type ReviewDelayMetrics struct {
    Repository  string
    Median      time.Duration
    P90         time.Duration
    P95         time.Duration
    Average     time.Duration
    SlowestPRs  []PRDelay
}

type PRDelay struct {
    Number      int
    Title       string
    Delay       time.Duration
    FirstReviewer string
}

func CalculateReviewDelay(prs []PullRequest, reviews []Review) ReviewDelayMetrics {
    var delays []time.Duration
    prDelays := make(map[int]*PRDelay)

    // Group reviews by PR
    reviewsByPR := make(map[int][]Review)
    for _, review := range reviews {
        reviewsByPR[review.PRNumber] = append(reviewsByPR[review.PRNumber], review)
    }

    // Calculate delay for each PR
    for _, pr := range prs {
        prReviews := reviewsByPR[pr.Number]
        if len(prReviews) == 0 {
            continue  // No reviews yet
        }

        // Sort reviews by timestamp
        sort.Slice(prReviews, func(i, j int) bool {
            return prReviews[i].SubmittedAt.Before(prReviews[j].SubmittedAt)
        })

        firstReview := prReviews[0]
        delay := firstReview.SubmittedAt.Sub(pr.CreatedAt)

        delays = append(delays, delay)
        prDelays[pr.Number] = &PRDelay{
            Number:        pr.Number,
            Title:         pr.Title,
            Delay:         delay,
            FirstReviewer: firstReview.User,
        }
    }

    if len(delays) == 0 {
        return ReviewDelayMetrics{Repository: prs[0].Repository}
    }

    // Sort delays
    sort.Slice(delays, func(i, j int) bool {
        return delays[i] < delays[j]
    })

    // Calculate percentiles
    median := delays[len(delays)/2]
    p90 := delays[int(float64(len(delays))*0.9)]
    p95 := delays[int(float64(len(delays))*0.95)]

    // Calculate average
    var sum time.Duration
    for _, d := range delays {
        sum += d
    }
    average := sum / time.Duration(len(delays))

    // Get slowest 5 PRs
    var slowest []PRDelay
    count := 0
    for i := len(delays) - 1; i >= 0 && count < 5; i-- {
        for _, pd := range prDelays {
            if pd.Delay == delays[i] {
                slowest = append(slowest, *pd)
                count++
                break
            }
        }
    }

    return ReviewDelayMetrics{
        Repository: prs[0].Repository,
        Median:     median,
        P90:        p90,
        P95:        p95,
        Average:    average,
        SlowestPRs: slowest,
    }
}
```

## Open Questions

1. **CI Analytics:**
   - Should we support custom cost-per-minute for GitHub Actions?
   - How to track which files trigger CI without heavy git operations?
   - Should we cache git diffs or recompute on demand?

2. **AI Review Metrics:**
   - How to distinguish helpful AI reviews from spam?
   - Should we track AI review quality (e.g., reviews that led to changes)?
   - What other AI tools should we detect (CodeRabbit, etc.)?

3. **Activity Metrics:**
   - How far back should we analyze (30 days? 90 days? All time)?
   - Should metrics be exportable for external BI tools?
   - How to handle private repos (rate limits)?

4. **Visualization:**
   - Should we use ASCII charts (termgraph) or rendered images?
   - What chart types: bar, line, heatmap, scatter?

## Test Cases

### Unit Tests
- `TestAnalyzeCITriggersPerFile`: File trigger counting
- `TestAnalyzeAIReviews`: AI vs human review detection
- `TestCalculateReviewDelay`: Review delay percentiles
- `TestAnalyzeContributorChurn`: Contributor retention

### Integration Tests
- `TestFetchWorkflowRunsForAnalytics`: API calls
- `TestCacheMetrics`: Metrics caching

### TUI Tests
- `TestAnalyticsDashboard`: Dashboard rendering
- `TestChartRendering`: Chart display

## Success Criteria

- [ ] CI analytics show accurate run counts and duration trends
- [ ] AI review detection identifies >90% of bot reviews
- [ ] Review delay metrics calculated correctly (percentiles)
- [ ] Analytics exportable to CSV/JSON
- [ ] Dashboard responsive (<2s load time)
- [ ] Test coverage >80%
- [ ] Demo videos for analytics features

## Performance Targets

- Analytics dashboard: <5s to load 30 days of data (50 repos)
- CI runs analysis: <10s for 1000 workflow runs
- Review delay calculation: <3s for 500 PRs

## Export Formats

### CSV Example
```csv
Repository,CI Runs,Success Rate,Avg Duration,AI Reviews,Human Reviews
repo1,150,94%,5m30s,10,40
repo2,200,88%,8m15s,5,60
```

### JSON Example
```json
{
  "repository": "repo1",
  "ci_runs": {
    "total": 150,
    "success_rate": 0.94,
    "avg_duration_seconds": 330
  },
  "reviews": {
    "ai_count": 10,
    "human_count": 40,
    "ai_percentage": 20.0
  },
  "review_delay": {
    "median_hours": 4.5,
    "p90_hours": 24.0
  }
}
```

## Related Documentation

- See Phase 2 for Actions metadata (foundation for CI analytics)
- See Phase 1-4 for other features
- See `docs/analytics-guide.md` for interpretation guide
- See `anti-phases.md` for features explicitly NOT in scope
