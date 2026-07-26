# Next steps

A review pass on 2026-07-21 looked at an earlier commit; most of what it flagged (stub commands, the bubbletea v2 migration, release plumbing, template adoption) has since shipped and is recorded in `ROADMAP.md` under "Shipped". Feature work belongs in `ROADMAP.md`. This file holds the safety gaps in the destructive `orphans` path, which the roadmap does not track and which are still present as of v0.1.0.

## Destructive cleanup has no confirmation

`runCleanup` in `internal/cli/orphans.go` deletes every classified branch immediately. `--dry-run` previews, but `gh-sweep orphans --cleanup` without it deletes with no prompt, no count summary to confirm, and no limit. If `--org` and `--namespace` are both omitted, the namespace resolves to the authenticated user and the scan covers every non-archived repo you own, so a bare `orphans --cleanup` is an account-wide branch delete.

Add an interactive confirmation before the delete loop: print the count and the full list, then require a typed `yes` (or a `--yes`/`--force` flag to skip it for automation). The TUI orphans component already gates deletion behind a `y/N` screen, so the CLI is the asymmetric path.

## Closed-but-unmerged PR branches are treated as orphans

`internal/orphans/detector.go` classifies a branch whose PR was closed without merging as `OrphanTypeClosedPR` and returns it as deletable. A closed PR often means abandoned-for-now, not safe-to-delete, so this can erase unmerged work. Consider excluding `OrphanTypeClosedPR` from `--cleanup` by default and requiring an explicit opt-in flag to include it, or at least calling it out separately in the confirmation list so it is never deleted silently alongside merged branches.

## The default stale threshold is aggressive

`DefaultScanOptions` in `internal/orphans/types.go` sets `StaleDaysThreshold` to 7. A branch with no PR whose last commit is a week old is classified as stale and deletable. Seven days is short for personal repos where a branch can sit untouched between sessions. Consider raising the default to 30, or leaving 7 but making the confirmation prompt above non-negotiable for the stale class.

## Watch-status changes are unguarded

`watching --watch-all` has no confirmation and no dry-run, and the TUI `w`/`u` keys mutate subscriptions with no prompt. Lower stakes than branch deletion since it only changes notifications, but the same confirmation pattern would make the tool consistent.

## Working tree

As of this writing the working tree carries uncommitted changes in `hk.pkl`, `internal/tui/main.go`, `internal/tui/components/orphans/model.go`, and several golden files, from work in progress. Fold these into a commit or discard them before acting on the above, so a cleanup change lands on a clean base.

## Coverage

`ROADMAP.md` M3 already tracks the coverage gap (12 of 22 packages untested, including TUI components). The orphan detector and the CLI cleanup path deserve first priority there, since they are the code that deletes things.
