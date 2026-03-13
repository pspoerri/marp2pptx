package mermaid

const (
	classHeaderH = emuPerInch * 2 / 5
	classMemberH = emuPerInch * 3 / 10
	classMinW    = emuPerInch * 3 / 2
	classCharW   = emuPerInch / 12
	classGapX    = emuPerInch * 3 / 4
	classGapY    = emuPerInch * 3 / 4
)

func computeClassLayout(g Graph, maxW, maxH int) Layout {
	cd := g.Class
	if len(cd.Classes) == 0 {
		return Layout{}
	}

	classIndex := make(map[string]int)
	for i, c := range cd.Classes {
		classIndex[c.Name] = i
	}

	// Compute each class node's intrinsic size
	nodeSizes := make([]struct{ w, h int }, len(cd.Classes))
	for i, c := range cd.Classes {
		w := classMinW
		textW := len(c.Name)*classCharW + emuPerInch/4
		if textW > w {
			w = textW
		}
		for _, m := range c.Members {
			mw := len(formatMember(m))*classCharW + emuPerInch/4
			if mw > w {
				w = mw
			}
		}
		h := classHeaderH + len(c.Members)*classMemberH
		if len(c.Members) == 0 {
			h = classHeaderH + classMemberH/2
		}
		nodeSizes[i] = struct{ w, h int }{w, h}
	}

	// Build directed graph for layout: edges from parent to child
	// For inheritance/realization with triangle marker, the marker side is the parent
	n := len(cd.Classes)
	adj := make([][]int, n)
	radj := make([][]int, n)
	for _, rel := range cd.Relations {
		fi, fok := classIndex[rel.From]
		ti, tok := classIndex[rel.To]
		if !fok || !tok {
			continue
		}
		// Triangle marker indicates parent side
		if rel.FromMarker == MarkerTriangle {
			// From is parent → edge from From to To (parent → child)
			adj[fi] = append(adj[fi], ti)
			radj[ti] = append(radj[ti], fi)
		} else if rel.ToMarker == MarkerTriangle {
			// To is parent → edge from To to From (parent → child)
			adj[ti] = append(adj[ti], fi)
			radj[fi] = append(radj[fi], ti)
		} else {
			// Non-hierarchical: use From → To for layout
			adj[fi] = append(adj[fi], ti)
			radj[ti] = append(radj[ti], fi)
		}
	}

	// Assign layers using BFS
	layers := make([]int, n)
	hasIncoming := make([]bool, n)
	for i := 0; i < n; i++ {
		if len(radj[i]) > 0 {
			hasIncoming[i] = true
		}
	}

	queue := make([]int, 0)
	for i := 0; i < n; i++ {
		if !hasIncoming[i] {
			queue = append(queue, i)
		}
	}
	if len(queue) == 0 {
		queue = append(queue, 0)
	}
	visited := make([]bool, n)
	for _, q := range queue {
		visited[q] = true
	}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, next := range adj[curr] {
			newLayer := layers[curr] + 1
			if newLayer > layers[next] {
				layers[next] = newLayer
			}
			if !visited[next] {
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}

	maxLayer := 0
	layerNodes := make(map[int][]int)
	for i, layer := range layers {
		layerNodes[layer] = append(layerNodes[layer], i)
		if layer > maxLayer {
			maxLayer = layer
		}
	}

	// Barycenter ordering
	reorderLayersBarycenter(layerNodes, adj, radj, maxLayer)

	numLayers := maxLayer + 1

	// Compute max width and height per layer
	layerMaxH := make([]int, numLayers)
	layerTotalW := make([]int, numLayers)
	for layer := 0; layer < numLayers; layer++ {
		for i, idx := range layerNodes[layer] {
			if nodeSizes[idx].h > layerMaxH[layer] {
				layerMaxH[layer] = nodeSizes[idx].h
			}
			layerTotalW[layer] += nodeSizes[idx].w
			if i > 0 {
				layerTotalW[layer] += classGapX
			}
		}
	}

	// Scale down if needed
	totalH := 0
	for _, h := range layerMaxH {
		totalH += h
	}
	totalH += (numLayers - 1) * classGapY
	scaleH := 1.0
	if totalH > maxH {
		scaleH = float64(maxH) / float64(totalH)
	}
	maxTotalW := 0
	for _, w := range layerTotalW {
		if w > maxTotalW {
			maxTotalW = w
		}
	}
	scaleW := 1.0
	if maxTotalW > maxW {
		scaleW = float64(maxW) / float64(maxTotalW)
	}
	scale := scaleH
	if scaleW < scale {
		scale = scaleW
	}
	if scale > 1 {
		scale = 1
	}

	// Position nodes
	classNodes := make([]ClassLayoutNode, n)
	curY := 0
	for layer := 0; layer < numLayers; layer++ {
		indices := layerNodes[layer]
		rowH := int(float64(layerMaxH[layer]) * scale)
		totalW := 0
		for i, idx := range indices {
			totalW += int(float64(nodeSizes[idx].w) * scale)
			if i > 0 {
				totalW += int(float64(classGapX) * scale)
			}
		}
		startX := (maxW - totalW) / 2
		curX := startX
		for _, idx := range indices {
			w := int(float64(nodeSizes[idx].w) * scale)
			h := int(float64(nodeSizes[idx].h) * scale)
			hdrH := int(float64(classHeaderH) * scale)
			rowMemberH := int(float64(classMemberH) * scale)
			classNodes[idx] = ClassLayoutNode{
				ClassDef: cd.Classes[idx],
				X:        curX,
				Y:        curY + (rowH-h)/2,
				W:        w,
				H:        h,
				HeaderH:  hdrH,
				RowH:     rowMemberH,
			}
			curX += w + int(float64(classGapX)*scale)
		}
		curY += rowH + int(float64(classGapY)*scale)
	}

	// Build relation layouts
	relLayouts := make([]ClassLayoutRelation, len(cd.Relations))
	for i, rel := range cd.Relations {
		var from, to ClassLayoutNode
		if fi, ok := classIndex[rel.From]; ok {
			from = classNodes[fi]
		}
		if ti, ok := classIndex[rel.To]; ok {
			to = classNodes[ti]
		}
		relLayouts[i] = ClassLayoutRelation{
			ClassRelation: rel,
			FromNode:      from,
			ToNode:        to,
		}
	}

	return Layout{
		Type: DiagramClass,
		Class: &ClassLayout{
			Classes:   classNodes,
			Relations: relLayouts,
		},
		W: maxW,
		H: maxH,
	}
}

func formatMember(m ClassMember) string {
	s := m.Visibility
	if m.IsMethod {
		s += m.Name + "()"
		if m.Type != "" {
			s += " " + m.Type
		}
	} else {
		if m.Type != "" {
			s += m.Type + " " + m.Name
		} else {
			s += m.Name
		}
	}
	return s
}
