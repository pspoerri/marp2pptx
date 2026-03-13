package mermaid

import (
	"fmt"
	"regexp"
	"strings"
)

// Parse parses mermaid syntax into a Graph.
func Parse(source string) (Graph, error) {
	lines := strings.Split(source, "\n")

	// Detect diagram type from first non-empty, non-comment line
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		lower := strings.ToLower(line)
		switch {
		case lower == "sequencediagram":
			return parseSequenceDiagram(lines)
		case lower == "classdiagram":
			return parseClassDiagram(lines)
		case lower == "statediagram" || lower == "statediagram-v2":
			return parseStateDiagram(lines)
		case lower == "journey":
			return parseJourneyDiagram(lines)
		case lower == "erdiagram":
			return parseERDiagram(lines)
		}
		break
	}
	return parseFlowchart(lines)
}

// ---------------------------------------------------------------------------
// Flowchart parser
// ---------------------------------------------------------------------------

func parseFlowchart(lines []string) (Graph, error) {
	g := Graph{Type: DiagramFlowchart, Direction: "TD"}

	nodeMap := make(map[string]bool)
	var started bool

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}

		if !started {
			if dir, ok := parseHeader(line); ok {
				g.Direction = dir
				started = true
				continue
			}
			started = true
		}

		if isDirective(line) {
			continue
		}

		nodes, edges, err := parseLine(line)
		if err != nil {
			continue
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
	for _, prefix := range []string{"style ", "classdef ", "class ", "click ", "linkstyle ", "subgraph ", "end"} {
		if strings.HasPrefix(lower, prefix) || lower == strings.TrimSpace(prefix) {
			return true
		}
	}
	return false
}

// edgePattern matches edge operators with optional labels
var edgePattern = regexp.MustCompile(
	`^(.+?)` +
		`\s*` +
		`(` +
		`-->` +
		`|---` +
		`|-\.->` +
		`|-\.-` +
		`|==>` +
		`|===` +
		`|--\s*[^-].*?-->` +
		`|--\s*[^-].*?---` +
		`|-\.\s*.*?\.->` +
		`|==\s*.*?==>` +
		`)` +
		`\s*(.+)$`,
)

var labelInPipe = regexp.MustCompile(`^\|([^|]*)\|\s*(.+)$`)

func parseLine(line string) ([]Node, []Edge, error) {
	var allNodes []Node
	var allEdges []Edge

	remaining := line
	for {
		m := edgePattern.FindStringSubmatchIndex(remaining)
		if m == nil {
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

		if pm := labelInPipe.FindStringSubmatch(afterEdge); pm != nil {
			edge.Label = pm[1]
			afterEdge = pm[2]
		}

		toStr := afterEdge
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
	i := strings.Index(op, prefix)
	j := strings.LastIndex(op, suffix)
	if i >= 0 && j > i+len(prefix) {
		label := strings.TrimSpace(op[i+len(prefix) : j])
		return label
	}
	return ""
}

// nodeShapePatterns lists shape bracket pairs from most specific to least.
// Order matters: longer/more-specific patterns must come first.
var nodeShapePatterns = []struct {
	open  string
	close string
	shape NodeShape
}{
	{"(((", ")))", ShapeDoubleCircle},
	{"((", "))", ShapeCircle},
	{"([", "])", ShapeStadium},
	{"[[", "]]", ShapeSubroutine},
	{"[(", ")]", ShapeCylinder},
	{"{{", "}}", ShapeHexagon},
	{"[\\", "/]", ShapeTrapezoidAlt},
	{"[/", "\\]", ShapeTrapezoid},
	{"[/", "/]", ShapeParallel},
	{"[\\", "\\]", ShapeParallelAlt},
	{">", "]", ShapeAsymmetric},
	{"(", ")", ShapeRound},
	{"{", "}", ShapeDiamond},
	{"[", "]", ShapeRect},
}

func parseNodeDef(s string) (Node, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Node{}, fmt.Errorf("empty node definition")
	}

	for _, p := range nodeShapePatterns {
		idx := strings.Index(s, p.open)
		if idx > 0 && strings.HasSuffix(s, p.close) {
			id := strings.TrimSpace(s[:idx])
			label := s[idx+len(p.open) : len(s)-len(p.close)]
			label = strings.Trim(label, `"`)
			return Node{ID: id, Label: label, Shape: p.shape}, nil
		}
	}

	id := strings.TrimSpace(s)
	if id == "" {
		return Node{}, fmt.Errorf("empty node ID")
	}
	return Node{ID: id, Label: id, Shape: ShapeRect}, nil
}

// ---------------------------------------------------------------------------
// Sequence diagram parser
// ---------------------------------------------------------------------------

// seqArrowPattern matches sequence diagram message lines: Actor->>Actor: Message
// Uses word-char groups for actors to avoid consuming arrow characters.
// Longer arrow patterns must come first to avoid partial matches.
var seqArrowPattern = regexp.MustCompile(
	`^([A-Za-z]\w*)` +
		`\s*(-->>|->>|--\)|--x|-->|->|-\)|-x)\s*` +
		`([+\-]?[A-Za-z]\w*)` +
		`\s*:\s*(.*)$`,
)

func parseSequenceDiagram(lines []string) (Graph, error) {
	seq := &SequenceDiagram{}
	participantMap := make(map[string]bool)

	ensureParticipant := func(id string) {
		if !participantMap[id] {
			seq.Participants = append(seq.Participants, Participant{ID: id, Label: id})
			participantMap[id] = true
		}
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		lower := strings.ToLower(line)

		// Skip header
		if lower == "sequencediagram" {
			continue
		}

		// Skip control flow keywords
		if isSeqDirective(lower) {
			continue
		}

		// Parse participant/actor declaration
		if p, ok := parseParticipantDecl(line); ok {
			if !participantMap[p.ID] {
				seq.Participants = append(seq.Participants, p)
				participantMap[p.ID] = true
			}
			continue
		}

		// Parse note
		if strings.HasPrefix(lower, "note ") {
			continue // skip notes for now
		}

		// Parse message
		if m := seqArrowPattern.FindStringSubmatch(line); m != nil {
			from := m[1]
			arrow := m[2]
			to := m[3]
			label := strings.TrimSpace(m[4])

			// Strip activation markers (+/-)
			to = strings.TrimPrefix(to, "+")
			to = strings.TrimPrefix(to, "-")
			from = strings.TrimSuffix(from, "+")
			from = strings.TrimSuffix(from, "-")

			ensureParticipant(from)
			ensureParticipant(to)

			seq.Messages = append(seq.Messages, Message{
				From:  from,
				To:    to,
				Label: label,
				Style: parseSeqArrow(arrow),
			})
			continue
		}
	}

	if len(seq.Participants) == 0 {
		return Graph{}, fmt.Errorf("no participants found in sequence diagram")
	}

	return Graph{
		Type:     DiagramSequence,
		Sequence: seq,
	}, nil
}

func isSeqDirective(lower string) bool {
	for _, kw := range []string{
		"autonumber", "loop ", "end", "alt ", "else ", "opt ", "par ",
		"and ", "break ", "critical ", "option ", "rect ", "activate ",
		"deactivate ", "create ", "destroy ",
	} {
		if strings.HasPrefix(lower, kw) || lower == strings.TrimSpace(kw) {
			return true
		}
	}
	return false
}

var participantPattern = regexp.MustCompile(
	`^(?i)(participant|actor)\s+(\S+)(?:\s+as\s+(.+))?$`,
)

func parseParticipantDecl(line string) (Participant, bool) {
	m := participantPattern.FindStringSubmatch(line)
	if m == nil {
		return Participant{}, false
	}
	id := m[2]
	label := id
	if m[3] != "" {
		label = strings.TrimSpace(m[3])
	}
	return Participant{ID: id, Label: label}, true
}

func parseSeqArrow(arrow string) MessageStyle {
	switch arrow {
	case "->>":
		return MsgSolidArrow
	case "-->>":
		return MsgDottedArrow
	case "->":
		return MsgSolid
	case "-->":
		return MsgDotted
	case "-x":
		return MsgSolidCross
	case "--x":
		return MsgDottedCross
	case "-)":
		return MsgSolidAsync
	case "--)":
		return MsgDottedAsync
	default:
		return MsgSolidArrow
	}
}
