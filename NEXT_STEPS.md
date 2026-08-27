# Next steps

## 2026-08-27

- Policy Check is red by design, not a bug: `.gh-sweep-policy.yaml` declares
  `delete_branch_on_merge`, `allow_squash_merge`, and secret scanning as
  desired, but the live repo doesn't match, and `gh sweep policy --list`
  exits 1 on any drift for exactly this reason. Applying the policy
  (`gh sweep policy --apply --yes`) also closes drift on `required_reviews: 1`
  and `require_status_checks: ci` in branch protection, which would block
  direct pushes to `main` (your workflow and this freshen pass both rely on
  that). Left untouched pending your call: apply and adjust the push
  workflow, relax the protection block in the policy file, or wire up the
  already-built `policy-apply.yml` (gated by a GitHub Environment) instead of
  hand-applying.
- Noticed while investigating the above: the CI run's default `GITHUB_TOKEN`
  reported different drift (`allow_squash_merge`, `secret_scanning`,
  `secret_scanning_push_protection`) than my personal `gh auth token` did
  locally (`allow_merge_commit`, `allow_rebase_merge`, `required_reviews`,
  `require_status_checks`) against the same policy file. Looks like a
  token-scope difference in what each can read back from the repo settings
  API, which means the CI drift report may be incomplete. Worth checking
  what scopes the Actions token has for repo admin settings.
- `TestTUINavigateViewsAndBack` failed once in CI on 2026-08-20 (sha
  `d63094c`) with a teatest `WaitFor` timeout on "Branch Protection Rules".
  Could not reproduce locally across 50+ runs (default, `-race`, the exact
  CI coverage command, `GOMAXPROCS=2`, `-cpu=1`), and CI passed clean on the
  next push (`c053c1f` and the bump commit). Treating as a one-off runner flake;
  flag if it recurs so it gets a real fix instead of another shrug.
