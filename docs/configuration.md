# Configuration

gh-sweep reads the first file it finds of `./.gh-sweep.yaml`,
`~/.gh-sweep.yaml`, or `~/.config/gh-sweep/config.yaml`. A missing file means the
built-in defaults, and flags override whatever the file says.

[.gh-sweep.yaml.example](../.gh-sweep.yaml.example) mirrors the config struct
field for field. The common keys:

```yaml
# Default GitHub organization; bare repo names below are qualified with it
default_org: your-org

# Repositories to manage ("owner/repo" or "repo" with default_org)
repositories:
  - owner/repo1
  - repo2

# Baseline repository for protection/settings comparisons
baseline: owner/repo1

branches:
  default_branch: main
  protected_patterns: [main, master, develop, release/*]

orphans:
  stale_days_threshold: 7
  exclude_patterns: [main, master, develop, release/*, hotfix/*]

gha_perf:
  default_lookback_days: 30
  base_branch: main
  regression_threshold: 20.0
```

The persistent `--org` and `--repos` flags override `default_org` and
`repositories`. Per-command flags such as `--stale-days`, `--days`, and
`--base-branch` fall back to the config only when you do not pass them.

`protected_patterns` decides which branches the delete paths refuse to touch, so
keep it accurate before running any cleanup.
