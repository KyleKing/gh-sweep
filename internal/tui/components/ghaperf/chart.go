package ghaperf

import "strings"

const sparkMidpointDivisor = 2

var sparkChars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// sparkline renders values as a run of block characters scaled between their
// own min and max, most recent last. It keeps only the trailing width values.
func sparkline(values []float64, width int) string {
	if len(values) == 0 || width <= 0 {
		return ""
	}

	if len(values) > width {
		values = values[len(values)-width:]
	}

	minV, maxV := values[0], values[0]
	for _, v := range values {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}

	span := maxV - minV

	var b strings.Builder
	for _, v := range values {
		idx := len(sparkChars) / sparkMidpointDivisor
		if span > 0 {
			idx = int((v - minV) / span * float64(len(sparkChars)-1))
		}
		b.WriteRune(sparkChars[idx])
	}

	return b.String()
}

// bar renders value as a filled/empty block bar out of width cells, scaled
// against max. A non-positive max or width renders an empty string.
func bar(value, maxValue float64, width int) string {
	if maxValue <= 0 || width <= 0 {
		return ""
	}

	filled := int(value / maxValue * float64(width))
	filled = min(max(filled, 0), width)

	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
