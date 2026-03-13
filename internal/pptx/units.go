package pptx

// EMU (English Metric Units) conversions.
// 1 inch = 914400 EMU, 1 point = 12700 EMU, 1 cm = 360000 EMU.

const (
	emuPerInch  = 914400
	emuPerPoint = 12700
	emuPerCm    = 360000

	// Standard slide dimensions (10" x 7.5" widescreen)
	slideWidth  = 10 * emuPerInch // 9144000
	slideHeight = 6858000         // 7.5 * 914400

	// Content area with margins
	marginLeft    = emuPerInch / 2           // 0.5"
	marginTop     = emuPerInch / 2           // 0.5"
	contentWidth  = slideWidth - emuPerInch  // 10" - 1" margins
	contentHeight = slideHeight - emuPerInch // 7.5" - 1" margins

	// Title placeholder (Title and Content layout)
	titlePlcY  = marginTop
	titlePlcCY = emuPerInch + emuPerInch/4 // 1.25"

	// Body area (below title in Title and Content layout)
	bodyAreaY  = titlePlcY + titlePlcCY
	bodyAreaCY = contentHeight - titlePlcCY

	// Center title (Title Slide layout)
	ctrTitleX  = emuPerInch                   // 1"
	ctrTitleY  = slideHeight/3 - emuPerInch/2 // ~2"
	ctrTitleCX = slideWidth - 2*emuPerInch    // 8"
	ctrTitleCY = emuPerInch + emuPerInch/4    // 1.25"

	// Subtitle (Title Slide layout)
	subTitleX  = emuPerInch + emuPerInch/2             // 1.5"
	subTitleY  = ctrTitleY + ctrTitleCY + emuPerInch/4 // below title + 0.25" gap
	subTitleCX = slideWidth - 3*emuPerInch             // 7"
	subTitleCY = emuPerInch                            // 1"
)

// Pt converts points to EMU.
func pt(points int) int64 {
	return int64(points) * emuPerPoint
}

// HalfPt converts to half-points (used for font sizes in OOXML).
func halfPt(points int) int {
	return points * 100
}
