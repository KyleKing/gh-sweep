package tui

import (
	"time"

	"github.com/kyleking/aragonite/keyusage"

	"github.com/KyleKing/gh-sweep/internal/tui/components/analytics"
	"github.com/KyleKing/gh-sweep/internal/tui/components/collaborators"
	"github.com/KyleKing/gh-sweep/internal/tui/components/comments"
	"github.com/KyleKing/gh-sweep/internal/tui/components/ghaperf"
	orphanstui "github.com/KyleKing/gh-sweep/internal/tui/components/orphans"
	policytui "github.com/KyleKing/gh-sweep/internal/tui/components/policy"
	"github.com/KyleKing/gh-sweep/internal/tui/components/protection"
	"github.com/KyleKing/gh-sweep/internal/tui/components/releases"
	"github.com/KyleKing/gh-sweep/internal/tui/components/secrets"
	"github.com/KyleKing/gh-sweep/internal/tui/components/settings"
	"github.com/KyleKing/gh-sweep/internal/tui/components/watching"
	"github.com/KyleKing/gh-sweep/internal/tui/components/webhooks"
	"github.com/KyleKing/gh-sweep/internal/tui/keymap"
)

// StaleAfter is how long a key may go unpressed before `gh-sweep keys` lists it as neglected.
const StaleAfter = 28 * 24 * time.Hour

var viewContexts = map[ViewMode]string{
	ViewHome:          "home",
	ViewProtection:    "protection",
	ViewComments:      "comments",
	ViewAnalytics:     "analytics",
	ViewGHAPerf:       "ghaperf",
	ViewSettings:      "settings",
	ViewWatching:      "watching",
	ViewWebhooks:      "webhooks",
	ViewCollaborators: "collaborators",
	ViewSecrets:       "secrets",
	ViewReleases:      "releases",
	ViewOrphans:       "orphans",
	ViewPolicy:        "policy",
}

// Keymap is every context Update records keys under, with the keys each handles.
func Keymap() keyusage.Keymap {
	return keyusage.Keymap{
		viewContexts[ViewHome]:          homeBindings(),
		viewContexts[ViewProtection]:    protection.Bindings(),
		viewContexts[ViewComments]:      comments.Bindings(),
		viewContexts[ViewAnalytics]:     analytics.Bindings(),
		viewContexts[ViewGHAPerf]:       ghaperf.Bindings(),
		viewContexts[ViewSettings]:      settings.Bindings(),
		viewContexts[ViewWatching]:      watching.Bindings(),
		viewContexts[ViewWebhooks]:      webhooks.Bindings(),
		viewContexts[ViewCollaborators]: collaborators.Bindings(),
		viewContexts[ViewSecrets]:       secrets.Bindings(),
		viewContexts[ViewReleases]:      releases.Bindings(),
		viewContexts[ViewOrphans]:       orphanstui.Bindings(),
		viewContexts[ViewPolicy]:        policytui.Bindings(),
	}
}

func homeBindings() []keyusage.Binding {
	items := (MainModel{}).buildMenuItems()

	out := make([]keyusage.Binding, 0, len(items)+5)
	out = append(out,
		keymap.Bind("quit", "q", "ctrl+c"),
		keymap.Bind("filter", "/"),
		keymap.Bind("up", "up", "k"),
		keymap.Bind("down", "down", "j"),
		keymap.Bind("open selected", "enter"),
	)

	for _, item := range items {
		out = append(out, keymap.Bind("open "+viewContexts[item.view], item.key))
	}

	return out
}

// typing reports whether the key about to be dispatched lands in a text field.
func (m MainModel) typing() bool {
	switch m.mode {
	case ViewHome:
		return m.menuFiltering
	case ViewWatching:
		return m.watchingModel.Typing()
	case ViewOrphans:
		return m.orphansModel.Typing()
	case ViewPolicy:
		return m.policyModel.Typing()
	default:
		return false
	}
}

func (m MainModel) recordKey(key string) {
	if !m.typing() {
		m.usage.Key(viewContexts[m.mode], key)
	}
}
