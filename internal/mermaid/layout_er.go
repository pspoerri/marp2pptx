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
	nodeSizes := make([]struct{ w, h int }, n)
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
		nodeSizes[i] = struct{ w, h int }{w, h}
	}

	// Order entities by connectivity (most connected first) for better layout
	order := orderByConnectivity(er, entityIndex, n)

	// Grid layout: determine columns
	cols := 2
	if n <= 2 {
		cols = n
	} else if n <= 3 {
		cols = 3
	} else if n > 6 {
		cols = 3
	}
	rows := (n + cols - 1) / cols

	// Compute column widths and row heights
	colW := make([]int, cols)
	rowH := make([]int, rows)
	for pos, idx := range order {
		c := pos % cols
		r := pos / cols
		if nodeSizes[idx].w > colW[c] {
			colW[c] = nodeSizes[idx].w
		}
		if nodeSizes[idx].h > rowH[r] {
			rowH[r] = nodeSizes[idx].h
		}
	}

	// Check total fits
	totalW := 0
	for _, w := range colW {
		totalW += w
	}
	totalW += (cols - 1) * erGapX
	totalH := 0
	for _, h := range rowH {
		totalH += h
	}
	totalH += (rows - 1) * erGapY

	scale := 1.0
	if totalW > maxW {
		s := float64(maxW) / float64(totalW)
		if s < scale {
			scale = s
		}
	}
	if totalH > maxH {
		s := float64(maxH) / float64(totalH)
		if s < scale {
			scale = s
		}
	}
	if scale > 1 {
		scale = 1
	}

	// Apply scale
	for i := range colW {
		colW[i] = int(float64(colW[i]) * scale)
	}
	for i := range rowH {
		rowH[i] = int(float64(rowH[i]) * scale)
	}
	gapX := int(float64(erGapX) * scale)
	gapY := int(float64(erGapY) * scale)
	scaledHeaderH := int(float64(erHeaderH) * scale)
	scaledAttrH := int(float64(erAttrH) * scale)

	// Compute column X offsets
	totalScaledW := 0
	for _, w := range colW {
		totalScaledW += w
	}
	totalScaledW += (cols - 1) * gapX
	startX := (maxW - totalScaledW) / 2

	colX := make([]int, cols)
	curX := startX
	for c := 0; c < cols; c++ {
		colX[c] = curX
		curX += colW[c] + gapX
	}

	// Compute row Y offsets
	totalScaledH := 0
	for _, h := range rowH {
		totalScaledH += h
	}
	totalScaledH += (rows - 1) * gapY
	startY := (maxH - totalScaledH) / 2

	rowY := make([]int, rows)
	curY := startY
	for r := 0; r < rows; r++ {
		rowY[r] = curY
		curY += rowH[r] + gapY
	}

	// Position entities
	entityNodes := make([]EREntityLayout, n)
	for pos, idx := range order {
		c := pos % cols
		r := pos / cols
		w := int(float64(nodeSizes[idx].w) * scale)
		h := int(float64(nodeSizes[idx].h) * scale)
		entityNodes[idx] = EREntityLayout{
			EREntity: er.Entities[idx],
			X:        colX[c] + (colW[c]-w)/2,
			Y:        rowY[r] + (rowH[r]-h)/2,
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

// orderByConnectivity returns entity indices ordered so that connected entities
// are placed near each other in the grid.
func orderByConnectivity(er *ERDiagram, entityIndex map[string]int, n int) []int {
	if n == 0 {
		return nil
	}

	// Count connections per entity
	connCount := make([]int, n)
	neighbors := make([][]int, n)
	for _, rel := range er.Relationships {
		ai, aok := entityIndex[rel.EntityA]
		bi, bok := entityIndex[rel.EntityB]
		if aok && bok {
			connCount[ai]++
			connCount[bi]++
			neighbors[ai] = append(neighbors[ai], bi)
			neighbors[bi] = append(neighbors[bi], ai)
		}
	}

	// BFS from the most connected entity
	start := 0
	for i := 1; i < n; i++ {
		if connCount[i] > connCount[start] {
			start = i
		}
	}

	visited := make([]bool, n)
	order := make([]int, 0, n)
	queue := []int{start}
	visited[start] = true

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		order = append(order, curr)
		for _, nb := range neighbors[curr] {
			if !visited[nb] {
				visited[nb] = true
				queue = append(queue, nb)
			}
		}
	}

	// Add any unvisited entities
	for i := 0; i < n; i++ {
		if !visited[i] {
			order = append(order, i)
		}
	}

	return order
}
