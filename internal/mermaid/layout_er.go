package mermaid

const (
	erHeaderH = emuPerInch * 2 / 5
	erAttrH   = emuPerInch * 3 / 10
	erMinW    = emuPerInch * 3 / 2
	erCharW   = emuPerInch / 12
	erGapX    = emuPerInch
	erGapY    = emuPerInch / 2
)

func computeERLayout(g Graph, maxW, maxH int) Layout {
	er := g.ER
	if len(er.Entities) == 0 {
		return Layout{}
	}

	entityIndex := make(map[string]int)
	for i, e := range er.Entities {
		entityIndex[e.Name] = i
	}

	n := len(er.Entities)

	// Compute intrinsic sizes
	widths := make([]int, n)
	heights := make([]int, n)
	for i, e := range er.Entities {
		w := erMinW
		textW := len(e.Name)*erCharW + emuPerInch/4
		if textW > w {
			w = textW
		}
		for _, a := range e.Attributes {
			aw := (len(a.Type)+len(a.Name)+4)*erCharW + emuPerInch/4
			if aw > w {
				w = aw
			}
		}
		h := erHeaderH + len(e.Attributes)*erAttrH
		if len(e.Attributes) == 0 {
			h = erHeaderH + erAttrH/2
		}
		widths[i] = w
		heights[i] = h
	}

	// Build edge list for force-directed simulation
	edges := make([][2]int, 0, len(er.Relationships))
	for _, rel := range er.Relationships {
		ai, aok := entityIndex[rel.EntityA]
		bi, bok := entityIndex[rel.EntityB]
		if aok && bok {
			edges = append(edges, [2]int{ai, bi})
		}
	}

	// Use force-directed algorithm to compute entity positions
	pos := forceDirectedPositions(n, edges, widths, heights, maxW, maxH)

	// Push apart any remaining overlaps
	resolveNodeOverlaps(pos, widths, heights, erGapX/2)

	// Scale and center to fit bounding box
	scale := fitPositionsToBox(pos, widths, heights, maxW, maxH, erGapX/2)
	if scale > 1 {
		scale = 1
	}

	scaledHeaderH := int(float64(erHeaderH) * scale)
	scaledAttrH := int(float64(erAttrH) * scale)

	// Build entity layouts from force-directed positions
	entityNodes := make([]EREntityLayout, n)
	for i := 0; i < n; i++ {
		w := int(float64(widths[i]) * scale)
		h := int(float64(heights[i]) * scale)
		entityNodes[i] = EREntityLayout{
			EREntity: er.Entities[i],
			X:        int(pos[i].x) - w/2,
			Y:        int(pos[i].y) - h/2,
			W:        w,
			H:        h,
			HeaderH:  scaledHeaderH,
			RowH:     scaledAttrH,
		}
	}

	// Build relationship layouts
	relLayouts := make([]ERRelationshipLayout, len(er.Relationships))
	for i, rel := range er.Relationships {
		var from, to EREntityLayout
		if fi, ok := entityIndex[rel.EntityA]; ok {
			from = entityNodes[fi]
		}
		if ti, ok := entityIndex[rel.EntityB]; ok {
			to = entityNodes[ti]
		}
		relLayouts[i] = ERRelationshipLayout{
			ERRelationship: rel,
			FromEntity:     from,
			ToEntity:       to,
		}
	}

	return Layout{
		Type: DiagramER,
		ER: &ERLayout{
			Entities:      entityNodes,
			Relationships: relLayouts,
		},
		W: maxW,
		H: maxH,
	}
}
