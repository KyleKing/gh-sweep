// Package theme provides color theme detection and management for the TUI.
package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme defines semantic color roles for the UI.
type Theme struct {
	Primary   color.Color // Mauve - titles, focused elements
	Secondary color.Color // Surface2 - selection backgrounds, subtle chrome
	Accent    color.Color // Teal - selected items
	Muted     color.Color // Overlay2 - subtitles, help text
	Text      color.Color // Text - normal text
	Success   color.Color // Green - healthy/success states
	Warning   color.Color // Yellow - warnings, active highlights
	Error     color.Color // Red - error messages
}

var current = Macchiato()

// Init sets the active theme used by all views.
func Init(t Theme) {
	current = t
}

// Current returns the active theme.
func Current() Theme {
	return current
}

// Latte returns the Catppuccin Latte (light) theme.
func Latte() Theme {
	return Theme{
		Primary:   lipgloss.Color("#8839ef"), // Mauve
		Secondary: lipgloss.Color("#acb0be"), // Surface2
		Accent:    lipgloss.Color("#179299"), // Teal
		Muted:     lipgloss.Color("#7c7f93"), // Overlay2
		Text:      lipgloss.Color("#4c4f69"), // Text
		Success:   lipgloss.Color("#40a02b"), // Green
		Warning:   lipgloss.Color("#df8e1d"), // Yellow
		Error:     lipgloss.Color("#d20f39"), // Red
	}
}

// Macchiato returns the Catppuccin Macchiato (medium-dark) theme.
func Macchiato() Theme {
	return Theme{
		Primary:   lipgloss.Color("#c6a0f6"), // Mauve
		Secondary: lipgloss.Color("#5b6078"), // Surface2
		Accent:    lipgloss.Color("#8bd5ca"), // Teal
		Muted:     lipgloss.Color("#939ab7"), // Overlay2
		Text:      lipgloss.Color("#cad3f5"), // Text
		Success:   lipgloss.Color("#a6da95"), // Green
		Warning:   lipgloss.Color("#eed49f"), // Yellow
		Error:     lipgloss.Color("#ed8796"), // Red
	}
}
