# TUI UX

## Layout

Running `gh-sweep` with no subcommand opens a full-screen home menu. Each entry opens a single-view component; `esc` returns home from any view. Components are rebuilt fresh on every entry (no state carries across visits).

```
+--------------------------------------------------------------+
| gh-sweep                                                     |
|                                                              |
| Namespace Audit (scans your whole account or org, ignores    |
| --repo)                                                      |
|   [w]atch status         - Audit and manage repo watching    |
|   [o]rphan branches      - Detect and clean up orphaned...   |
|                                                              |
| Single Repo (needs --repo)                                   |
| > [b]ranch management    - Interactive branch operations     |
|   pr [c]omments          - Review unresolved comments        |
|   [a]nalytics            - CI/CD and repository statistics   |
|   [g]ha performance      - Workflow timing analysis          |
|                                                              |
| Cross-Repo (needs --repos or --org)                          |
|   branch pro[t]ection    - Compare and sync protection rules |
|   [s]ettings comparison  - Cross-repo settings diff          |
|   web[h]ooks             - Webhook health monitoring         |
|   co[l]laborators        - Manage repository access          |
|   s[e]crets audit        - Review secrets usage (read-only)  |
|   [r]releases             - Release version overview          |
|                                                              |
| Policy (diffs and applies against a declared policy file)    |
|   polic[y]               - Diff and sync settings against... |
|                                                              |
| j/k or ↑/↓: move | enter: select | letter in [brackets]: jump|
+--------------------------------------------------------------+
```

With no repo or repos configured, the menu shows a hint to pass `--repo` or add repositories to `.gh-sweep.yaml`. Branches, comments, analytics, and GHA performance use the single target repo; protection, settings, webhooks, collaborators, and releases use the repo list; secrets needs an org plus the repo list.

## Representative view: branches

```
+--------------------------------------------------------------+
| Branch Management: owner/repo                                |
|                                                              |
|   [x] feature/old-experiment   merged    3 ahead / 40 behind |
| > [ ] fix/flaky-test           stale     1 ahead / 12 behind |
|   [ ] release/1.2              protected                     |
|                                                              |
| Delete 2 branches? feature/old-experiment, fix/flaky-test    |
| Press 'y' to confirm, 'n' or 'esc' to cancel                 |
|                                                              |
| j/k: navigate | space: select | a/n: all/none | d: delete    |
| | r: refresh | q: quit                                       |
+--------------------------------------------------------------+
```

`d` with no selection targets the branch under the cursor. Protected branches are excluded from deletion and reported in the status line.

## Keybindings

Global: `q` or `ctrl+c` quits from any view, `esc` returns to the home menu.

### Branches (delete confirm: `y` confirm, `n`/`esc` cancel)

| Key                  | Action                         |
| -------------------- | ------------------------------ |
| `j`/`k`, `down`/`up` | Move cursor                    |
| `space`              | Toggle selection               |
| `a` / `n`            | Select all / none              |
| `d`                  | Delete selected (with confirm) |
| `r`                  | Refresh                        |

### Orphans (same confirm flow as branches)

| Key                         | Action                                   |
| --------------------------- | ---------------------------------------- |
| `space`, `a`, `n`, `d`, `r` | As in branches                           |
| `1`-`4`                     | Filter: All / Merged / Closed / Stale    |
| `v`                         | Cycle grouping: by repo / by type / flat |

### Watching

| Key             | Action                                                     |
| --------------- | ----------------------------------------------------------- |
| `1`/`2`/`3`/`4` | Tab: Unwatched / Watched / Ignored / All                    |
| `space`         | Toggle selection                                             |
| `w` / `u` / `i` | Watch (all activity) / unwatch (default) / ignore selected (no confirm step) |

Each row also shows stars, total watcher count, archived/fork flags, and last-pushed date, all sourced from a single GraphQL query per page rather than one REST call per repo. GitHub's API has no representation of the "Custom" per-notification-type watch setting, so a repo configured that way on github.com shows here as its nearest visible state (usually "default"); see DESIGN.md.

### Comments

| Key     | Action                                |
| ------- | ------------------------------------- |
| `j`/`k` | Move cursor                           |
| `r`     | Toggle resolved threads (not refresh) |

### Analytics

| Key         | Action                                      |
| ----------- | ------------------------------------------- |
| `1`/`2`/`3` | Tab: Overview / Flaky Tests / Errors        |
| `s`         | Export errors to markdown (errors tab only) |

### GHA Performance

| Key     | Action                                      |
| ------- | ------------------------------------------- |
| `1`-`4` | Tab: Overview / Workflows / Jobs / Branches |
| `j`/`k` | Scroll                                      |
| `r`     | Refresh (bypasses cache)                    |

### Tabbed read-only views

| View          | Keys        | Tabs                               |
| ------------- | ----------- | ---------------------------------- |
| Settings      | `1`/`2`     | Overview / Differences             |
| Collaborators | `1`/`2`     | By Repository / By User            |
| Secrets       | `1`/`2`/`3` | Organization / Repository / Unused |
| Releases      | `1`/`2`/`3` | Latest / All Releases / Outdated   |

Protection and webhooks have only `j`/`k` navigation.

## Status and theme

- Only three views mutate GitHub state: branches and orphans (branch deletion, behind a `y`/`n` confirm) and watching (subscription changes, no confirm)
- Colors come from `internal/tui/theme`: Catppuccin Latte (light) or Macchiato (dark), auto-detected from the terminal background, overridable with `CATPPUCCIN_THEME=latte|macchiato` (aliases `light`/`dark`)
- Roles: Primary (titles), Muted (help text), Success/Warning/Error (state coloring, e.g. release age over 90 days renders Error)
