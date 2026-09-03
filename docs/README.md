# gh-sweep docs

A Bubble Tea TUI and Cobra CLI for maintaining many GitHub repositories at once.
The bare `gh-sweep` command opens a home menu of thirteen views. Each subcommand
opens its view directly, and `--list` prints a table instead of starting the TUI.

## Pages

- [Interface](./interface.md) for the home menu, the views, and the keys
- [Command line](./cli.md) for every command and flag
- [Configuration](./configuration.md) for the YAML config file
- [Troubleshooting](./troubleshooting.md) for auth failures and empty results
- [Alternatives](./alternatives.md) for the tools that cover what gh-sweep leaves out
- [UX mockups](../UX.md) for per-view layout sketches

Setup, tasks, and the release flow live in
[CONTRIBUTING.md](../CONTRIBUTING.md). Architecture lives in
[DESIGN.md](../DESIGN.md).

## Requirements

- gh CLI, logged in. gh-sweep reuses that authentication through `cli/go-gh`, so
  no separate token setup is needed
- Go 1.25+, only to build or `go install` from source

## What it gives you

- Orphan detection across an org or user: branches whose PR merged, whose PR
  closed, or with no PR and no recent activity, with a guarded batch delete
- `policy`'s branch domain applies the same detection across a repo list
  declaratively, with pruning gated behind `--prune`
- Unresolved review threads through the GraphQL `reviewThreads` API, filterable
  by PR, author, date, and text
- Branch-protection drift against a baseline repo
- GitHub Actions timing with per-workflow, per-job, and per-branch stats,
  regression flagging, CSV export, and a JSON cache
- Read-only audits of settings drift, webhook health, collaborator access,
  secrets inventory, and release age
- Watch-status audit with bulk watch and unwatch
- Catppuccin Latte or Macchiato, chosen from the terminal background, with a
  `CATPPUCCIN_THEME` override
