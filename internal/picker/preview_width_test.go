package picker

import (
	"strings"
	"testing"
)

func TestPreviewLayoutUnsetKeepsDefaultCeiling(t *testing.T) {
	tests := map[string]struct {
		width       int
		wantPreview int
	}{
		"too narrow for a split": {80, 0},
		// The narrowest width that splits is 88, where half is already 44, so
		// minPreviewWidth is unreachable on this path. It only bites once
		// picker.preview_width asks for something smaller.
		"narrowest split":        {88, 44},
		"half below the ceiling": {96, 48},
		"ceiling applies":        {200, maxPreviewWidth},
		"ceiling still applies":  {400, maxPreviewWidth},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			listWidth, previewWidth := previewLayout(tc.width, previewWidthSpec{})
			if previewWidth != tc.wantPreview {
				t.Fatalf("preview = %d, want %d", previewWidth, tc.wantPreview)
			}
			if tc.wantPreview == 0 {
				if listWidth != tc.width {
					t.Fatalf("list = %d, want the full width %d", listWidth, tc.width)
				}
				return
			}
			if got := listWidth + previewWidth + 3; got != tc.width {
				t.Fatalf("list+preview+gap = %d, want %d", got, tc.width)
			}
		})
	}
}

func TestPreviewLayoutHonorsConfiguredWidth(t *testing.T) {
	tests := map[string]struct {
		width       int
		spec        previewWidthSpec
		wantPreview int
	}{
		"percentage beats the default ceiling": {200, previewWidthSpec{percent: 50}, 100},
		"small percentage":                     {200, previewWidthSpec{percent: 25}, 50},
		"fixed columns":                        {200, previewWidthSpec{columns: 120}, 120},
		"fixed columns past the ceiling":       {400, previewWidthSpec{columns: 250}, 250},
		"percentage under the floor is raised": {200, previewWidthSpec{percent: 1}, minPreviewWidth},
		"greedy percentage leaves a list":      {200, previewWidthSpec{percent: 100}, 200 - minListWidth - 3},
		"greedy columns leave a list":          {200, previewWidthSpec{columns: 500}, 200 - minListWidth - 3},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			listWidth, previewWidth := previewLayout(tc.width, tc.spec)
			if previewWidth != tc.wantPreview {
				t.Fatalf("preview = %d, want %d", previewWidth, tc.wantPreview)
			}
			if listWidth < minListWidth {
				t.Fatalf("list = %d, want at least %d", listWidth, minListWidth)
			}
			if got := listWidth + previewWidth + 3; got != tc.width {
				t.Fatalf("list+preview+gap = %d, want %d", got, tc.width)
			}
		})
	}
}

func TestPreviewLayoutConfiguredWidthStillHidesWhenTooNarrow(t *testing.T) {
	listWidth, previewWidth := previewLayout(80, previewWidthSpec{percent: 50})
	if previewWidth != 0 {
		t.Fatalf("preview = %d, want 0 below the split threshold", previewWidth)
	}
	if listWidth != 80 {
		t.Fatalf("list = %d, want the full width 80", listWidth)
	}
}

func TestPreviewWidthSpec(t *testing.T) {
	if (previewWidthSpec{}).isSet() {
		t.Fatal("zero spec should not be set")
	}
	if !(previewWidthSpec{percent: 10}).isSet() {
		t.Fatal("percent spec should be set")
	}
	if !(previewWidthSpec{columns: 10}).isSet() {
		t.Fatal("columns spec should be set")
	}
	if got := (previewWidthSpec{percent: 50}).resolve(200); got != 100 {
		t.Fatalf("resolve(200) = %d, want 100", got)
	}
	if got := (previewWidthSpec{columns: 70}).resolve(200); got != 70 {
		t.Fatalf("resolve(200) = %d, want 70", got)
	}
}

// trimPad strips the right-hand padding lipgloss adds to every visual line, so
// these cases can assert which lines survived rather than how they were padded.
func trimPad(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " ")
	}
	return strings.Join(lines, "\n")
}

func TestFixedVisualLinesAnchor(t *testing.T) {
	text := "one\ntwo\nthree\nfour\nfive"

	t.Run("top keeps the opening lines", func(t *testing.T) {
		got := trimPad(fixedVisualLines(text, 20, 3, false))
		want := "one\ntwo\n..."
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("bottom keeps the closing lines", func(t *testing.T) {
		got := trimPad(fixedVisualLines(text, 20, 3, true))
		want := "...\nfour\nfive"
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("single line has room only for the ellipsis", func(t *testing.T) {
		for _, anchorBottom := range []bool{false, true} {
			if got := trimPad(fixedVisualLines(text, 20, 1, anchorBottom)); got != "..." {
				t.Fatalf("anchorBottom=%v: got %q, want %q", anchorBottom, got, "...")
			}
		}
	})

	t.Run("content that fits is padded below either way", func(t *testing.T) {
		short := "one\ntwo"
		want := "one\ntwo\n\n"
		for _, anchorBottom := range []bool{false, true} {
			if got := trimPad(fixedVisualLines(short, 20, 4, anchorBottom)); got != want {
				t.Fatalf("anchorBottom=%v: got %q, want %q", anchorBottom, got, want)
			}
		}
	})

	t.Run("exact fit is untouched", func(t *testing.T) {
		for _, anchorBottom := range []bool{false, true} {
			if got := trimPad(fixedVisualLines(text, 20, 5, anchorBottom)); got != text {
				t.Fatalf("anchorBottom=%v: got %q, want %q", anchorBottom, got, text)
			}
		}
	})
}
