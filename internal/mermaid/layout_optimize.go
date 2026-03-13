package mermaid

// Graph layout optimization algorithms.
//
// Two complementary approaches are used depending on graph structure:
//
// 1. Sugiyama framework (directed/hierarchical graphs: flowcharts, state, class)
//    - Layer assignment via topological BFS (longest-path heuristic)
//    - Edge crossing minimization via barycenter ordering (8 forward+backward sweeps)
//    - Cross-axis position refinement via iterative median alignment:
//      each node is pulled toward the median position of its neighbors in
//      adjacent layers (damped at 2/3 per step), with overlap and boundary
//      constraints enforced after each pass. Converges when max displacement
//      drops below minGap/4.
//
// 2. Fruchterman-Reingold force-directed layout (undirected graphs: ER diagrams)
//    - Repulsive forces (Coulomb) between all node pairs: F = k²/d
//    - Attractive forces (Hooke) along edges: F = d/k
//    - Gravity toward center for compactness
//    - Simulated annealing with linear cooling over 200 iterations
//    - Node-size-aware overlap amplification (2× repulsion when overlapping)
//    - Boundary clamping per iteration to keep nodes within the slide
//    - Post-simulation overlap resolution and fit-to-box scaling

import "math"

// vec2 represents a 2D position for force-directed layout.
type vec2 struct{ x, y float64 }

// forceDirectedPositions computes 2D center positions for n nodes using
// the Fruchterman-Reingold force-directed algorithm. widths and heights
// give each node's dimensions (EMU). maxW and maxH define the bounding box.
func forceDirectedPositions(n int, edges [][2]int, widths, heights []int, maxW, maxH int) []vec2 {
	if n == 0 {
		return nil
	}
	if n == 1 {
		return []vec2{{float64(maxW) / 2, float64(maxH) / 2}}
	}

	pos := make([]vec2, n)

	// Initialize nodes on a circle
	cx, cy := float64(maxW)/2, float64(maxH)/2
	radius := math.Min(float64(maxW), float64(maxH)) * 0.3
	for i := 0; i < n; i++ {
		angle := 2 * math.Pi * float64(i) / float64(n)
		pos[i] = vec2{
			x: cx + radius*math.Cos(angle),
			y: cy + radius*math.Sin(angle),
		}
	}

	area := float64(maxW) * float64(maxH)
	k := math.Sqrt(area / float64(n)) // ideal edge length

	const iterations = 200
	temp := float64(maxW+maxH) / 4
	cooling := temp / iterations

	for iter := 0; iter < iterations; iter++ {
		forces := make([]vec2, n)

		// Repulsive forces between all node pairs (Coulomb's law)
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				dx := pos[i].x - pos[j].x
				dy := pos[i].y - pos[j].y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < 1 {
					dist = 1
				}
				// Amplify repulsion when nodes overlap
				minSep := float64(widths[i]+widths[j])/2 + float64(heights[i]+heights[j])/4
				repK := k
				if dist < minSep {
					repK = k * 2
				}
				force := repK * repK / dist
				fx := force * dx / dist
				fy := force * dy / dist
				forces[i].x += fx
				forces[i].y += fy
				forces[j].x -= fx
				forces[j].y -= fy
			}
		}

		// Attractive forces along edges (Hooke's law)
		for _, e := range edges {
			i, j := e[0], e[1]
			dx := pos[j].x - pos[i].x
			dy := pos[j].y - pos[i].y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < 1 {
				dist = 1
			}
			force := dist / k
			fx := force * dx / dist
			fy := force * dy / dist
			forces[i].x += fx
			forces[i].y += fy
			forces[j].x -= fx
			forces[j].y -= fy
		}

		// Gentle gravity toward center to keep layout compact
		for i := 0; i < n; i++ {
			forces[i].x += (cx - pos[i].x) * 0.01
			forces[i].y += (cy - pos[i].y) * 0.01
		}

		// Apply forces capped by temperature, then clamp to bounds
		for i := 0; i < n; i++ {
			fx, fy := forces[i].x, forces[i].y
			mag := math.Sqrt(fx*fx + fy*fy)
			if mag > temp {
				fx = fx * temp / mag
				fy = fy * temp / mag
			}
			pos[i].x += fx
			pos[i].y += fy

			// Clamp to bounding box (keep nodes fully inside)
			hw := float64(widths[i]) / 2
			hh := float64(heights[i]) / 2
			if pos[i].x-hw < 0 {
				pos[i].x = hw
			}
			if pos[i].x+hw > float64(maxW) {
				pos[i].x = float64(maxW) - hw
			}
			if pos[i].y-hh < 0 {
				pos[i].y = hh
			}
			if pos[i].y+hh > float64(maxH) {
				pos[i].y = float64(maxH) - hh
			}
		}

		temp -= cooling
		if temp < 0 {
			temp = 0
		}
	}

	return pos
}

// resolveNodeOverlaps iteratively pushes apart overlapping nodes.
// gap is the minimum required spacing between node edges.
func resolveNodeOverlaps(pos []vec2, widths, heights []int, gap int) {
	n := len(pos)
	gapF := float64(gap)

	for iter := 0; iter < 100; iter++ {
		moved := false
		for i := 0; i < n; i++ {
			for j := i + 1; j < n; j++ {
				dx := pos[j].x - pos[i].x
				dy := pos[j].y - pos[i].y
				overlapX := (float64(widths[i]+widths[j])/2 + gapF) - math.Abs(dx)
				overlapY := (float64(heights[i]+heights[j])/2 + gapF) - math.Abs(dy)

				if overlapX > 0 && overlapY > 0 {
					moved = true
					// Push apart along the axis with less overlap
					if overlapX < overlapY {
						shift := overlapX/2 + 1
						if dx >= 0 {
							pos[i].x -= shift
							pos[j].x += shift
						} else {
							pos[i].x += shift
							pos[j].x -= shift
						}
					} else {
						shift := overlapY/2 + 1
						if dy >= 0 {
							pos[i].y -= shift
							pos[j].y += shift
						} else {
							pos[i].y += shift
							pos[j].y -= shift
						}
					}
				}
			}
		}
		if !moved {
			break
		}
	}
}

// fitPositionsToBox scales and translates center positions so that all
// nodes (with their sizes) fit within maxW x maxH. Returns the scale factor.
func fitPositionsToBox(pos []vec2, widths, heights []int, maxW, maxH, padding int) float64 {
	n := len(pos)
	if n == 0 {
		return 1
	}

	// Find bounding box of all nodes
	minX := pos[0].x - float64(widths[0])/2
	maxXf := pos[0].x + float64(widths[0])/2
	minY := pos[0].y - float64(heights[0])/2
	maxYf := pos[0].y + float64(heights[0])/2

	for i := 1; i < n; i++ {
		hw := float64(widths[i]) / 2
		hh := float64(heights[i]) / 2
		if pos[i].x-hw < minX {
			minX = pos[i].x - hw
		}
		if pos[i].x+hw > maxXf {
			maxXf = pos[i].x + hw
		}
		if pos[i].y-hh < minY {
			minY = pos[i].y - hh
		}
		if pos[i].y+hh > maxYf {
			maxYf = pos[i].y + hh
		}
	}

	contentW := maxXf - minX
	contentH := maxYf - minY

	pad := float64(padding)
	availW := float64(maxW) - 2*pad
	availH := float64(maxH) - 2*pad
	if availW < 1 {
		availW = 1
	}
	if availH < 1 {
		availH = 1
	}

	scale := 1.0
	if contentW > availW {
		s := availW / contentW
		if s < scale {
			scale = s
		}
	}
	if contentH > availH {
		s := availH / contentH
		if s < scale {
			scale = s
		}
	}

	// Scale positions around content center, then translate to box center
	contentCX := (minX + maxXf) / 2
	contentCY := (minY + maxYf) / 2
	targetCX := float64(maxW) / 2
	targetCY := float64(maxH) / 2

	for i := 0; i < n; i++ {
		pos[i].x = targetCX + (pos[i].x-contentCX)*scale
		pos[i].y = targetCY + (pos[i].y-contentCY)*scale
	}

	return scale
}

// ---------------------------------------------------------------------------
// Sugiyama cross-axis position refinement
// ---------------------------------------------------------------------------

// refineCrossPositions adjusts the cross-axis positions (X for TD/BT,
// Y for LR/RL) of nodes within each Sugiyama layer using iterative
// median-based alignment with connected nodes in neighboring layers.
// This improves on simple uniform spacing by pulling connected nodes
// into vertical/horizontal alignment, reducing total edge length.
func refineCrossPositions(
	nodes []LayoutNode,
	layerNodes map[int][]int,
	adj, radj [][]int,
	maxLayer int,
	horizontal bool,
	maxSpan int,
) {
	if maxLayer == 0 {
		return
	}

	getCross := func(ln LayoutNode) int { return ln.X }
	setCross := func(ln *LayoutNode, v int) { ln.X = v }
	getSize := func(ln LayoutNode) int { return ln.W }
	minGap := nodeGapX / 4

	if horizontal {
		getCross = func(ln LayoutNode) int { return ln.Y }
		setCross = func(ln *LayoutNode, v int) { ln.Y = v }
		getSize = func(ln LayoutNode) int { return ln.H }
		minGap = nodeGapY / 4
	}

	for iter := 0; iter < 30; iter++ {
		maxDelta := 0

		// Down sweep: align each layer with predecessors
		for layer := 1; layer <= maxLayer; layer++ {
			d := refineLayerPositions(layerNodes[layer], nodes, radj,
				getCross, setCross, getSize, minGap, maxSpan)
			if d > maxDelta {
				maxDelta = d
			}
		}

		// Up sweep: align each layer with successors
		for layer := maxLayer - 1; layer >= 0; layer-- {
			d := refineLayerPositions(layerNodes[layer], nodes, adj,
				getCross, setCross, getSize, minGap, maxSpan)
			if d > maxDelta {
				maxDelta = d
			}
		}

		if maxDelta < minGap/4 {
			break
		}
	}
}

// refineLayerPositions adjusts cross-axis positions of nodes in a single
// layer toward the median position of their neighbors. Returns the
// largest position change made (for convergence detection).
func refineLayerPositions(
	indices []int,
	nodes []LayoutNode,
	neighbors [][]int,
	getCross func(LayoutNode) int,
	setCross func(*LayoutNode, int),
	getSize func(LayoutNode) int,
	minGap, maxSpan int,
) int {
	if len(indices) <= 1 {
		return 0
	}

	maxDelta := 0

	for _, idx := range indices {
		nbs := neighbors[idx]
		if len(nbs) == 0 {
			continue
		}

		// Collect center positions of connected neighbors
		centers := make([]int, len(nbs))
		for i, nb := range nbs {
			centers[i] = getCross(nodes[nb]) + getSize(nodes[nb])/2
		}
		insertionSortInts(centers)

		// Use median for robustness against outliers
		median := centers[len(centers)/2]
		target := median - getSize(nodes[idx])/2

		current := getCross(nodes[idx])
		delta := (target - current) * 2 / 3 // damped movement
		if delta != 0 {
			setCross(&nodes[idx], current+delta)
			ad := delta
			if ad < 0 {
				ad = -ad
			}
			if ad > maxDelta {
				maxDelta = ad
			}
		}
	}

	// Enforce minimum gap between consecutive nodes
	for i := 1; i < len(indices); i++ {
		prev := indices[i-1]
		curr := indices[i]
		minPos := getCross(nodes[prev]) + getSize(nodes[prev]) + minGap
		if getCross(nodes[curr]) < minPos {
			setCross(&nodes[curr], minPos)
		}
	}

	// Keep within bounding box
	last := indices[len(indices)-1]
	overflow := getCross(nodes[last]) + getSize(nodes[last]) - maxSpan
	if overflow > 0 {
		shift := overflow / len(indices)
		if shift < 1 {
			shift = 1
		}
		for _, idx := range indices {
			pos := getCross(nodes[idx]) - shift
			if pos < 0 {
				pos = 0
			}
			setCross(&nodes[idx], pos)
		}
		// Re-resolve overlaps after shift
		for i := 1; i < len(indices); i++ {
			prev := indices[i-1]
			curr := indices[i]
			minPos := getCross(nodes[prev]) + getSize(nodes[prev]) + minGap
			if getCross(nodes[curr]) < minPos {
				setCross(&nodes[curr], minPos)
			}
		}
	}

	return maxDelta
}

func insertionSortInts(a []int) {
	for i := 1; i < len(a); i++ {
		key := a[i]
		j := i - 1
		for j >= 0 && a[j] > key {
			a[j+1] = a[j]
			j--
		}
		a[j+1] = key
	}
}
