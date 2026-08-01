# Next steps

Immediate follow-ups and open questions. Feature work belongs in `ROADMAP.md`;
the destructive-command guardrails are tracked there as M0.

## Open questions from Kyle (screenshots on desktop)

1. I want to 'open' the repo and make edits
1. Remove excess metadata (fork is useful, but star/follower not)
1. Are the categories right? There seems to be overlap
1. Unwatched is misleading because it is the default (I just prefer to have All activity)
1. Custom isn't supported because I used to have it for Corallium with releases turned off, but that appeared as default

## Homebrew cask

The `TAP_DEPLOY_KEY` secret landed on 2026-07-30, minutes after v0.5.0 published,
so no gh-sweep cask exists in `KyleKing/homebrew-tap` yet. The next release is the
first one that can push it. Check `KyleKing/homebrew-tap` after it and drop the
caveat in `docs/troubleshooting.md` once the cask is there.
