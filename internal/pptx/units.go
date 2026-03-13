package pptx

// EMU (English Metric Units) conversions.
// 1 inch = 914400 EMU, 1 point = 12700 EMU, 1 cm = 360000 EMU.

const (
	emuPerInch  = 914400
	emuPerPoint = 12700
	emuPerCm    = 360000

	// Standard slide dimensions (10" x 7.5" widescreen)
	slideWidth  = 10 * emuPerInch      // 9144000
	slideHeight = 6858000              // 7.5 * 914400

	// Content area with margins
	marginLeft    = emuPerInch / 2           // 0.5"
	marginTop     = emuPerInch / 2           // 0.5"
	contentWidth  = slideWidth - emuPerInch  // 10" - 1" margins
	contentHeight = slideHeight - emuPerInch // 7.5" - 1" margins
)

// Pt converts points to EMU.
func pt(points int) int64 {
	return int64(points) * emuPerPoint
}

// HalfPt converts to half-points (used for font sizes in OOXML).
func halfPt(points int) int {
	return points * 100
}
