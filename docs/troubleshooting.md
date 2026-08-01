# Troubleshooting

Every view errors on authentication because gh-sweep reuses the `gh` CLI login.
Run `gh auth status`, then `gh auth login` if needed.

A view is empty because no repos are in scope. Pass `--org` or `--repos`, or set
`default_org` and `repositories` in the
[config file](./configuration.md).

Comments finds nothing on a busy repo because, without `--pr`, it only scans the
newest 20 open pull requests. Pass `--pr <number>` for an older one.

A repo shows as unwatched when GitHub says otherwise, because neither REST nor
GraphQL exposes the "Custom" per-notification-type watch setting. A repo left at
Custom reports the same state as one at the plain default. This is a GitHub API
limit, so gh-sweep says so rather than asserting a state it cannot confirm.

`gha-perf` reports stale numbers because it merges into a JSON cache under
`~/.cache/gh-sweep/gha-perf/`. Pass `--no-cache` to ignore it for one run.

`brew install --cask KyleKing/tap/gh-sweep` fails. The tap holds no gh-sweep cask
until the first release published after the `TAP_DEPLOY_KEY` secret was added on
2026-07-30; v0.5.0 predates it. Install through `gh extension install` or
`go install` until a later release appears in the tap.
