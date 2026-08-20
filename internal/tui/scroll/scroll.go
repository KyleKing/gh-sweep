// Package scroll computes the visible line window for a cursor-navigable
// list rendered inside a fixed-height terminal viewport, so a namespace scan
// or cross-repo list that outgrows the screen scrolls instead of clipping
// silently or pushing the footer off-screen.
package scroll

const halfDivisor = 2

// Window returns the [start, end) line range to render out of total lines,
// keeping cursorLine inside the window. Visible is the number of lines
// available for the list body; a total that fits within visible returns the
// full range unchanged. A non-positive visible or total returns [0, total).
func Window(total, cursorLine, visible int) (int, int) {
	if visible <= 0 || total <= visible {
		return 0, total
	}

	start := cursorLine - visible/halfDivisor
	if start < 0 {
		start = 0
	}

	end := start + visible
	if end > total {
		end = total
		start = end - visible
	}

	return start, end
}
