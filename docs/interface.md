# Interface

## Keys

Every view prints its own keys along the bottom of the screen. That footer reads
the live code, so trust it over this page, which lists only enough to start.

`q` or `ctrl+c` quits from any view, and `esc` returns to the home menu.

| Key | Action |
|-----|--------|
| `j` / `k` | Move the cursor |
| `enter` | Open the highlighted view |
| `/` | Filter the home menu by name |
| `space` | Toggle selection, in views that select |
| `r` | Refresh |

[UX.md](../UX.md) has layout sketches for each view.

## Home menu

| Key | View |
| --- | ------------------- |
| `0` | Watch status |
| `o` | Orphan branches |
| `1` | Branch management |
| `2` | Branch protection |
| `3` | PR comments |
| `4` | Analytics |
| `p` | GHA performance |
| `5` | Settings comparison |
| `6` | Webhooks |
| `7` | Collaborators |
| `8` | Secrets audit |
| `9` | Releases |

Pressing a key opens that view directly. `j`/`k` and the arrow keys move through
the list, and `enter` opens whatever the cursor sits on.

## Selecting and deleting

The branches and orphans views select rows with `space`, select all or none with
`a` and `n`, and delete with `d`. A delete asks for confirmation: `y` goes ahead,
`n` or `esc` backs out. Protected patterns from the config file never appear as
deletable.

Orphans adds `1` through `4` to filter by all, merged, closed, or stale, and `v`
to cycle grouping by repo, by type, or flat.

## Tabbed views

Number keys switch tabs, and `j`/`k` move within the current tab.

| View | Tabs |
| --------------- | ---------------------------------------------------------------------------- |
| Watching | `1` Unwatched, `2` Watched, `3` Ignored, `4` All, with `space` select, `w` watch, `u` unwatch, `i` ignore |
| Analytics | `1` Overview, `2` Flaky Tests, `3` Errors, with `s` to export errors to markdown |
| GHA performance | `1` Overview, `2` Workflows, `3` Jobs, `4` Branches, with `r` to refresh |
| Settings | `1` Overview, `2` Differences |
| Collaborators | `1` By Repository, `2` By User |
| Secrets | `1` Organization, `2` Repository, `3` Unused |
| Releases | `1` Latest, `2` All Releases, `3` Outdated |

Comments uses `j`/`k` plus `r` to toggle resolved threads. Protection and
webhooks are navigation only.

## Theme

Catppuccin Latte on light terminals and Macchiato on dark, picked by reading the
terminal background. Set `CATPPUCCIN_THEME` to `latte`, `light`, `macchiato`, or
`dark` to override.
