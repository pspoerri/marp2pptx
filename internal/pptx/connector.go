package pptx

// connectorGeom returns the OOXML preset geometry name for a connector.
// It uses curvedConnector3 when the edge is significantly diagonal,
// and straightConnector1 for mostly-axis-aligned edges.
func connectorGeom(cx, cy int) string {
	if cx < 0 {
		cx = -cx
	}
	if cy < 0 {
		cy = -cy
	}
	threshold := emuPerInch / 8
	if cx > threshold && cy > threshold {
		mn, mx := cx, cy
		if mn > mx {
			mn, mx = mx, mn
		}
		if mn*3 > mx {
			return "curvedConnector3"
		}
	}
	return "straightConnector1"
}

// labelOffset computes a perpendicular displacement for a label placed at the
// midpoint of an edge so it doesn't overlap the connector line.
// Returns (dx, dy) offsets to apply to the label position.
func labelOffset(edgeDX, edgeDY, labelW, labelH int) (int, int) {
	margin := emuPerInch / 16
	adx := edgeDX
	if adx < 0 {
		adx = -adx
	}
	ady := edgeDY
	if ady < 0 {
		ady = -ady
	}

	if ady >= adx {
		// Mostly vertical edge → offset label to the right
		return margin + labelW/4, 0
	}
	// Mostly horizontal edge → offset label above
	return 0, -(labelH + margin)
}
