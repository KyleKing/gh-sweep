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
  stale_days_threshold: 30
  min_age_days: 0
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

`stale_days_threshold` classifies a branch with no PR, so it does not bound what
a cleanup deletes: a branch whose PR merged yesterday is a `merged_pr` orphan at
any threshold. `min_age_days` is the one that spares recent work, and it applies
to every orphan type. A branch whose commit date cannot be read counts as too
recent to touch.

`cache.ttl` serves repeat reads from `cache.path` instead of the API. Cached
responses cost no rate-limit quota, which is what keeps a cross-repo sweep
inside the hourly budget; a stale read is the price. Set `ttl: 0` to disable.

Pass `--config <path>` to use a file outside the search paths, which is how one
machine keeps a separate profile per org or per repo group. An explicit path
that cannot be read is an error rather than a silent fall back to defaults.

## Policy file

The `policy` command reads a second, separate file: `.gh-sweep-policy.yaml`
(project directory, or `~/.gh-sweep-policy.yaml`). Where `.gh-sweep.yaml` holds
flag defaults, `.gh-sweep-policy.yaml` holds desired state: the settings you
want every listed repo to converge on. A field left out is never reported or
changed, so a narrow policy only touches what it declares.

[.gh-sweep-policy.yaml.example](../.gh-sweep-policy.yaml.example) has the full
schema. The shape:

```yaml
default_org: your-org
repositories:
  - owner/repo1
  - repo2 # uses default_org

settings:
  delete_branch_on_merge: true
  allow_squash_merge: true

security:
  secret_scanning: enabled

releases:
  immutable: true

protection:
  required_reviews: 1
  require_status_checks: [ci]

ruleset:
  name: main
  enforcement: active
  block_deletion: true
  block_force_push: true
  pull_request:
    required_approvals: 0
    allowed_merge_methods: [squash]
```

### Protection or ruleset

The two are separate GitHub features that both apply, most-restrictive-wins, so
declaring both means two rules to reason about. Pick one per repo unless you
want that.

`protection:` writes a classic branch-protection rule on the default branch. It
cannot require a pull request without also requiring an approval: setting
`required_reviews: 0` sends `required_pull_request_reviews: null`, which drops
the PR requirement entirely and re-opens direct pushes to the branch.

`ruleset:` writes a repository ruleset, matched by `name`. A `pull_request:`
block with `required_approvals: 0` requires the PR and no approval on it, which
is the usual want for a repo with one or two maintainers. gh-sweep manages only
the rule types it models (`block_deletion`, `block_force_push`,
`require_linear_history`, `require_status_checks`, and `pull_request`); other
rule types and every bypass actor on a live ruleset are carried across
untouched when it updates one.

See [gh sweep policy](./cli.md#policy) for diffing and applying it.
