# Phase 3: Access Management & Release Overview

**Status:** Planned
**Dependencies:** Phase 1, Phase 2 complete
**Goal:** Add temporary collaborator management, secrets audit, and release tracking.

## Features

### 3.1 Temporary Collaborator Management

**Use Case:** Managing contractors, interns, work trial candidates, or temporary team members.

**Capabilities:**
- **Time-Boxed Access Grants:**
  - Add collaborator with expiration date
  - Automatic reminder N days before expiration (via TUI notification)
  - Optional auto-revoke on expiration (requires periodic check)
  - Track access duration (granted date, expiration, days remaining)

- **Cross-Repo Access Review:**
  - Show all repositories a user has access to
  - Display permission level (read, write, admin)
  - Highlight orphaned access (user no longer in org but has repo access)
  - Filter by user, permission level, date range

- **Bulk Operations:**
  - Onboard contractor across multiple repos at once
  - Offboard (revoke access) across all repos
  - Change permission level (e.g., write -> read) bulk
  - Clone access pattern from one user to another

- **Access Audit Log:**
  - Who added whom, when, for what duration
  - Track permission changes
  - Export audit log (CSV, JSON)
  - Filter by date, user, repository

**Implementation:**
- Store access grants in SQLite: `collaborator_grants` table
- Fields: `user, repo, permission, granted_by, granted_at, expires_at, revoked_at`
- Periodic check (daily cron or manual): notify/revoke expiring access
- Use `/repos/{owner}/{repo}/collaborators` API

### 3.2 Secrets & Variables Audit (Read-Only)

**Goal:** Visibility into secrets/variables usage, NOT management (use Vault/etc. for that).

**Capabilities:**
- **Secrets Inventory:**
  - List all organization-level secrets
  - Show which repos have access to each secret
  - List repository-level secrets
  - Group by secret name (show org vs repo secrets)

- **Unused Secrets Detection:**
  - Parse workflow files (`.github/workflows/*.yml`)
  - Identify secrets defined but not referenced
  - Show last workflow run that used each secret
  - Suggest secrets for deletion

- **Naming Convention Compliance:**
  - Define regex patterns for secret names (e.g., `^[A-Z_]+$`)
  - Flag secrets that don't follow convention
  - Show inconsistencies (e.g., `API_KEY` vs `APIKEY`)

- **Variables Overview:**
  - List organization and repository variables
  - Show variable values (if permissions allow)
  - Compare variable values across repos
  - Detect duplicate variables (same name, different scopes)

**Compliance Features:**
- Show secrets without rotation metadata (if using external tools)
- Flag secrets shared across many repos (potential security risk)
- Export inventory for compliance audits

**Implementation:**
- Use `/orgs/{org}/actions/secrets` API
- Use `/repos/{owner}/{repo}/actions/secrets` API
- Parse workflow YAML with `gopkg.in/yaml.v3`
- Match `${{ secrets.NAME }}` references

### 3.3 Release & Tag Overview

**Gap:** Existing tools automate releases ([semantic-release](https://github.com/semantic-release/semantic-release)) but don't provide multi-repo visibility.

**Capabilities:**
- **Multi-Repo Release Dashboard:**
  - View latest release for each repository
  - Display version, release date, author
  - Show release notes (truncated with expand option)
  - Filter by date range, author, version pattern

- **Version Comparison:**
  - Compare versions across repos (e.g., show all repos on v1.x vs v2.x)
  - Highlight repos behind on versioning (last release >90 days ago)
  - Show semver compliance (flag non-semver releases)
  - Display version progression timeline

- **Release Notes Aggregation:**
  - Combine release notes from multiple repos (monorepo-style)
  - Group by version prefix (e.g., all `v2.1.*` releases)
  - Export aggregated notes (Markdown, HTML)
  - Useful for changelog generation

- **Tag Management:**
  - List tags across repositories
  - Show tag author, date, commit SHA
  - Compare tag naming conventions
  - Delete tags (with confirmation)

**Semver Compliance Check:**
- Parse version strings (e.g., `v1.2.3`, `1.2.3-beta.1`)
- Validate semver format
- Flag invalid versions (e.g., `v1.2`, `release-2024-01-01`)

**Implementation:**
- Use `/repos/{owner}/{repo}/releases` API
- Use `/repos/{owner}/{repo}/tags` API
- Parse versions with `github.com/Masterminds/semver/v3`
- Cache releases (24h TTL)

## Architecture Changes

### New Packages
```
internal/
├── github/
│   ├── collaborators.go   # Collaborators API
│   ├── secrets.go         # Secrets/Variables API
│   └── releases.go        # Releases/Tags API
├── tui/components/
│   ├── collaborators/     # Collaborator management UI
│   │   ├── list.go
│   │   ├── grant.go
│   │   └── audit.go
│   ├── secrets/           # Secrets audit UI
│   │   ├── inventory.go
│   │   ├── usage.go
│   │   └── compliance.go
│   └── releases/          # Release overview UI
│       ├── dashboard.go
│       ├── comparison.go
│       └── aggregator.go
├── workflows/
│   └── parser.go          # Parse workflow YAML
└── scheduler/
    └── cron.go            # Periodic checks (access expiration)
```

## Implementation Logic

### 3.1 Access Expiration Reminder

```go
type CollaboratorGrant struct {
    User       string
    Repository string
    Permission string
    GrantedBy  string
    GrantedAt  time.Time
    ExpiresAt  time.Time
    RevokedAt  *time.Time
}

func CheckExpiringAccess(grants []CollaboratorGrant, warnDays int) []Notification {
    now := time.Now()
    var notifications []Notification

    for _, grant := range grants {
        if grant.RevokedAt != nil {
            continue  // Already revoked
        }

        daysUntilExpiry := int(grant.ExpiresAt.Sub(now).Hours() / 24)

        if daysUntilExpiry <= warnDays && daysUntilExpiry > 0 {
            notifications = append(notifications, Notification{
                Type:    "warning",
                Message: fmt.Sprintf("Access for %s on %s expires in %d days",
                                     grant.User, grant.Repository, daysUntilExpiry),
                Action:  "extend_or_revoke",
            })
        } else if daysUntilExpiry <= 0 {
            notifications = append(notifications, Notification{
                Type:    "critical",
                Message: fmt.Sprintf("Access for %s on %s has expired",
                                     grant.User, grant.Repository),
                Action:  "revoke_now",
            })
        }
    }

    return notifications
}
```

### 3.2 Unused Secrets Detection

```go
type SecretUsage struct {
    Name         string
    Scope        string  // "org" or "repo"
    Repository   string  // Empty for org secrets
    ReferencedIn []string  // Workflow files that reference this secret
    LastUsed     *time.Time  // Last workflow run that used it
}

func DetectUnusedSecrets(secrets []Secret, workflows []WorkflowFile, runs []WorkflowRun) []SecretUsage {
    usageMap := make(map[string]*SecretUsage)

    // Initialize usage map
    for _, secret := range secrets {
        key := fmt.Sprintf("%s:%s", secret.Scope, secret.Name)
        usageMap[key] = &SecretUsage{
            Name:       secret.Name,
            Scope:      secret.Scope,
            Repository: secret.Repository,
        }
    }

    // Find references in workflows
    for _, wf := range workflows {
        secretRefs := parseSecretsFromYAML(wf.Content)
        for _, ref := range secretRefs {
            key := fmt.Sprintf("org:%s", ref)  // Try org first
            if usage, ok := usageMap[key]; ok {
                usage.ReferencedIn = append(usage.ReferencedIn, wf.Path)
            } else {
                key = fmt.Sprintf("repo:%s", ref)  // Try repo
                if usage, ok := usageMap[key]; ok {
                    usage.ReferencedIn = append(usage.ReferencedIn, wf.Path)
                }
            }
        }
    }

    // Find last usage from workflow runs
    for _, run := range runs {
        // This requires analyzing logs, which is expensive
        // Alternative: just show secrets with no references
    }

    // Return secrets with no references
    var unused []SecretUsage
    for _, usage := range usageMap {
        if len(usage.ReferencedIn) == 0 {
            unused = append(unused, *usage)
        }
    }

    return unused
}

func parseSecretsFromYAML(content string) []string {
    // Regex to match ${{ secrets.NAME }}
    re := regexp.MustCompile(`\$\{\{\s*secrets\.([A-Z_][A-Z0-9_]*)\s*\}\}`)
    matches := re.FindAllStringSubmatch(content, -1)

    var secrets []string
    for _, match := range matches {
        if len(match) > 1 {
            secrets = append(secrets, match[1])
        }
    }

    return secrets
}
```

### 3.3 Version Comparison

```go
type RepositoryRelease struct {
    Repository  string
    Version     string
    SemVer      *semver.Version  // nil if not semver
    ReleasedAt  time.Time
    Author      string
    Notes       string
}

func CompareVersions(releases []RepositoryRelease) VersionReport {
    versionGroups := make(map[string][]string)  // major.minor -> repos

    for _, rel := range releases {
        if rel.SemVer == nil {
            continue  // Skip non-semver
        }

        majorMinor := fmt.Sprintf("v%d.%d", rel.SemVer.Major(), rel.SemVer.Minor())
        versionGroups[majorMinor] = append(versionGroups[majorMinor], rel.Repository)
    }

    return VersionReport{
        Groups: versionGroups,
        Outdated: findOutdatedRepos(releases),
        NonSemVer: findNonSemVer(releases),
    }
}

func findOutdatedRepos(releases []RepositoryRelease) []string {
    threshold := time.Now().AddDate(0, -3, 0)  // 90 days ago
    var outdated []string

    for _, rel := range releases {
        if rel.ReleasedAt.Before(threshold) {
            outdated = append(outdated, rel.Repository)
        }
    }

    return outdated
}
```

## Open Questions

1. **Collaborator Management:**
   - Should we integrate with HR systems for auto-expiration?
   - How to handle team-based access (GitHub teams vs direct collaborators)?
   - Should we track access via GitHub teams separately?

2. **Secrets Audit:**
   - How to verify secrets are rotated (requires external tool integration)?
   - Should we support custom secret naming conventions per org?
   - What to do with environment-specific secrets (staging vs prod)?

3. **Release Management:**
   - Should we support creating releases from the TUI?
   - How to handle monorepo releases (single release, multiple packages)?
   - Should we validate release notes format (conventional changelog)?

4. **Auto-Revoke:**
   - Should auto-revoke be opt-in or opt-out?
   - How to handle cases where access was extended manually via GitHub UI?
   - Should we send notifications before auto-revoking?

## Test Cases

### 3.1 Collaborator Management Tests

**Unit Tests:**
- `TestCheckExpiringAccess`: Expiration detection logic
- `TestBulkAccessGrant`: Bulk onboarding across repos
- `TestAccessAuditLog`: Audit log generation

**Integration Tests:**
- `TestAddCollaborator`: API call to add collaborator
- `TestRevokeAccess`: API call to remove collaborator
- `TestFetchCollaborators`: List collaborators across repos

**TUI Tests:**
- `TestCollaboratorListView`: Display collaborator list
- `TestAccessGrantForm`: Interactive access grant form

### 3.2 Secrets Audit Tests

**Unit Tests:**
- `TestDetectUnusedSecrets`: Unused secret detection
- `TestParseSecretsFromYAML`: YAML parsing for secret refs
- `TestNamingConventionCheck`: Secret name validation

**Integration Tests:**
- `TestFetchOrgSecrets`: API call for org secrets
- `TestFetchRepoSecrets`: API call for repo secrets

**TUI Tests:**
- `TestSecretsInventoryView`: Display secrets list
- `TestUnusedSecretsView`: Highlight unused secrets

### 3.3 Release Overview Tests

**Unit Tests:**
- `TestCompareVersions`: Version comparison logic
- `TestSemVerValidation`: Semver parsing and validation
- `TestAggregateReleaseNotes`: Combine release notes

**Integration Tests:**
- `TestFetchReleases`: API call for releases
- `TestFetchTags`: API call for tags

**TUI Tests:**
- `TestReleaseDashboardView`: Display release dashboard
- `TestVersionComparisonView`: Version comparison table

## Success Criteria

- [ ] Collaborator management with expiration tracking works
- [ ] Secrets audit identifies unused secrets (>90% accuracy)
- [ ] Release dashboard shows multi-repo version overview
- [ ] Semver validation flags invalid versions
- [ ] Test coverage >80%
- [ ] Demo videos for all features

## Performance Targets

- Collaborator list: <3s for 50 repos
- Secrets inventory: <5s for 100 secrets + 200 workflows
- Release dashboard: <5s for 50 repos

## Related Documentation

- See Phase 1 for foundational features
- See Phase 2 for Actions and settings features
- See Phase 4 for integrations (Linear, mani)
- See `anti-phases.md` for features explicitly NOT in scope
