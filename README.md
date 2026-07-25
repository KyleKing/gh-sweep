# gh-sweep

TUI for sweeping GitHub repositories: dead branches, unresolved review threads, protection drift, and slow workflows.

## Installation

As a GitHub CLI extension:

> **Note:** Requires a [GitHub Release](https://github.com/KyleKing/gh-sweep/releases) with precompiled binaries. For development, see [local install instructions](CONTRIBUTING.md#development-install).

```bash
gh extension install kyleking/gh-sweep
```

With Homebrew, directly from the repository formula:

```bash
brew install --formula https://github.com/KyleKing/gh-sweep/raw/main/Formula/gh-sweep.rb
```

With Go:

```bash
go install github.com/KyleKing/gh-sweep/cmd/gh-sweep@latest
```

Or build from source:

```bash
go build -o gh-sweep ./cmd/gh-sweep
```

## Usage

```bash
# Launch the full TUI (home menu with all views)
gh sweep

# Interactive branch management for one repo; --list prints a table instead
gh sweep branches --repo owner/repo
gh sweep branches --repo owner/repo --base develop --list

# Review unresolved PR review threads (GraphQL reviewThreads);
# without --pr, scans the newest open PRs (capped at 20)
gh sweep comments --repo owner/repo
gh sweep comments --repo owner/repo --pr 42 --list
gh sweep comments --repo owner/repo --author octocat --since 2026-01-01 --search TODO

# Compare branch protection across repos, optionally against a baseline
gh sweep protection --repos owner/repo1,owner/repo2 --baseline owner/repo1 --list

# Analyze GitHub Actions workflow performance
gh sweep gha-perf --repo owner/repo --list-workflows
gh sweep gha-perf --repo owner/repo -w ci.yml -l 50 --days 14
gh sweep gha-perf --repo owner/repo -w ci.yml --by-branch --base-branch main
gh sweep gha-perf --repo owner/repo -w ci.yml -j test --csv runs.csv

# Detect orphaned branches (merged PR, closed PR, or stale) across a namespace
gh sweep orphans --org my-org
gh sweep orphans --repos owner/repo1,owner/repo2 --stale-days 14 --list
gh sweep orphans --org my-org --cleanup --dry-run

# Audit repository watch status
gh sweep watching --unwatched
gh sweep watching --watch-all
```

The persistent `--org` and `--repos` flags work on every command and override the config file.

## Configuration

An optional YAML config is read from the first of `./.gh-sweep.yaml`, `~/.gh-sweep.yaml`, or `~/.config/gh-sweep/config.yaml`. Flags take precedence. See [.gh-sweep.yaml.example](.gh-sweep.yaml.example) for every key; the common ones:

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

## Keybindings

`q` or `ctrl+c` quits from any view; `esc` returns to the home menu. See [UX.md](UX.md) for layout mockups.

### Home menu

| Key | View                |
| --- | ------------------- |
| `0` | Watch status        |
| `o` | Orphan branches     |
| `1` | Branch management   |
| `2` | Branch protection   |
| `3` | PR comments         |
| `4` | Analytics           |
| `p` | GHA performance     |
| `5` | Settings comparison |
| `6` | Webhooks            |
| `7` | Collaborators       |
| `8` | Secrets audit       |
| `9` | Releases            |

### Branches and orphans

| Key       | Action                                                              |
| --------- | ------------------------------------------------------------------- |
| `j`/`k`   | Move cursor                                                         |
| `space`   | Toggle selection                                                    |
| `a` / `n` | Select all / none                                                   |
| `d`       | Delete selected (or cursor row), then `y` confirm, `n`/`esc` cancel |
| `r`       | Refresh                                                             |
| `1`-`4`   | Filter all/merged/closed/stale (orphans only)                       |
| `v`       | Cycle grouping by repo/type/flat (orphans only)                     |

### Tabbed views

Number keys switch tabs; `j`/`k` navigate within a tab.

| View            | Tabs                                                                         |
| --------------- | ---------------------------------------------------------------------------- |
| Watching        | `1` Unwatched, `2` Watched, `3` All (`space` select, `w` watch, `u` unwatch) |
| Analytics       | `1` Overview, `2` Flaky Tests, `3` Errors (`s` exports errors to markdown)   |
| GHA performance | `1` Overview, `2` Workflows, `3` Jobs, `4` Branches (`r` refresh)            |
| Settings        | `1` Overview, `2` Differences                                                |
| Collaborators   | `1` By Repository, `2` By User                                               |
| Secrets         | `1` Organization, `2` Repository, `3` Unused                                 |
| Releases        | `1` Latest, `2` All Releases, `3` Outdated                                   |

Comments uses `j`/`k` plus `r` to toggle resolved threads; protection and webhooks are navigation only.

## Features

- Branch management with ahead/behind counts and guarded batch delete (protected patterns are never deleted)
- Orphan detection across a whole org or user: branches whose PR merged, whose PR closed, or with no PR and no recent activity
- Unresolved review threads via the GraphQL `reviewThreads` API, filterable by PR, author, date, and text
- Branch protection drift table against a baseline repo
- GitHub Actions timing analysis with per-workflow, per-job, and per-branch stats, regression flagging, CSV export, and a JSON cache
- Read-only views for settings drift, webhook health, collaborator access, secrets inventory, and release age
- Watch-status audit with bulk watch/unwatch
- Catppuccin theme (Latte/Macchiato) auto-detected from the terminal background, `CATPPUCCIN_THEME` override

## Development

```bash
mise install && hk install --mise
mise run ci
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the task table and release flow, [DESIGN.md](DESIGN.md) for architecture, and [ROADMAP.md](ROADMAP.md) for planned milestones (including a demo GIF).

## Alternatives

[gh-dash](https://github.com/dlvhdr/gh-dash) is a personal PR and issue dashboard: a unified view of the PRs and issues assigned to you across repos, with a quick review workflow. Use it for daily triage of your own work, and gh-sweep for repository administration across many repos.

[watchgha](https://github.com/nedbat/watchgha) tails GitHub Actions runs live with streaming status updates. Use it while actively developing to watch CI in real time. gh-sweep's gha-perf command covers the historical side (duration trends, regressions, flaky heuristics).

[gh-poi](https://github.com/seachicken/gh-poi) safely deletes local branches whose PRs have merged, from a single checkout. Use it for local single-repo hygiene. gh-sweep's branches and orphans commands work on remote branches across many repos without a checkout.

For automation categories gh-sweep deliberately avoids (dependency updates, settings as code, stale-issue bots, release automation), see [docs/alternatives.md](docs/alternatives.md).

## License

MIT
