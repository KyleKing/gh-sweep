# Next steps

## 2026-08-27

- Policy Check stays red until you run the new apply. The `protection:` block
  is gone from `.gh-sweep-policy.yaml`, so applying it no longer locks `main`,
  and `.github/workflows/policy.yml` calls the gated `policy-apply.yml` on
  `workflow_dispatch`. Two things only you can do: create a `policy-apply`
  GitHub Environment with yourself as a required reviewer, and set
  `POLICY_APPLY_TOKEN` to a fine-grained PAT scoped to this repo with
  Administration read and write. The default `GITHUB_TOKEN` cannot write repo
  admin settings or toggle secret scanning, which is why the apply job takes a
  secret at all
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
