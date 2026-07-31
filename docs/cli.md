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
| `--stale-days` | Days of inactivity before a branch counts as stale, default 7 |
| `--exclude` | Branch patterns to skip |
| `--include-recent` | Include recent branches that have no PR |
| `--cleanup` | Delete the orphaned branches |
| `--dry-run` | Preview deletions and delete nothing |
| `-o`, `--output` | Output file path |
| `--format` | `table`, `json`, or `markdown`, default `table` |
| `--list` | Print a table instead of the TUI |

Always run `--cleanup --dry-run` before `--cleanup`.

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

Neither REST nor GraphQL can see or set GitHub's "Custom" per-notification-type
watch setting, so a repo left at Custom reports the same state here as one at the
plain default. See
[community discussion 65099](https://github.com/orgs/community/discussions/65099).
