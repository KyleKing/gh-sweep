# Phase 1: MVP - Core Features

**Status:** Planned
**Goal:** Establish foundational TUI with three core features that fill critical gaps in GitHub management tooling.

## Features

### 1.1 Interactive Branch Management
Port the Python Rich CLI interactive branch management from [dotfiles PR#5](https://github.com/KyleKing/dotfiles/pull/5) to Go TUI.

**Capabilities:**
- Category-level actions: batch delete, skip, or review individual branches
- Per-branch operations: view details, delete branches, create pull requests, skip
- Safe confirmations for all destructive operations
- Commit analysis with ahead/behind counts
- Batch management with progress tracking

**Advanced Graph Features:**
- Multi-select interface (comma-separated IDs, ranges like "1-10", or "all")
- Tree visualization showing hierarchical branch relationships
- Color-coded ahead/behind metrics in tree view
- Comparison matrix for pairwise analysis (up to 10 branches)
- Dependency analysis: identify closest parent branches

**Stacked PR Workflow:**
- Intelligent ordering of dependent feature branches
- Automatic dependency detection by comparing branches
- Sort by distance from default branch
- Find closest parent branches within selection
- Create PR chains in optimal sequence

### 1.2 Branch Protection Rules Management
Cross-repo branch protection rule management with interactive TUI.

**Capabilities:**
- View branch protection rules across multiple repositories
- Visual comparison of protection settings
- Apply/modify rules across repos interactively
- Template-based rule application
- Detect inconsistencies across organization
- Export/import rule configurations

**Gap Analysis:**
- Existing tools: [gh-branch-rules](https://github.com/katiem0/gh-branch-rules) (CLI only, not interactive)
- Our advantage: Interactive TUI with visual diff and bulk operations

### 1.3 Unresolved PR Comments Review
Interactive TUI for reviewing and managing unresolved GitHub PR comments with advanced filtering.

**Capabilities:**
- List all unresolved comments across repositories
- Preview comment context with surrounding code
- Filter by GitHub Search DSL:
  - Repository
  - PR author
  - Date range
  - State (open/closed/merged)
- Custom comment-specific filters:
  - Comment author
  - PR number
  - Created/updated timestamp
  - Participants
  - Plain text fuzzy/similarity search
- Navigate to comment in browser
- Mark as resolved from TUI
- Cache results for offline browsing

**Caching Strategy:**
- Use SQLite for local storage
- Cache TTL: 1 hour (configurable)
- Invalidate on explicit refresh
- Store comment metadata + GitHub API ETags

**Gap Analysis:**
- Existing tools: [pr-comments-cli](https://github.com/pragmatic-ai-org/pr-comments-cli) (basic Python CLI)
- Browser extension: [GitHub PR Comment Tracker](https://chromewebstore.google.com/detail/github-pr-comment-tracker/acmoflfoahncbbhbgibefmfklcpajleo)
- Our advantage: Terminal-native, advanced filtering, caching

## Architecture

### Project Structure
```
gh-sweep/
├── cmd/
│   ├── root.go           # Main CLI entry (Cobra)
│   ├── branches.go       # Branch management commands
│   ├── protection.go     # Branch protection commands
│   └── comments.go       # Comment review commands
├── internal/
│   ├── tui/
│   │   ├── model.go      # Main Bubble Tea model
│   │   ├── update.go     # Update logic
│   │   ├── view.go       # Rendering
│   │   └── components/
│   │       ├── branches/  # Branch management UI
│   │       │   ├── tree.go
│   │       │   ├── list.go
│   │       │   └── actions.go
│   │       ├── protection/ # Protection rules UI
│   │       │   ├── diff.go
│   │       │   ├── editor.go
│   │       │   └── templates.go
│   │       └── comments/  # Comments review UI
│   │           ├── list.go
│   │           ├── filter.go
│   │           └── preview.go
│   ├── github/           # GitHub API client
│   │   ├── client.go
│   │   ├── branches.go
│   │   ├── protection.go
│   │   └── comments.go
│   ├── cache/            # Caching layer
│   │   ├── sqlite.go
│   │   └── manager.go
│   ├── config/           # Configuration
│   │   └── config.go
│   └── git/              # Local git operations
│       └── local.go
├── pkg/
│   └── models/           # Shared data models
├── .config/
│   └── mise.toml         # Task definitions
├── .github/workflows/
│   └── ci.yaml           # CI/CD
├── .golangci.yml         # Linter config
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

### Tech Stack
- **TUI:** bubbletea + bubbles (components) + lipgloss (styling)
- **CLI:** cobra (command framework)
- **GitHub API:** go-gh (official GitHub CLI library)
- **Cache:** modernc.org/sqlite (pure Go SQLite)
- **Testing:** testify + teatest (bubbletea testing)
- **Linting:** golangci-lint
- **Formatting:** gofumpt + goimports
- **Task Runner:** mise
- **CI:** GitHub Actions

## Implementation Logic

### 1.1 Branch Management Logic

```go
// Pseudocode for branch operations

type BranchNode struct {
    Name           string
    Ahead          int
    Behind         int
    LastCommitDate time.Time
    Parent         *BranchNode
    Children       []*BranchNode
}

// Build dependency tree
func BuildBranchTree(branches []Branch) *BranchNode {
    // 1. Fetch merge-base for each branch pair
    // 2. Calculate ahead/behind counts
    // 3. Determine parent-child relationships
    // 4. Build tree structure
}

// Create stacked PRs
func CreateStackedPRs(selectedBranches []Branch) error {
    // 1. Sort branches by distance from default branch
    // 2. For each branch, find closest parent in selection
    // 3. Create PRs in order: child -> parent
    // 4. Link PRs in description
}
```

### 1.2 Branch Protection Logic

```go
// Pseudocode for protection rules

type ProtectionRule struct {
    Repository               string
    Branch                   string
    RequiredReviews          int
    RequireCodeOwnerReviews  bool
    RequireStatusChecks      []string
    EnforceAdmins            bool
    RequireLinearHistory     bool
    AllowForcePushes         bool
    AllowDeletions           bool
}

// Compare rules across repos
func CompareProtectionRules(repos []string, branch string) ([]ProtectionRule, []Difference) {
    // 1. Fetch rules for each repo
    // 2. Identify differences
    // 3. Return unified view + diff
}

// Apply rule template
func ApplyRuleTemplate(template ProtectionRule, repos []string) error {
    // 1. Preview changes
    // 2. Confirm with user
    // 3. Apply via GitHub API
    // 4. Report success/failures
}
```

### 1.3 Unresolved Comments Logic

```go
// Pseudocode for comment filtering

type Comment struct {
    ID           int
    Repository   string
    PRNumber     int
    Author       string
    Body         string
    Path         string
    Line         int
    CreatedAt    time.Time
    UpdatedAt    time.Time
    InReplyToID  *int
    Resolved     bool
}

// Fetch unresolved comments
func FetchUnresolvedComments(repos []string, filters Filters) ([]Comment, error) {
    // 1. Check cache first
    // 2. Query GitHub API for each repo
    // 3. Filter by GitHub Search DSL
    // 4. Apply custom filters (author, fuzzy search, etc.)
    // 5. Cache results
    // 6. Return filtered list
}

// Determine if comment is resolved
func IsResolved(comment Comment, allComments []Comment) bool {
    // GitHub's concept: a thread is resolved if marked or conversation concluded
    // 1. Check if explicitly marked as resolved
    // 2. Check if thread has conclusive reply (e.g., "done", "fixed")
    // 3. Check if PR is merged and comment not in latest diff
}
```

## Open Questions

1. **Branch Dependency Detection:**
   - How to handle ambiguous parent-child relationships?
   - Should we use merge-base or commit date heuristics?
   - What if a branch has multiple potential parents?

2. **Branch Protection Templates:**
   - Should we support custom template formats (YAML/JSON)?
   - How to handle partial template application (merge vs replace)?
   - Should templates be shareable across organizations?

3. **Comment Resolution:**
   - How to determine "resolved" state when not explicitly marked?
   - Should we use heuristics (e.g., "fixed", "done" keywords)?
   - How to handle outdated comments on force-pushed branches?

4. **Caching Strategy:**
   - Should cache be per-user or per-machine?
   - How to handle cache invalidation on concurrent access?
   - Should we cache GitHub API rate limit info?

5. **Multi-Org Support:**
   - Should we support multiple GitHub organizations in one config?
   - How to handle different credentials per org?

## Test Cases

### 1.1 Branch Management Tests

**Unit Tests:**
- `TestBuildBranchTree`: Verify correct parent-child relationships
- `TestCalculateAheadBehind`: Validate commit counting logic
- `TestSortByDistance`: Ensure correct ordering for stacked PRs
- `TestBatchDelete`: Verify safe deletion with confirmations

**Integration Tests:**
- `TestCreateStackedPRs`: End-to-end PR creation flow
- `TestBranchTreeVisualization`: Verify tree rendering accuracy

**TUI Tests (using teatest):**
- `TestBranchSelection`: Multi-select with ranges ("1-10", "all")
- `TestConfirmationPrompts`: Ensure destructive actions require confirmation

### 1.2 Branch Protection Tests

**Unit Tests:**
- `TestCompareProtectionRules`: Verify diff logic
- `TestApplyRuleTemplate`: Validate API call structure
- `TestProtectionRuleValidation`: Ensure rule constraints

**Integration Tests:**
- `TestFetchProtectionRules`: Real API calls (mocked)
- `TestBulkRuleApplication`: Apply rules to multiple repos

**TUI Tests:**
- `TestProtectionDiffView`: Verify visual diff rendering
- `TestRuleEditing`: Interactive rule modification

### 1.3 Unresolved Comments Tests

**Unit Tests:**
- `TestCommentFiltering`: Verify filter logic (DSL + custom)
- `TestCacheHitMiss`: Validate caching behavior
- `TestFuzzySearch`: Comment text similarity matching
- `TestResolvedDetection`: Heuristic-based resolution detection

**Integration Tests:**
- `TestFetchCommentsMultiRepo`: Query multiple repos
- `TestCacheInvalidation`: Ensure stale data is refreshed

**TUI Tests:**
- `TestCommentPreview`: Code context rendering
- `TestFilterInteraction`: Dynamic filter application

## Success Criteria

- [ ] All three features have working TUIs
- [ ] GitHub API integration functional with rate limit handling
- [ ] SQLite caching operational with TTL
- [ ] Test coverage >80% for core logic
- [ ] CI/CD pipeline green (tests, lint, build)
- [ ] README with installation and usage instructions
- [ ] Demo GIF/video for each feature

## Dependencies

**Required for MVP:**
- GitHub API access (personal access token or OAuth)
- Local git installation (for branch operations)
- Terminal with 256-color support (for lipgloss rendering)

**Optional:**
- Delta (for better diff visualization)
- GitHub CLI (gh) installed (fallback for some operations)

## Performance Targets

- Branch tree visualization: <2s for 100 branches
- Protection rule comparison: <5s for 50 repos
- Comment search: <10s for 1000 comments (cached: <1s)
- TUI responsiveness: <100ms input latency

## Related Documentation

- See `anti-phases.md` for features explicitly NOT in scope
- See `docs/alternatives.md` for when to use other tools
- See Phase 2 for next features (GitHub Actions, settings comparison)
