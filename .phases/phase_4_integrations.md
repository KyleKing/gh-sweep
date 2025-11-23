# Phase 4: Integrations (Linear, Local Git Tools)

**Status:** Planned
**Dependencies:** Phase 1, Phase 2, Phase 3 complete
**Goal:** Integrate with external project management (Linear) and local multi-repo tools (mani, ghq, etc.).

## Features

### 4.1 Linear Integration

**Resources:**
- [Linear API Docs](https://linear.app/docs/github-integration)
- [Official GitHub Integration](https://linear.app/integrations/github)

**Capabilities:**
- **Issue-PR Linking:**
  - View Linear issues linked to GitHub PRs
  - Show issue status (Todo, In Progress, Done)
  - Display assignee, project, cycle
  - Navigate to Linear issue from TUI

- **Workflow Automation Insights:**
  - Show which Linear states trigger on GitHub events
  - Example: "In Progress" → PR opened, "Done" → PR merged
  - Visualize workflow mappings per team
  - Detect broken automations (e.g., issue stuck in "In Progress" with merged PR)

- **Sync Status Dashboard:**
  - List PRs with/without Linear issue links
  - Highlight PRs missing issue links (policy violation)
  - Show last sync time between Linear and GitHub
  - Detect drift (e.g., PR closed but Linear issue still "In Progress")

- **Cross-Repo Issue Tracking:**
  - Aggregate Linear issues across multiple GitHub repos
  - Group by Linear project/cycle
  - Filter by team, assignee, status
  - Show issue counts per repository

**Implementation:**
- Use Linear GraphQL API: `https://api.linear.app/graphql`
- Requires Linear API key (user-provided)
- Extract issue IDs from PR descriptions (e.g., `Fixes LIN-123`)
- Cache Linear data (1h TTL)

**Example Query:**
```graphql
query IssueDetails {
  issue(id: "LIN-123") {
    id
    title
    state { name }
    assignee { name }
    project { name }
    cycle { name }
  }
}
```

### 4.2 mani Integration

**Resources:**
- [mani GitHub](https://github.com/alajmo/mani)
- [mani docs](https://manicli.com/)

**About mani:** CLI tool to manage multiple repositories with declarative YAML config (`mani.yaml`).

**Capabilities:**
- **Import mani Projects:**
  - Parse `mani.yaml` from current directory
  - Extract repository list and metadata
  - Use as source for gh-sweep operations
  - Example: "Run gh-sweep on all repos in mani.yaml"

- **Export to mani Config:**
  - Generate `mani.yaml` from GitHub org repos
  - Include repository URLs, paths, tags
  - Useful for bootstrapping mani config

- **Task Execution Bridge:**
  - Show mani tasks defined in `mani.yaml`
  - Execute mani tasks from gh-sweep TUI
  - Display task output in TUI
  - Filter repos for task execution

- **Status Aggregation:**
  - Combine `mani run git status` output
  - Show uncommitted changes across all repos
  - Highlight repos needing attention

**Example mani.yaml:**
```yaml
projects:
  - name: gh-sweep
    path: ~/code/gh-sweep
    url: https://github.com/KyleKing/gh-sweep
    tags: [go, tui]

  - name: yak-shears
    path: ~/code/yak-shears
    url: https://github.com/KyleKing/yak-shears
    tags: [python, cli]

tasks:
  status:
    cmd: git status -s
    desc: Show git status

  pull:
    cmd: git pull
    desc: Pull latest changes
```

### 4.3 Other Local Git Tool Integrations

**Alternatives to mani:**
- **[ghq](https://github.com/x-motemen/ghq)**: Repository organization by URL (`~/ghq/github.com/user/repo`)
- **[myrepos (mr)](https://myrepos.branchable.com/)**: VCS-agnostic multi-repo management
- **[gita](https://github.com/nosarthur/gita)**: Side-by-side status display
- **[meta](https://github.com/mateodelnorte/meta)**: Monorepo + multi-repo hybrid

**Proposed Integration Strategy:**
- **Auto-detect** which tool is in use:
  - Check for `mani.yaml`, `.mr/`, `ghq` installation
- **Unified interface:**
  - Abstract common operations (list repos, run command)
  - Tool-specific adapters for each
- **Read-only mode:**
  - Import repo lists from tools
  - Don't modify tool configs (user manages those)

**ghq Integration Example:**
```go
func ListGhqRepos() ([]Repository, error) {
    // Run: ghq list -p
    // Parse output (one path per line)
    // Convert to Repository structs
}
```

**myrepos Integration Example:**
```go
func ListMrRepos() ([]Repository, error) {
    // Parse ~/.mrconfig
    // Extract [path] sections
    // Convert to Repository structs
}
```

## Architecture Changes

### New Packages
```
internal/
├── integrations/
│   ├── linear/
│   │   ├── client.go      # Linear GraphQL client
│   │   ├── issues.go      # Issue queries
│   │   └── sync.go        # Sync detection
│   ├── mani/
│   │   ├── parser.go      # Parse mani.yaml
│   │   ├── executor.go    # Execute mani tasks
│   │   └── exporter.go    # Generate mani.yaml
│   └── localtools/
│       ├── detector.go    # Auto-detect tool
│       ├── ghq.go         # ghq integration
│       ├── mr.go          # myrepos integration
│       ├── gita.go        # gita integration
│       └── adapter.go     # Unified interface
├── tui/components/
│   ├── linear/
│   │   ├── issues.go
│   │   ├── sync.go
│   │   └── workflow.go
│   └── localtools/
│       ├── repos.go
│       └── tasks.go
└── github/
    └── linear.go          # PR-issue linking logic
```

## Implementation Logic

### 4.1 PR-Issue Linking

```go
type LinearIssue struct {
    ID          string
    Title       string
    State       string
    Assignee    string
    Project     string
    Cycle       string
    UpdatedAt   time.Time
}

type PRIssuePair struct {
    Repository  string
    PRNumber    int
    PRStatus    string  // open, merged, closed
    IssueID     string
    Issue       *LinearIssue
    InSync      bool
}

func ExtractLinearIssueIDs(prBody string) []string {
    // Regex to match: Fixes LIN-123, Closes LIN-456, etc.
    re := regexp.MustCompile(`(?:Fixes|Closes|Resolves)\s+(LIN-\d+)`)
    matches := re.FindAllStringSubmatch(prBody, -1)

    var issueIDs []string
    for _, match := range matches {
        if len(match) > 1 {
            issueIDs = append(issueIDs, match[1])
        }
    }

    return issueIDs
}

func CheckPRIssueSyncStatus(pair PRIssuePair) bool {
    // PR merged but issue not "Done"
    if pair.PRStatus == "merged" && pair.Issue.State != "Done" {
        return false
    }

    // PR closed but issue not "Canceled"
    if pair.PRStatus == "closed" && pair.Issue.State != "Canceled" {
        return false
    }

    // PR open but issue "Done"
    if pair.PRStatus == "open" && pair.Issue.State == "Done" {
        return false
    }

    return true
}
```

### 4.2 mani Config Parsing

```go
type ManiProject struct {
    Name   string   `yaml:"name"`
    Path   string   `yaml:"path"`
    URL    string   `yaml:"url"`
    Tags   []string `yaml:"tags"`
}

type ManiTask struct {
    Cmd  string `yaml:"cmd"`
    Desc string `yaml:"desc"`
}

type ManiConfig struct {
    Projects []ManiProject       `yaml:"projects"`
    Tasks    map[string]ManiTask `yaml:"tasks"`
}

func ParseManiYAML(path string) (*ManiConfig, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var config ManiConfig
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, err
    }

    return &config, nil
}

func ExecuteManiTask(config *ManiConfig, taskName string, projects []string) (map[string]string, error) {
    task, ok := config.Tasks[taskName]
    if !ok {
        return nil, fmt.Errorf("task %s not found", taskName)
    }

    results := make(map[string]string)

    for _, project := range config.Projects {
        // Filter projects
        if len(projects) > 0 && !contains(projects, project.Name) {
            continue
        }

        // Execute task in project directory
        cmd := exec.Command("sh", "-c", task.Cmd)
        cmd.Dir = project.Path

        output, err := cmd.CombinedOutput()
        if err != nil {
            results[project.Name] = fmt.Sprintf("ERROR: %s", err)
        } else {
            results[project.Name] = string(output)
        }
    }

    return results, nil
}
```

### 4.3 Tool Auto-Detection

```go
type LocalTool string

const (
    ToolMani    LocalTool = "mani"
    ToolGhq     LocalTool = "ghq"
    ToolMyrepos LocalTool = "myrepos"
    ToolGita    LocalTool = "gita"
    ToolNone    LocalTool = "none"
)

func DetectLocalTool() LocalTool {
    // Check for mani.yaml
    if _, err := os.Stat("mani.yaml"); err == nil {
        return ToolMani
    }

    // Check for ghq installation
    if _, err := exec.LookPath("ghq"); err == nil {
        return ToolGhq
    }

    // Check for .mrconfig
    home, _ := os.UserHomeDir()
    if _, err := os.Stat(filepath.Join(home, ".mrconfig")); err == nil {
        return ToolMyrepos
    }

    // Check for gita installation
    if _, err := exec.LookPath("gita"); err == nil {
        return ToolGita
    }

    return ToolNone
}

type ToolAdapter interface {
    ListRepos() ([]Repository, error)
    RunCommand(cmd string, repos []string) (map[string]string, error)
}

func GetAdapter(tool LocalTool) (ToolAdapter, error) {
    switch tool {
    case ToolMani:
        return &ManiAdapter{}, nil
    case ToolGhq:
        return &GhqAdapter{}, nil
    case ToolMyrepos:
        return &MrAdapter{}, nil
    case ToolGita:
        return &GitaAdapter{}, nil
    default:
        return nil, fmt.Errorf("no adapter for tool: %s", tool)
    }
}
```

## Open Questions

1. **Linear Integration:**
   - Should we support two-way sync (update Linear from gh-sweep)?
   - How to handle multiple Linear workspaces?
   - Should we cache Linear team/project metadata?

2. **mani Integration:**
   - Should we write to `mani.yaml` or just read?
   - How to handle conflicts if user edits `mani.yaml` externally?
   - Should we support mani themes/environment variables?

3. **Tool Prioritization:**
   - If multiple tools detected, which takes precedence?
   - Should user be able to override auto-detection?
   - How to handle repos managed by multiple tools?

4. **API Rate Limits:**
   - Linear API has rate limits (how to handle?)
   - Should we batch Linear queries (GraphQL supports batching)?

## Test Cases

### 4.1 Linear Integration Tests

**Unit Tests:**
- `TestExtractLinearIssueIDs`: Regex extraction from PR body
- `TestCheckPRIssueSyncStatus`: Sync detection logic
- `TestLinearGraphQLQuery`: Query building

**Integration Tests:**
- `TestFetchLinearIssue`: Real Linear API call (mocked)
- `TestPRIssueMapping`: Map PRs to Linear issues

**TUI Tests:**
- `TestLinearIssuesView`: Display Linear issues
- `TestSyncStatusView`: Show sync status

### 4.2 mani Integration Tests

**Unit Tests:**
- `TestParseManiYAML`: YAML parsing
- `TestExecuteManiTask`: Task execution logic
- `TestExportManiConfig`: Generate mani.yaml

**Integration Tests:**
- `TestManiTaskExecution`: Run real mani task

**TUI Tests:**
- `TestManiProjectsView`: Display mani projects
- `TestManiTasksView`: Show mani tasks

### 4.3 Tool Detection Tests

**Unit Tests:**
- `TestDetectLocalTool`: Tool auto-detection
- `TestGhqAdapter`: ghq integration
- `TestMrAdapter`: myrepos integration

## Success Criteria

- [ ] Linear issues linked to PRs display correctly
- [ ] Sync status detection identifies drift
- [ ] mani.yaml parsing and task execution works
- [ ] Auto-detection identifies tool correctly (>90%)
- [ ] At least 2 local tools supported (mani + ghq)
- [ ] Test coverage >75%
- [ ] Demo videos for integrations

## Performance Targets

- Linear API queries: <2s for 50 issues
- mani task execution: depends on task (no added overhead)
- Tool auto-detection: <100ms

## Related Documentation

- See Phase 1-3 for core features
- See Phase 5 for analytics features
- See `docs/integrations.md` for integration guides
- See `anti-phases.md` for features explicitly NOT in scope
