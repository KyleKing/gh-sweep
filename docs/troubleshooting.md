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

`brew install` fails. No cask has published to `kyleking/homebrew-tap` for
gh-sweep yet, and `Formula/gh-sweep.rb` in this repo is an unfilled stub. Install
through `gh extension install` or `go install` instead.
