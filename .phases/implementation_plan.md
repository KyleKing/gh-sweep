# Incremental Implementation Plan

**Goal:** Enhance existing implementations with tests, debugging features, and functional-inspired design patterns.

## Principles

1. **Functional Composition**: Pure functions, immutability where possible, composition over inheritance
2. **Comprehensive Testing**: >80% coverage with unit + integration tests
3. **Debugging Features**: Rich error context, workflow analysis, AI-friendly output
4. **Best Practices**: Clear error handling, type safety, composable APIs

## Stage 1: GitHub Actions Analytics + Debugging

### Current State
- Basic workflow run fetching ✅
- Simple statistics calculation ✅
- Missing: flaky test detection, error log extraction, debugging features

### Enhancements
1. **Flaky Test Detection** (`internal/github/actions_flaky.go`)
   - Pure functions for pattern detection
   - Composable filters (by commit SHA, by time window)
   - Statistical analysis (failure rate, flip count)
   - Tests: various flaky patterns, edge cases

2. **Error Log Extraction** (`internal/github/actions_errors.go`)
   - Fetch job logs from API
   - Extract last N lines with context
   - Filter noise (timestamps, stack traces)
   - AI-friendly formatting (JSON, Markdown)
   - Tests: log parsing, filtering, formatting

3. **Workflow Debugging** (`internal/github/actions_debug.go`)
   - Performance regression detection (>20% slower)
   - Queue time analysis
   - Workflow dependency graph
   - Tests: regression detection, graph building

4. **Tests** (`internal/github/actions_test.go`, etc.)
   - TestAnalyzeWorkflowRuns
   - TestDetectFlakyTests
   - TestExtractErrorLogs
   - TestDetectRegressions

### Success Criteria
- [x] All functions are pure (no side effects except I/O)
- [x] Composable filters and analyzers
- [x] >80% test coverage
- [x] Error logs in AI-friendly format

## Stage 2: Cross-Repo Settings

### Current State
- Basic settings fetching ✅
- Simple diff calculation ✅
- Missing: template system, drift detection, bulk sync

### Enhancements
1. **Settings Template System** (`internal/github/settings_template.go`)
   - YAML-based templates
   - Validation before apply
   - Preview changes (dry-run mode)
   - Rollback capability
   - Tests: template parsing, validation, preview

2. **Drift Detection** (`internal/github/settings_drift.go`)
   - Severity classification (critical/warning/info)
   - Policy-based rules
   - Composable drift analyzers
   - Tests: severity classification, policy evaluation

3. **Functional Composition** (refactor `settings.go`)
   - Pure comparison functions
   - Composable diff builders
   - Pipeline pattern for settings transformations
   - Tests: composition, transformations

4. **Tests** (`internal/github/settings_test.go`)
   - TestCompareSettings
   - TestTemplateValidation
   - TestDriftDetection
   - TestApplyTemplate (dry-run)

### Success Criteria
- [x] Template system with validation
- [x] Drift detection with policies
- [x] Composable diff builders
- [x] >80% test coverage

## Stage 3: Secrets Audit

### Current State
- Basic secret listing ✅
- Simple unused detection ✅
- Missing: workflow scanning, usage tracking, compliance checks

### Enhancements
1. **Workflow Scanner** (`internal/github/secrets_scanner.go`)
   - Parse workflow YAML files
   - Extract secret references (secrets.*, vars.*)
   - Build usage map
   - Tests: YAML parsing, reference extraction

2. **Usage Tracking** (`internal/github/secrets_usage.go`)
   - Cross-reference secrets with workflows
   - Detect unused secrets
   - Identify secrets used in multiple workflows
   - Tests: usage analysis, multi-workflow tracking

3. **Compliance Checks** (`internal/github/secrets_compliance.go`)
   - Check naming conventions
   - Detect old secrets (>90 days)
   - Identify secrets with broad scope
   - Tests: compliance rules, reporting

4. **Tests** (`internal/github/secrets_test.go`)
   - TestListSecrets
   - TestScanWorkflows
   - TestDetectUnused
   - TestComplianceChecks

### Success Criteria
- [x] Workflow YAML parsing
- [x] Usage tracking across workflows
- [x] Compliance checks
- [x] >80% test coverage

## Stage 4: Linear Integration

### Current State
- Basic GraphQL client ✅
- Issue fetching ✅
- Missing: PR linking, sync status, workflow insights

### Enhancements
1. **PR-Issue Linking** (`internal/integrations/linear/linking.go`)
   - Regex extraction from PR bodies
   - Multiple pattern support (Fixes/Closes/Resolves)
   - Parse PR description for Linear IDs
   - Tests: regex patterns, extraction

2. **Sync Status Detection** (`internal/integrations/linear/sync.go`)
   - Compare PR and issue states
   - Detect drift (PR merged but issue not done)
   - Composable sync rules
   - Tests: sync detection, various states

3. **Workflow Insights** (`internal/integrations/linear/workflow.go`)
   - Analyze Linear→GitHub automation
   - Detect broken workflows
   - Visualize state transitions
   - Tests: workflow analysis, transition detection

4. **Tests** (`internal/integrations/linear/client_test.go`, etc.)
   - TestGetIssue (mocked)
   - TestExtractIssueIDs
   - TestSyncStatus
   - TestWorkflowAnalysis

### Success Criteria
- [x] PR-issue linking with regex
- [x] Sync status detection
- [x] Workflow insights
- [x] >80% test coverage (with mocks)

## Stage 5: README Update

### Related Tools to Document
1. **gh-poi** - GitHub PRs/Issues TUI
2. **gh-enhance** - Enhanced GitHub CLI
3. **gh-dash** - GitHub dashboard
4. **watchgha** - Real-time Actions monitoring

### Comparison Matrix
```markdown
| Feature | gh-sweep | gh-dash | watchgha | gh-poi | gh-enhance |
|---------|----------|---------|----------|--------|------------|
| Branch Management | ✅ | ❌ | ❌ | ❌ | ❌ |
| Protection Rules | ✅ | ❌ | ❌ | ❌ | ❌ |
| Actions Analytics | ✅ | ❌ | ✅ Real-time | ❌ | ❌ |
| Settings Sync | ✅ | ❌ | ❌ | ❌ | ❌ |
| PR/Issue View | ✅ | ✅ | ❌ | ✅ | ✅ |
| Multi-repo | ✅ | ✅ | ✅ | ❌ | ❌ |
```

### Niche Positioning
- **gh-sweep**: Cross-repo management + settings sync
- **gh-dash**: PR/Issue dashboard
- **watchgha**: Real-time CI monitoring
- **gh-poi**: Single-repo PR/Issue focus
- **gh-enhance**: CLI enhancements

## Stage 6: Critical Review & Refactoring

### Code Quality Checklist
- [ ] All functions <50 lines
- [ ] No global mutable state
- [ ] Error wrapping with context
- [ ] Consistent naming conventions
- [ ] No code duplication
- [ ] Composable functions
- [ ] Pure functions where possible
- [ ] Clear separation of concerns

### Refactoring Targets
1. **Error Handling**: Consistent error wrapping with context
2. **Composability**: Extract reusable components
3. **Type Safety**: Add type aliases for clarity
4. **Documentation**: GoDoc for all exports
5. **Performance**: Identify bottlenecks, add caching

## Testing Strategy

### Unit Tests
- Pure function logic
- Data transformations
- Business rules
- Edge cases

### Integration Tests
- API interactions (mocked)
- File I/O
- External services

### Table-Driven Tests
```go
func TestFlakyDetection(t *testing.T) {
    tests := []struct {
        name     string
        runs     []WorkflowRun
        expected []FlakyTest
    }{
        {
            name: "same commit flip",
            runs: []WorkflowRun{
                {Conclusion: "failure", HeadSHA: "abc"},
                {Conclusion: "success", HeadSHA: "abc"},
            },
            expected: []FlakyTest{{Name: "test", Flaky: true}},
        },
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := DetectFlakyTests(tt.runs)
            // assertions
        })
    }
}
```

### Mocking Strategy
- HTTP responses for GitHub API
- GraphQL responses for Linear
- File system for workflow YAML

## Implementation Order

1. ✅ **Stage 1.1**: Flaky test detection + tests
2. ✅ **Stage 1.2**: Error log extraction + tests
3. ✅ **Stage 1.3**: Workflow debugging + tests
4. ✅ **Stage 2.1**: Settings templates + tests
5. ✅ **Stage 2.2**: Drift detection + tests
6. ✅ **Stage 3.1**: Workflow scanner + tests
7. ✅ **Stage 3.2**: Usage tracking + tests
8. ✅ **Stage 4.1**: PR-issue linking + tests
9. ✅ **Stage 4.2**: Sync status + tests
10. ✅ **Stage 5**: README update
11. ✅ **Stage 6**: Critical review + refactoring

## Commit Strategy

- Atomic commits per feature
- Clear commit messages
- Tests in same commit as implementation
- One stage = one commit (or more if large)
