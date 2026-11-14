// Package keymap holds the key bindings that every view shares, in the form
// keyusage reads them.
package keymap

import "github.com/kyleking/aragonite/keyusage"

// Bind is a binding for the keys as tea.KeyPressMsg.String() spells them.
func Bind(what string, keys ...string) keyusage.Binding {
	return keyusage.Binding{Keys: keys, What: what}
}

// Common is the quit, help, back, and cursor keys every list view answers.
func Common() []keyusage.Binding {
	return []keyusage.Binding{
		Bind("quit", "q", "ctrl+c"),
		Bind("help", "?"),
		Bind("back to menu", "esc"),
		Bind("first row", "g"),
		Bind("last row", "G"),
		Bind("up", "up", "k"),
		Bind("down", "down", "j"),
	}
}
