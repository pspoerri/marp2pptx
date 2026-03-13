package mermaid

const (
	emuPerInch = 914400

	// Node sizing
	nodeW    = 2 * emuPerInch // 2 inches wide
	nodeH    = emuPerInch / 2 // 0.5 inches tall
	nodeGapX = emuPerInch / 2 // horizontal gap between nodes
	nodeGapY = emuPerInch / 2 // vertical gap between nodes

	// Label sizing (must match pptx render constants)
	emuPerPoint    = emuPerInch / 72 // 12700
	labelCharW     = emuPerPoint * 8 // char width in edge labels (flowchart/state/class)
	labelBaseW     = emuPerInch / 4  // base padding on label width
	labelH         = emuPerInch / 4  // standard label height (0.25")
	erCardLabelW   = emuPerInch / 2  // ER cardinality label width (0.5")
	erRelLabelChrW = emuPerPoint * 7 // char width in ER relationship labels

	// Sequence diagram sizing
	seqParticipantW   = emuPerInch * 3 / 2 // 1.5 inches
	seqParticipantH   = emuPerInch / 2     // 0.5 inches
	seqParticipantGap = emuPerInch / 2     // gap between participants
	seqMessageGap     = emuPerInch / 2     // vertical gap between messages
)

// edgeLabelWidth returns the rendered width of an edge label.
func edgeLabelWidth(label string) int {
	if label == "" {
		return 0
	}
	return len(label)*labelCharW + labelBaseW
}

// ComputeLayout assigns positions to all nodes and edges in the graph.
// The layout fits within the given bounding box (maxW x maxH in EMU).
func ComputeLayout(g Graph, maxW, maxH int) Layout {
	switch g.Type {
	case DiagramSequence:
		if g.Sequence != nil {
			return computeSequenceLayout(g, maxW, maxH)
		}
	case DiagramClass:
		if g.Class != nil {
			return computeClassLayout(g, maxW, maxH)
		}
	case DiagramState:
		if g.State != nil {
			return computeStateLayout(g, maxW, maxH)
		}
	case DiagramJourney:
		if g.Journey != nil {
			return computeJourneyLayout(g, maxW, maxH)
		}
	case DiagramER:
		if g.ER != nil {
			return computeERLayout(g, maxW, maxH)
		}
	}
	return computeFlowchartLayout(g, maxW, maxH)
}

// ---------------------------------------------------------------------------
// Flowchart layout
// ---------------------------------------------------------------------------

func computeFlowchartLayout(g Graph, maxW, maxH int) Layout {
	if len(g.Nodes) == 0 {
		return Layout{}
	}

	nodeIndex := make(map[string]int)
	for i, n := range g.Nodes {
		nodeIndex[n.ID] = i
	}

	layers := assignLayers(g, nodeIndex)

	maxLayer := 0
	layerNodes := make(map[int][]int)
	for i, layer := range layers {
		layerNodes[layer] = append(layerNodes[layer], i)
		if layer > maxLayer {
			maxLayer = layer
		}
	}

	// Build adjacency for barycenter ordering
	adj := buildAdj(g, nodeIndex)
	radj := buildReverseAdj(g, nodeIndex)
	reorderLayersBarycenter(layerNodes, adj, radj, maxLayer)

	numLayers := maxLayer + 1
	maxPerLayer := 0
	for _, indices := range layerNodes {
		if len(indices) > maxPerLayer {
			maxPerLayer = len(indices)
		}
	}

	horizontal := g.Direction == "LR" || g.Direction == "RL"
	reverse := g.Direction == "RL" || g.Direction == "BT"

	nw, nh := fitNodeSize(numLayers, maxPerLayer, horizontal, maxW, maxH)

	// Compute maximum edge label dimensions for gap sizing.
	// Labels are rendered at edge midpoints; gaps must be large enough
	// so labels don't overlap adjacent nodes.
	maxLabelW := 0
	hasLabels := false
	for _, e := range g.Edges {
		if e.Label != "" {
			hasLabels = true
			if w := edgeLabelWidth(e.Label); w > maxLabelW {
				maxLabelW = w
			}
		}
	}

	var gapMain, gapCross int
	if horizontal {
		gapMain = (maxW - numLayers*nw) / (numLayers + 1)
		minGap := nodeGapX / 4
		// For LR/RL: labels need horizontal space between layers
		if hasLabels && maxLabelW+nodeGapX/8 > minGap {
			minGap = maxLabelW + nodeGapX/8
		}
		if gapMain < minGap {
			gapMain = minGap
		}
	} else {
		gapMain = (maxH - numLayers*nh) / (numLayers + 1)
		minGap := nodeGapY / 4
		// For TD/BT: labels need vertical space between layers
		if hasLabels && labelH+nodeGapY/8 > minGap {
			minGap = labelH + nodeGapY/8
		}
		if gapMain < minGap {
			gapMain = minGap
		}
	}

	if horizontal {
		gapCross = nodeGapY / 2
		// Labels at branch points need vertical separation
		if hasLabels && labelH+nodeGapY/8 > gapCross {
			gapCross = labelH + nodeGapY/8
		}
	} else {
		gapCross = nodeGapX / 2
		// Labels at branch points need horizontal separation
		if hasLabels && maxLabelW/2+nodeGapX/8 > gapCross {
			gapCross = maxLabelW/2 + nodeGapX/8
		}
	}

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

	// Refine cross-axis positions using median-based alignment
	crossMax := maxW
	if horizontal {
		crossMax = maxH
	}
	refineCrossPositions(layoutNodes, layerNodes, adj, radj, maxLayer, horizontal, crossMax)

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
		Type:  DiagramFlowchart,
		Nodes: layoutNodes,
		Edges: layoutEdges,
		W:     maxW,
		H:     maxH,
	}
}

func assignLayers(g Graph, nodeIndex map[string]int) []int {
	n := len(g.Nodes)
	layers := make([]int, n)
	hasIncoming := make([]bool, n)

	for _, e := range g.Edges {
		if ti, ok := nodeIndex[e.To]; ok {
			hasIncoming[ti] = true
		}
	}

	adj := make([][]int, n)
	for _, e := range g.Edges {
		fi, fok := nodeIndex[e.From]
		ti, tok := nodeIndex[e.To]
		if fok && tok {
			adj[fi] = append(adj[fi], ti)
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
			if !visited[next] {
				newLayer := layers[curr] + 1
				if newLayer > layers[next] {
					layers[next] = newLayer
				}
				visited[next] = true
				queue = append(queue, next)
			}
		}
	}

	return layers
}

func buildAdj(g Graph, nodeIndex map[string]int) [][]int {
	n := len(g.Nodes)
	adj := make([][]int, n)
	for _, e := range g.Edges {
		fi, fok := nodeIndex[e.From]
		ti, tok := nodeIndex[e.To]
		if fok && tok {
			adj[fi] = append(adj[fi], ti)
		}
	}
	return adj
}

func buildReverseAdj(g Graph, nodeIndex map[string]int) [][]int {
	n := len(g.Nodes)
	radj := make([][]int, n)
	for _, e := range g.Edges {
		fi, fok := nodeIndex[e.From]
		ti, tok := nodeIndex[e.To]
		if fok && tok {
			radj[ti] = append(radj[ti], fi)
		}
	}
	return radj
}

// reorderLayersBarycenter reorders nodes within each layer using the
// barycenter heuristic to reduce edge crossings.
func reorderLayersBarycenter(layerNodes map[int][]int, adj, radj [][]int, maxLayer int) {
	for sweep := 0; sweep < 8; sweep++ {
		// Forward sweep: order by average position of predecessors
		for layer := 1; layer <= maxLayer; layer++ {
			posMap := make(map[int]int)
			for pos, idx := range layerNodes[layer-1] {
				posMap[idx] = pos
			}
			barySort(layerNodes[layer], func(idx int) float64 {
				sum := 0.0
				count := 0
				for _, pred := range radj[idx] {
					if p, ok := posMap[pred]; ok {
						sum += float64(p)
						count++
					}
				}
				if count > 0 {
					return sum / float64(count)
				}
				return -1
			})
		}

		// Backward sweep: order by average position of successors
		for layer := maxLayer - 1; layer >= 0; layer-- {
			posMap := make(map[int]int)
			for pos, idx := range layerNodes[layer+1] {
				posMap[idx] = pos
			}
			barySort(layerNodes[layer], func(idx int) float64 {
				sum := 0.0
				count := 0
				for _, succ := range adj[idx] {
					if p, ok := posMap[succ]; ok {
						sum += float64(p)
						count++
					}
				}
				if count > 0 {
					return sum / float64(count)
				}
				return -1
			})
		}
	}
}

// barySort sorts indices by the barycenter value computed by fn.
// Indices where fn returns -1 (no neighbors) keep their relative order.
func barySort(indices []int, fn func(int) float64) {
	if len(indices) <= 1 {
		return
	}
	vals := make([]float64, len(indices))
	for i, idx := range indices {
		vals[i] = fn(idx)
	}
	// Assign default positions for unconnected nodes
	for i := range vals {
		if vals[i] < 0 {
			vals[i] = float64(i)
		}
	}
	// Insertion sort (layers are typically small)
	for i := 1; i < len(indices); i++ {
		keyIdx := indices[i]
		keyVal := vals[i]
		j := i - 1
		for j >= 0 && vals[j] > keyVal {
			indices[j+1] = indices[j]
			vals[j+1] = vals[j]
			j--
		}
		indices[j+1] = keyIdx
		vals[j+1] = keyVal
	}
}

func fitNodeSize(numLayers, maxPerLayer int, horizontal bool, maxW, maxH int) (nw, nh int) {
	nw = nodeW
	nh = nodeH

	if horizontal {
		availW := maxW - (numLayers+1)*(nodeGapX/4)
		if w := availW / numLayers; w < nw {
			nw = w
		}
		availH := maxH - (maxPerLayer+1)*(nodeGapY/4)
		if h := availH / maxPerLayer; h < nh {
			nh = h
		}
	} else {
		availH := maxH - (numLayers+1)*(nodeGapY/4)
		if h := availH / numLayers; h < nh {
			nh = h
		}
		availW := maxW - (maxPerLayer+1)*(nodeGapX/4)
		if w := availW / maxPerLayer; w < nw {
			nw = w
		}
	}

	if nw < emuPerInch/2 {
		nw = emuPerInch / 2
	}
	if nh < emuPerInch/4 {
		nh = emuPerInch / 4
	}

	return nw, nh
}

// ---------------------------------------------------------------------------
// Sequence diagram layout
// ---------------------------------------------------------------------------

func computeSequenceLayout(g Graph, maxW, maxH int) Layout {
	seq := g.Sequence
	if len(seq.Participants) == 0 {
		return Layout{}
	}

	numP := len(seq.Participants)
	numM := len(seq.Messages)

	// Compute participant box width to fit all across
	pw := seqParticipantW
	totalNeeded := numP*pw + (numP-1)*seqParticipantGap
	if totalNeeded > maxW {
		pw = (maxW - (numP-1)*seqParticipantGap) / numP
		if pw < emuPerInch/2 {
			pw = emuPerInch / 2
		}
	}

	ph := seqParticipantH
	gap := seqParticipantGap

	// Recalculate total width and center
	totalW := numP*pw + (numP-1)*gap
	startX := (maxW - totalW) / 2

	// Vertical spacing for messages
	topY := 0
	msgStartY := topY + ph + seqMessageGap/2
	msgGap := seqMessageGap
	if numM > 0 {
		availH := maxH - ph - seqMessageGap
		if computed := availH / (numM + 1); computed < msgGap {
			msgGap = computed
		}
		if msgGap < emuPerInch/6 {
			msgGap = emuPerInch / 6
		}
	}

	lifelineBot := msgStartY + numM*msgGap + seqMessageGap/2

	// Build participant index
	pIndex := make(map[string]int)
	for i, p := range seq.Participants {
		pIndex[p.ID] = i
	}

	// Position participants
	pLayouts := make([]SeqParticipantLayout, numP)
	for i, p := range seq.Participants {
		x := startX + i*(pw+gap)
		centerX := x + pw/2
		pLayouts[i] = SeqParticipantLayout{
			Participant:  p,
			X:            x,
			Y:            topY,
			W:            pw,
			H:            ph,
			LifelineX:    centerX,
			LifelineTopY: topY + ph,
			LifelineBotY: lifelineBot,
		}
	}

	// Position messages
	mLayouts := make([]SeqMessageLayout, numM)
	for i, msg := range seq.Messages {
		y := msgStartY + i*msgGap
		fromX := 0
		toX := 0
		if fi, ok := pIndex[msg.From]; ok {
			fromX = pLayouts[fi].LifelineX
		}
		if ti, ok := pIndex[msg.To]; ok {
			toX = pLayouts[ti].LifelineX
		}
		mLayouts[i] = SeqMessageLayout{
			Message: msg,
			FromX:   fromX,
			ToX:     toX,
			Y:       y,
		}
	}

	return Layout{
		Type: DiagramSequence,
		Sequence: &SequenceLayout{
			Participants: pLayouts,
			Messages:     mLayouts,
		},
		W: maxW,
		H: maxH,
	}
}
