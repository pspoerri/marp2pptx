package mermaid

const (
	stateNodeW    = emuPerInch * 3 / 2
	stateNodeH    = emuPerInch / 2
	stateStarSize = emuPerInch / 4
	stateGapX     = emuPerInch / 2
	stateGapY     = emuPerInch * 2 / 3
)

func computeStateLayout(g Graph, maxW, maxH int) Layout {
	sd := g.State
	if len(sd.States) == 0 {
		return Layout{}
	}

	// Convert states to nodes and transitions to edges for layout
	stateIndex := make(map[string]int)
	nodes := make([]Node, len(sd.States))
	stars := make(map[string]bool)

	for i, s := range sd.States {
		stateIndex[s.ID] = i
		shape := ShapeRound
		if s.Type == StateStar {
			shape = ShapeCircle
			stars[s.ID] = true
		}
		nodes[i] = Node{ID: s.ID, Label: s.Label, Shape: shape}
	}

	edges := make([]Edge, len(sd.Transitions))
	for i, t := range sd.Transitions {
		edges[i] = Edge{From: t.From, To: t.To, Label: t.Label, Arrow: true, Style: EdgeSolid}
	}

	// Build a temporary graph for layout computation
	tempG := Graph{
		Type:      DiagramFlowchart,
		Direction: "TD",
		Nodes:     nodes,
		Edges:     edges,
	}

	fl := computeFlowchartLayout(tempG, maxW, maxH)

	// Adjust star node sizes (make them smaller)
	for i := range fl.Nodes {
		if stars[fl.Nodes[i].ID] {
			cx := fl.Nodes[i].X + fl.Nodes[i].W/2
			cy := fl.Nodes[i].Y + fl.Nodes[i].H/2
			fl.Nodes[i].W = stateStarSize
			fl.Nodes[i].H = stateStarSize
			fl.Nodes[i].X = cx - stateStarSize/2
			fl.Nodes[i].Y = cy - stateStarSize/2
		}
	}

	// Recompute edge endpoints after star node adjustment
	for i, e := range fl.Edges {
		if fi, ok := stateIndex[e.From]; ok {
			fl.Edges[i].FromNode = fl.Nodes[fi]
		}
		if ti, ok := stateIndex[e.To]; ok {
			fl.Edges[i].ToNode = fl.Nodes[ti]
		}
	}

	return Layout{
		Type: DiagramState,
		State: &StateLayout{
			Nodes: fl.Nodes,
			Edges: fl.Edges,
			Stars: stars,
		},
		W: maxW,
		H: maxH,
	}
}
