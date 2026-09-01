package config

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsePreviewWidth reads a picker.preview_width value into either a
// percentage of the picker's content width or a fixed column count. Exactly
// one of the two is non-zero on success; both are zero when raw is empty,
// which means the picker keeps its default layout.
//
// A trailing percent sign selects the proportional form. Terminals are
// measured in cells rather than pixels, so the other form is a column count.
// The percentage form is usually the better choice, because it keeps the split
// sensible across a resized window or a different display.
func ParsePreviewWidth(raw string) (percent int, columns int, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, 0, nil
	}
	if pct, ok := strings.CutSuffix(s, "%"); ok {
		n, convErr := strconv.Atoi(strings.TrimSpace(pct))
		if convErr != nil {
			return 0, 0, fmt.Errorf("%q is not a percentage such as \"50%%\"", raw)
		}
		if n < 1 || n > 100 {
			return 0, 0, fmt.Errorf("percentage must be between 1 and 100, got %d", n)
		}
		return n, 0, nil
	}
	n, convErr := strconv.Atoi(s)
	if convErr != nil {
		return 0, 0, fmt.Errorf("%q is neither a column count such as \"100\" nor a percentage such as \"50%%\"", raw)
	}
	if n < 1 {
		return 0, 0, fmt.Errorf("column count must be at least 1, got %d", n)
	}
	return 0, n, nil
}
