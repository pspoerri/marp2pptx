package pptx

// connectorGeom returns the OOXML preset geometry name for a connector.
// We use straightConnector1 for all edges because curvedConnector3 always
// exits horizontally, which looks disconnected on vertical edge attachments.
func connectorGeom(_, _ int) string {
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
