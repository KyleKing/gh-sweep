# Command line

`gh sweep` with no subcommand opens the home menu. Each subcommand opens its own
view, and `--list` prints a table and exits instead.

`--org` and `--repos` are persistent flags that work on every command and
override the [config file](./configuration.md).

## branches

Interactive branch management for one repo.

```bash
gh sweep branches --repo owner/repo
gh sweep branches --repo owner/repo --base develop --list
```

| Flag | Effect |
|------|--------|
| `--repo` | Repository, as `owner/repo` |
| `--base` | Base branch for ahead/behind, defaults to the repository default |
| `--list` | Print a table instead of the TUI |

## comments

Unresolved PR review threads, read through the GraphQL `reviewThreads` API
because REST cannot report resolution state. Without `--pr` it scans the newest
open PRs, capped at 20 to bound API cost.

```bash
gh sweep comments --repo owner/repo
gh sweep comments --repo owner/repo --pr 42 --list
gh sweep comments --repo owner/repo --author octocat --since 2026-01-01 --search TODO
```

| Flag | Effect |
|------|--------|
| `--repo` | Repository, as `owner/repo` |
| `--pr` | One pull request number |
| `--author` | Filter by comment author |
| `--since` | Filter by activity date, `YYYY-MM-DD` |
| `--search` | Case-insensitive substring search in the path or comment text |
| `--list` | Print a table instead of the TUI |

## protection

Compare branch protection across repos, optionally against a baseline.

```bash
gh sweep protection --repos owner/repo1,owner/repo2 --baseline owner/repo1 --list
```

| Flag | Effect |
|------|--------|
| `--repos` | Repos to compare |
| `--baseline` | Repository whose settings the others are measured against |
| `--list` | Print a table instead of the TUI |

## gha-perf

GitHub Actions timing analysis.

```bash
gh sweep gha-perf --repo owner/repo --list-workflows
gh sweep gha-perf --repo owner/repo -w ci.yml -l 50 --days 14
gh sweep gha-perf --repo owner/repo -w ci.yml --by-branch --base-branch main
gh sweep gha-perf --repo owner/repo -w ci.yml -j test --csv runs.csv
```

| Flag | Effect |
|------|--------|
| `--repo` | Repository, as `owner/repo` |
| `-w`, `--workflow` | Workflow file to analyze |
| `-b`, `--branch` | Filter by branch name |
| `-l`, `--limit` | Runs to fetch, default 30 |
| `--days` | Lookback in days, default 30 |
| `-c`, `--compare` | Compare current runs against another branch |
| `--base-branch` | Base branch for comparisons, default `main` |
| `--by-branch` | Group runs by branch and compare against the base |
| `-j`, `--job` | Step breakdown for one job name |
| `--csv` | Export detailed data to a CSV file |
| `--cache-only` | Read the cache and fetch nothing |
| `--no-cache` | Neither read nor update the cache |
| `--list-workflows` | List available workflows and exit |

Fetched runs land in a JSON cache at
`~/.cache/gh-sweep/gha-perf/<owner>_<repo>.json`, merged by run ID, so a repeat
invocation only fetches runs it has not seen.

## orphans

Branches whose PR merged, whose PR closed, or which have no PR and no recent
activity, across a namespace.

```bash
gh sweep orphans --org my-org
gh sweep orphans --repos owner/repo1,owner/repo2 --stale-days 14 --list
gh sweep orphans --org my-org --cleanup --dry-run
```

| Flag | Effect |
|------|--------|
| `--org` | Organization to scan |
| `--namespace` | Namespace, org or user, to scan |
| `--repos` | Specific repos to scan |
| `--stale-days` | Days of inactivity before a branch counts as stale, default 30 |
| `--exclude` | Branch patterns to skip |
| `--include-recent` | Include recent branches that have no PR |
| `--cleanup` | Delete the orphaned branches |
| `--dry-run` | Preview deletions and delete nothing |
| `--yes` | Skip the cleanup confirmation prompt |
| `--include-closed-pr` | Include closed-PR branches in `--cleanup` (excluded by default) |
| `-o`, `--output` | Output file path |
| `--format` | `table`, `json`, or `markdown`, default `table` |
| `--list` | Print a table instead of the TUI |

Always run `--cleanup --dry-run` before `--cleanup`.

## pages

Cross-check GitHub Pages custom domains against live DNS, in both directions:
repos whose domain no longer resolves to GitHub Pages, repos where DNS still
points at GitHub Pages while Pages itself is disabled (a subdomain-takeover
risk), and repos with an unverified custom domain. Set the `pages.domains`
[config key](./configuration.md) to reverse-check DNS-configured subdomains
that should have a live Pages site behind them.

A domain proxied through Cloudflare or another CDN resolves to the proxy's
IPs rather than GitHub's, so the audit can't see past the proxy and reports a
false "dangling" finding for it; verify those by hand.

```bash
gh sweep pages --org my-org
gh sweep pages --namespace my-user --format json
```

| Flag | Effect |
|------|--------|
| `--org` | Organization to scan |
| `--namespace` | Namespace, org or user, to scan |
| `-o`, `--output` | Output file path |
| `--format` | `table`, `json`, or `markdown`, default `table` |

## policy

Diff and sync repo settings, security & analysis, release immutability, branch
protection, and repository rulesets against a declared
[policy file](./configuration.md#policy-file).
A field left out of the policy is never reported or changed.

```bash
gh sweep policy
gh sweep policy --list
gh sweep policy --list --format json
gh sweep policy --apply
gh sweep policy --apply --yes
```

| Flag | Effect |
|------|--------|
| `--policy` | Path to the policy file, default `.gh-sweep-policy.yaml` |
| `--list` | Print a table instead of the TUI; exits 1 if any repo has drift |
| `--apply` | Sync drifted repos toward the policy, confirming each repo |
| `--yes` | Skip the confirmation prompt when applying (scripted or CI use) |
| `--format` | `table`, `json`, or `markdown`, default `table` |

`--list --format json` exits non-zero on drift, which makes it usable as a
scheduled drift check without parsing its output:

```yaml
# .github/workflows/policy-check.yml
on:
  schedule:
    - cron: "0 6 * * 1" # every Monday
  workflow_dispatch:

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: cli/gh-extension-precompile@v2 # or `go install` the binary
      - run: gh extension install kyleking/gh-sweep
      - run: gh sweep policy --list --format json
        env:
          GITHUB_TOKEN: ${{ secrets.GH_SWEEP_TOKEN }}
```

The job fails when drift is found. `GH_SWEEP_TOKEN` needs read access to every
repo the policy covers; the default `GITHUB_TOKEN` only covers the repo the
workflow runs in.

To apply drift unattended, call the
[`policy-apply.yml`](../.github/workflows/policy-apply.yml) reusable workflow
instead of running `--apply --yes` directly. It checks for drift first and
only runs the apply job when drift exists, and that job targets a GitHub
Environment you name, so a required reviewer on that Environment has to
approve before anything gets written:

```yaml
# .github/workflows/policy-apply.yml
on:
  schedule:
    - cron: "0 6 * * 1" # every Monday
  workflow_dispatch:

jobs:
  policy:
    uses: KyleKing/gh-sweep/.github/workflows/policy-apply.yml@v0.6.0
    with:
      environment: policy-apply
    secrets:
      token: ${{ secrets.GH_SWEEP_TOKEN }}
```

Pin the `@` ref to the gh-sweep release you want; the caller only needs this
short block, not a copy of the workflow itself. Create the `policy-apply`
Environment in the caller repo's settings and add required reviewers there —
that's the approval gate, gh-sweep does not implement its own.

## watching

Audit repository watch status.

```bash
gh sweep watching --unwatched
gh sweep watching --watch-all
```

| Flag | Effect |
|------|--------|
| `--unwatched` | List unwatched repositories |
| `--watch-all` | Watch every unwatched repository |
| `--yes` | Skip the watch-all confirmation prompt |

Neither REST nor GraphQL can see or set GitHub's "Custom" per-notification-type
watch setting, so a repo left at Custom reports the same state here as one at the
plain default. See
[community discussion 65099](https://github.com/orgs/community/discussions/65099).
