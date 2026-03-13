package mermaid

import (
	"fmt"
	"regexp"
	"strings"
)

// Parse parses mermaid graph/flowchart syntax into a Graph.
func Parse(source string) (Graph, error) {
	lines := strings.Split(source, "\n")
	g := Graph{Direction: "TD"}

	nodeMap := make(map[string]bool)
	var started bool

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		// Parse header line: graph/flowchart direction
		if !started {
			if dir, ok := parseHeader(line); ok {
				g.Direction = dir
				started = true
				continue
			}
			// If the first non-empty line isn't a header, treat as TD
			started = true
		}

		// Skip style/class directives
		if isDirective(line) {
			continue
		}

		// Try to parse as edge(s)
		nodes, edges, err := parseLine(line)
		if err != nil {
			continue // skip unparseable lines
		}

		for _, n := range nodes {
			if !nodeMap[n.ID] {
				g.Nodes = append(g.Nodes, n)
				nodeMap[n.ID] = true
			}
		}
		g.Edges = append(g.Edges, edges...)
	}

	if len(g.Nodes) == 0 {
		return g, fmt.Errorf("no nodes found in mermaid diagram")
	}
	return g, nil
}

func parseHeader(line string) (string, bool) {
	lower := strings.ToLower(line)
	for _, prefix := range []string{"graph ", "flowchart "} {
		if strings.HasPrefix(lower, prefix) {
			dir := strings.TrimSpace(line[len(prefix):])
			dir = strings.ToUpper(dir)
			switch dir {
			case "LR", "RL", "TD", "TB", "BT":
				if dir == "TB" {
					dir = "TD"
				}
				return dir, true
			default:
				return "TD", true
			}
		}
	}
	return "", false
}

func isDirective(line string) bool {
	lower := strings.ToLower(line)
	for _, prefix := range []string{"style ", "classDef ", "class ", "click ", "linkStyle ", "subgraph ", "end"} {
		if strings.HasPrefix(lower, prefix) || lower == strings.TrimSpace(prefix) {
			return true
		}
	}
	return false
}

// edgePattern matches edge operators: -->, ---, -.->,-.-,==>,===, with optional labels
var edgePattern = regexp.MustCompile(
	`^(.+?)` + // from node
		`\s*` +
		`(` +
		`-->` + // solid arrow
		`|---` + // solid line
		`|-\.->` + // dotted arrow
		`|-\.-` + // dotted line
		`|==>` + // thick arrow
		`|===` + // thick line
		`|--\s*[^-].*?-->` + // labeled arrow --text-->
		`|--\s*[^-].*?---` + // labeled line --text---
		`|-\.\s*.*?\.->` + // labeled dotted arrow -.text.->
		`|==\s*.*?==>` + // labeled thick arrow ==text==>
		`)` +
		`\s*(.+)$`, // to node
)

// labelInPipe matches |label| after an edge operator
var labelInPipe = regexp.MustCompile(`^\|([^|]*)\|\s*(.+)$`)

func parseLine(line string) ([]Node, []Edge, error) {
	// Handle chained edges: A --> B --> C
	var allNodes []Node
	var allEdges []Edge

	remaining := line
	for {
		m := edgePattern.FindStringSubmatchIndex(remaining)
		if m == nil {
			// Could be a standalone node definition
			if len(allNodes) == 0 {
				n, err := parseNodeDef(strings.TrimSpace(remaining))
				if err == nil {
					allNodes = append(allNodes, n)
				}
			}
			break
		}

		fromStr := strings.TrimSpace(remaining[m[2]:m[3]])
		edgeStr := remaining[m[4]:m[5]]
		afterEdge := strings.TrimSpace(remaining[m[6]:m[7]])

		fromNode, err := parseNodeDef(fromStr)
		if err != nil {
			return nil, nil, err
		}
		allNodes = append(allNodes, fromNode)

		edge := parseEdgeOp(edgeStr)
		edge.From = fromNode.ID

		// Check for |label| after edge
		if pm := labelInPipe.FindStringSubmatch(afterEdge); pm != nil {
			edge.Label = pm[1]
			afterEdge = pm[2]
		}

		// Parse the "to" side - could be a node followed by another edge
		toStr := afterEdge
		// Find next edge in the remaining string
		nextM := edgePattern.FindStringSubmatchIndex(afterEdge)
		if nextM != nil {
			toStr = strings.TrimSpace(afterEdge[:nextM[4]])
		}

		toNode, err := parseNodeDef(toStr)
		if err != nil {
			return nil, nil, err
		}
		allNodes = append(allNodes, toNode)
		edge.To = toNode.ID
		allEdges = append(allEdges, edge)

		if nextM == nil {
			break
		}
		// Continue parsing the chain from the "to" node
		remaining = afterEdge
	}

	return allNodes, allEdges, nil
}

func parseEdgeOp(op string) Edge {
	e := Edge{Arrow: true, Style: EdgeSolid}

	switch {
	case strings.Contains(op, "==>"):
		e.Style = EdgeThick
		e.Arrow = true
		e.Label = extractInlineLabel(op, "==", "==>")
	case strings.Contains(op, "==="):
		e.Style = EdgeThick
		e.Arrow = false
		e.Label = extractInlineLabel(op, "==", "===")
	case strings.Contains(op, ".->"):
		e.Style = EdgeDotted
		e.Arrow = true
		e.Label = extractInlineLabel(op, "-.", ".->")
	case strings.Contains(op, "-.-"):
		e.Style = EdgeDotted
		e.Arrow = false
		e.Label = extractInlineLabel(op, "-.", ".-")
	case strings.Contains(op, "-->"):
		e.Style = EdgeSolid
		e.Arrow = true
		e.Label = extractInlineLabel(op, "--", "-->")
	case strings.Contains(op, "---"):
		e.Style = EdgeSolid
		e.Arrow = false
		e.Label = extractInlineLabel(op, "--", "---")
	}

	return e
}

func extractInlineLabel(op, prefix, suffix string) string {
	// e.g. "-- text -->" -> "text"
	i := strings.Index(op, prefix)
	j := strings.LastIndex(op, suffix)
	if i >= 0 && j > i+len(prefix) {
		label := strings.TrimSpace(op[i+len(prefix) : j])
		return label
	}
	return ""
}

// nodeDefPattern parses node definitions like: A, A[text], A(text), A{text}, A((text)), A([text])
var nodeShapePatterns = []struct {
	open  string
	close string
	shape NodeShape
}{
	{"((", "))", ShapeCircle},
	{"([", "])", ShapeStadium},
	{"{{", "}}", ShapeHexagon},
	{"[/", "\\]", ShapeTrapezoid},
	{"[/", "/]", ShapeParallel},
	{"(", ")", ShapeRound},
	{"{", "}", ShapeDiamond},
	{"[", "]", ShapeRect},
}

func parseNodeDef(s string) (Node, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Node{}, fmt.Errorf("empty node definition")
	}

	// Try each shape pattern
	for _, p := range nodeShapePatterns {
		idx := strings.Index(s, p.open)
		if idx > 0 && strings.HasSuffix(s, p.close) {
			id := strings.TrimSpace(s[:idx])
			label := s[idx+len(p.open) : len(s)-len(p.close)]
			label = strings.Trim(label, `"`)
			return Node{ID: id, Label: label, Shape: p.shape}, nil
		}
	}

	// Plain node ID (no shape brackets)
	// Validate it's a reasonable identifier
	id := strings.TrimSpace(s)
	if id == "" {
		return Node{}, fmt.Errorf("empty node ID")
	}
	return Node{ID: id, Label: id, Shape: ShapeRect}, nil
}
