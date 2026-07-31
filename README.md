# gh-sweep

<!-- TODO: no demo GIF is committed yet. Record one and replace this block.
     Tracked in ROADMAP.md. -->

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

Run `gh sweep` with no arguments for the home menu, which reaches all twelve
views. Each view prints its keys along the bottom of the screen.

## What it does not do

- Update dependencies. Use Renovate or Dependabot
- Declare settings or protection rules as code. It reports drift from a baseline
  repo, and Pulumi or Terraform enforce the rules
- Touch issues. The orphan sweep covers branches only
- Automate releases. The releases view reads release state and nothing more
- Follow a CI run live. `gha-perf` reports history, so use watchgha to tail a run
- Write anything in the settings, webhooks, collaborators, secrets, or releases
  views, which are read-only

Full docs: [./docs](./docs)

## License

MIT
