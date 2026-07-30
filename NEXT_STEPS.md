# Next steps

Immediate follow-ups and open questions. Feature work belongs in `ROADMAP.md`;
the destructive-command guardrails are tracked there as M0.

## Open questions from Kyle (screenshots on desktop)

1. I want to 'open' the repo and make edits
1. Remove excess metadata (fork is useful, but star/follower not)
1. Are the categories right? There seems to be overlap
1. Unwatched is misleading because it is the default (I just prefer to have All activity)
1. Custom isn't supported because I used to have it for Corallium with releases turned off, but that appeared as default

## Pending the next copier update

This repo is pinned at my_go_template v0.7.0; the template is at v0.9.0.

- `Formula/gh-sweep.rb` and the `brew:sha` task in
  `.config/mise/conf.d/template.toml` are dead. The formula still pins version
  `0.1.0` with `REPLACE_WITH_SHA256` placeholders, so its download URLs 404, and
  `.goreleaser.yml` has generated a real cask for `kyleking/homebrew-tap` since
  template v0.7.0. Template v0.8.0 added a `remove-if-found.txt` manifest that
  deletes the stub automatically, so the update removes both. Nothing to delete
  by hand, and `CONTRIBUTING.md`'s "Updating the Homebrew Formula" section should
  go at the same time
- The update also turns on a CI `hooks` job running `hk check --all`. The one
  finding measured against this repo was a `typos` hit on a commit SHA in
  `ROADMAP.md`, removed on 2026-07-27 with the rest of the Shipped list. Re-run
  `hk check --all` after the update rather than assuming it is clean
