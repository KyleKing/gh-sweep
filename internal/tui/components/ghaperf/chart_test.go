package ghaperf

import "testing"

func TestSparkline(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []float64
		width  int
		want   string
	}{
		{name: "empty", values: nil, width: 10, want: ""},
		{name: "zero width", values: []float64{1, 2}, width: 0, want: ""},
		{name: "flat series uses a mid-height bar", values: []float64{5, 5, 5}, width: 10, want: "▅▅▅"},
		{name: "ascending series spans low to high", values: []float64{0, 4, 8}, width: 10, want: "▁▄█"},
		{name: "truncates to the trailing width values", values: []float64{0, 100, 4, 8}, width: 2, want: "▁█"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := sparkline(tt.values, tt.width); got != tt.want {
				t.Errorf("sparkline(%v, %d) = %q, want %q", tt.values, tt.width, got, tt.want)
			}
		})
	}
}

func TestBar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    float64
		maxValue float64
		width    int
		want     string
	}{
		{name: "zero max renders empty", value: 5, maxValue: 0, width: 10, want: ""},
		{name: "zero width renders empty", value: 5, maxValue: 10, width: 0, want: ""},
		{name: "half filled", value: 5, maxValue: 10, width: 10, want: "█████░░░░░"},
		{name: "value exceeds max clamps full", value: 20, maxValue: 10, width: 4, want: "████"},
		{name: "zero value is empty bar", value: 0, maxValue: 10, width: 4, want: "░░░░"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := bar(tt.value, tt.maxValue, tt.width); got != tt.want {
				t.Errorf("bar(%v, %v, %d) = %q, want %q", tt.value, tt.maxValue, tt.width, got, tt.want)
			}
		})
	}
}
