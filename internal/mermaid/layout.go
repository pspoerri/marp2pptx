package mermaid

const (
	emuPerInch = 914400

	// Node sizing
	nodeW    = 2 * emuPerInch // 2 inches wide
	nodeH    = emuPerInch / 2 // 0.5 inches tall
	nodeGapX = emuPerInch / 2 // horizontal gap between nodes
	nodeGapY = emuPerInch / 2 // vertical gap between nodes
)

// ComputeLayout assigns positions to all nodes and edges in the graph.
// The layout fits within the given bounding box (maxW x maxH in EMU).
func ComputeLayout(g Graph, maxW, maxH int) Layout {
	if len(g.Nodes) == 0 {
		return Layout{}
	}

	// Build adjacency info
	nodeIndex := make(map[string]int)
	for i, n := range g.Nodes {
		nodeIndex[n.ID] = i
	}

	// Assign layers using longest-path from sources
	layers := assignLayers(g, nodeIndex)

	// Group nodes by layer
	maxLayer := 0
	layerNodes := make(map[int][]int) // layer -> node indices
	for i, layer := range layers {
		layerNodes[layer] = append(layerNodes[layer], i)
		if layer > maxLayer {
			maxLayer = layer
		}
	}

	numLayers := maxLayer + 1
	maxPerLayer := 0
	for _, indices := range layerNodes {
		if len(indices) > maxPerLayer {
			maxPerLayer = len(indices)
		}
	}

	horizontal := g.Direction == "LR" || g.Direction == "RL"
	reverse := g.Direction == "RL" || g.Direction == "BT"

	// Compute node size to fit within bounds
	nw, nh := fitNodeSize(numLayers, maxPerLayer, horizontal, maxW, maxH)

	// Compute gaps
	var gapMain, gapCross int
	if horizontal {
		gapMain = (maxW - numLayers*nw) / (numLayers + 1)
		if gapMain < nodeGapX/4 {
			gapMain = nodeGapX / 4
		}
	} else {
		gapMain = (maxH - numLayers*nh) / (numLayers + 1)
		if gapMain < nodeGapY/4 {
			gapMain = nodeGapY / 4
		}
	}

	if horizontal {
		gapCross = nodeGapY / 2
	} else {
		gapCross = nodeGapX / 2
	}

	// Position nodes
	layoutNodes := make([]LayoutNode, len(g.Nodes))
	for layer := 0; layer <= maxLayer; layer++ {
		indices := layerNodes[layer]
		count := len(indices)

		for pos, idx := range indices {
			var x, y int
			actualLayer := layer
			if reverse {
				actualLayer = maxLayer - layer
			}

			if horizontal {
				x = gapMain + actualLayer*(nw+gapMain)
				totalCross := count*nh + (count-1)*gapCross
				startY := (maxH - totalCross) / 2
				y = startY + pos*(nh+gapCross)
			} else {
				y = gapMain + actualLayer*(nh+gapMain)
				totalCross := count*nw + (count-1)*gapCross
				startX := (maxW - totalCross) / 2
				x = startX + pos*(nw+gapCross)
			}

			layoutNodes[idx] = LayoutNode{
				Node: g.Nodes[idx],
				X:    x, Y: y,
				W: nw, H: nh,
			}
		}
	}

	// Build layout edges with node references
	layoutEdges := make([]LayoutEdge, len(g.Edges))
	for i, e := range g.Edges {
		var from, to LayoutNode
		if fi, ok := nodeIndex[e.From]; ok {
			from = layoutNodes[fi]
		}
		if ti, ok := nodeIndex[e.To]; ok {
			to = layoutNodes[ti]
		}
		layoutEdges[i] = LayoutEdge{
			Edge:     e,
			FromNode: from,
			ToNode:   to,
		}
	}

	return Layout{
		Nodes: layoutNodes,
		Edges: layoutEdges,
		W:     maxW,
		H:     maxH,
	}
}

// assignLayers uses BFS from source nodes (no incoming edges) to assign layers.
func assignLayers(g Graph, nodeIndex map[string]int) []int {
	n := len(g.Nodes)
	layers := make([]int, n)
	hasIncoming := make([]bool, n)

	for _, e := range g.Edges {
		if ti, ok := nodeIndex[e.To]; ok {
			hasIncoming[ti] = true
		}
	}

	// Build adjacency list
	adj := make([][]int, n)
	for _, e := range g.Edges {
		fi, fok := nodeIndex[e.From]
		ti, tok := nodeIndex[e.To]
		if fok && tok {
			adj[fi] = append(adj[fi], ti)
		}
	}

	// BFS from sources
	queue := make([]int, 0)
	for i := 0; i < n; i++ {
		if !hasIncoming[i] {
			queue = append(queue, i)
			layers[i] = 0
		}
	}

	// If no sources found (cycle), start from first node
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

	return layers
}

// fitNodeSize computes node width and height to fit within the bounding box.
func fitNodeSize(numLayers, maxPerLayer int, horizontal bool, maxW, maxH int) (nw, nh int) {
	nw = nodeW
	nh = nodeH

	if horizontal {
		// layers go along X, nodes per layer along Y
		availW := maxW - (numLayers+1)*(nodeGapX/4)
		if w := availW / numLayers; w < nw {
			nw = w
		}
		availH := maxH - (maxPerLayer+1)*(nodeGapY/4)
		if h := availH / maxPerLayer; h < nh {
			nh = h
		}
	} else {
		// layers go along Y, nodes per layer along X
		availH := maxH - (numLayers+1)*(nodeGapY/4)
		if h := availH / numLayers; h < nh {
			nh = h
		}
		availW := maxW - (maxPerLayer+1)*(nodeGapX/4)
		if w := availW / maxPerLayer; w < nw {
			nw = w
		}
	}

	// Enforce minimums
	if nw < emuPerInch/2 {
		nw = emuPerInch / 2
	}
	if nh < emuPerInch/4 {
		nh = emuPerInch / 4
	}

	return nw, nh
}
