package picker

// previewWidthSpec is a resolved picker.preview_width setting: either a
// percentage of the picker's content width or a fixed number of columns.
// Configuration parses and validates the raw value, so this type only has to
// resolve it. The zero value means the setting was absent, in which case
// previewLayout keeps its historical behaviour of taking half the width and
// clamping it between minPreviewWidth and maxPreviewWidth.
type previewWidthSpec struct {
	percent int // 1..100 when set
	columns int // >=1 when set
}

func (s previewWidthSpec) isSet() bool {
	return s.percent > 0 || s.columns > 0
}

// resolve returns the requested preview width for a given content width,
// before previewLayout applies its own floor and its list-minimum ceiling.
func (s previewWidthSpec) resolve(width int) int {
	if s.percent > 0 {
		return width * s.percent / 100
	}
	return s.columns
}
