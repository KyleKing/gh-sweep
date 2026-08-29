// Package theme holds the palette gh-sweep's views draw from, resolved once
// per command and read through Current for the rest of the run.
package theme

import (
	aragonite "github.com/kyleking/aragonite/tui/theme"
)

// Theme is the semantic role set every view renders through.
type Theme = aragonite.Semantic

// Palette is a full Catppuccin flavor, as handed to Init.
type Palette = aragonite.Palette

var current = aragonite.Macchiato().Semantic()

// Detect picks a palette from the CATPPUCCIN_THEME override or the terminal's
// background color.
func Detect() Palette { return aragonite.Detect() }

// Latte returns the Catppuccin Latte (light) palette.
func Latte() Palette { return aragonite.Latte() }

// Macchiato returns the Catppuccin Macchiato (medium-dark) palette.
func Macchiato() Palette { return aragonite.Macchiato() }

// Init sets the active theme used by all views.
func Init(p Palette) { current = p.Semantic() }

// Current returns the active theme.
func Current() Theme { return current }
