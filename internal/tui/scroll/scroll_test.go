package scroll_test

import (
	"testing"

	"github.com/KyleKing/gh-sweep/internal/tui/scroll"
)

func TestWindow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		total, cursor, vis int
		wantStart, wantEnd int
	}{
		{"fits entirely", 5, 2, 10, 0, 5},
		{"zero visible falls back to full range", 20, 10, 0, 0, 20},
		{"cursor near top clamps start to zero", 20, 1, 5, 0, 5},
		{"cursor near bottom clamps end to total", 20, 19, 5, 15, 20},
		{"cursor mid-list centers the window", 20, 10, 5, 8, 13},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start, end := scroll.Window(tt.total, tt.cursor, tt.vis)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("Window(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.total, tt.cursor, tt.vis, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
