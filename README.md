# gh-sweep

![demo](https://raw.githubusercontent.com/KyleKing/gh-sweep/main/.github/assets/demo.gif)

Audit and clean up many GitHub repositories from the terminal: dead branches,
unresolved review threads, branch-protection drift, and slow workflows. Every
view works across an org or an explicit list of repos, so you never open one
repo at a time.

## Install

```bash
# GitHub CLI extension
gh extension install kyleking/gh-sweep
# Go
go install github.com/KyleKing/gh-sweep/cmd/gh-sweep@latest
# from source
go build -o gh-sweep ./cmd/gh-sweep
```

## Quick start

Find every branch whose PR merged or closed, across a whole org, and preview the
cleanup before deleting anything:

```bash
gh sweep orphans --org my-org --cleanup --dry-run
```

Run `gh sweep` with no arguments for the home menu, which reaches all thirteen
views. Each view prints its keys along the bottom of the screen.

## What it does not do

- Update dependencies. Use Renovate or Dependabot
- Touch issues. The orphan sweep covers branches only
- Automate releases. The releases view reads release state and nothing more
- Follow a CI run live. `gha-perf` reports history, so use watchgha to tail a run
- Write anything in the webhooks, collaborators, secrets, or releases views,
  which are read-only. Repo settings, security & analysis, release
  immutability, and branch protection are the exception: `gh sweep policy`
  diffs and applies those against a file you declare (see
  [docs/alternatives.md](docs/alternatives.md))

Full docs: [./docs](./docs)

## License

MIT
