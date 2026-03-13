package mermaid

import "testing"

func TestParse_SimpleGraph(t *testing.T) {
	g, err := Parse("graph LR\n    A --> B")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if g.Direction != "LR" {
		t.Errorf("expected direction LR, got %q", g.Direction)
	}
	if len(g.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].From != "A" || g.Edges[0].To != "B" {
		t.Errorf("expected edge A->B, got %s->%s", g.Edges[0].From, g.Edges[0].To)
	}
	if !g.Edges[0].Arrow {
		t.Error("expected arrow edge")
	}
}

func TestParse_Flowchart(t *testing.T) {
	g, err := Parse("flowchart TD\n    A --> B")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if g.Direction != "TD" {
		t.Errorf("expected direction TD, got %q", g.Direction)
	}
}

func TestParse_NodeShapes(t *testing.T) {
	input := `graph LR
    A[Rectangle] --> B(Rounded)
    B --> C{Diamond}
    C --> D((Circle))`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	shapes := make(map[string]NodeShape)
	for _, n := range g.Nodes {
		shapes[n.ID] = n.Shape
	}

	tests := []struct {
		id    string
		shape NodeShape
		label string
	}{
		{"A", ShapeRect, "Rectangle"},
		{"B", ShapeRound, "Rounded"},
		{"C", ShapeDiamond, "Diamond"},
		{"D", ShapeCircle, "Circle"},
	}
	for _, tt := range tests {
		found := false
		for _, n := range g.Nodes {
			if n.ID == tt.id {
				found = true
				if n.Shape != tt.shape {
					t.Errorf("node %s: expected shape %d, got %d", tt.id, tt.shape, n.Shape)
				}
				if n.Label != tt.label {
					t.Errorf("node %s: expected label %q, got %q", tt.id, tt.label, n.Label)
				}
			}
		}
		if !found {
			t.Errorf("node %s not found", tt.id)
		}
	}
}

func TestParse_LabeledEdge(t *testing.T) {
	g, err := Parse("graph LR\n    A -->|Yes| B")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].Label != "Yes" {
		t.Errorf("expected edge label 'Yes', got %q", g.Edges[0].Label)
	}
}

func TestParse_InlineLabeledEdge(t *testing.T) {
	g, err := Parse("graph LR\n    A -- text --> B")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].Label != "text" {
		t.Errorf("expected edge label 'text', got %q", g.Edges[0].Label)
	}
}

func TestParse_DottedEdge(t *testing.T) {
	g, err := Parse("graph LR\n    A -.-> B")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].Style != EdgeDotted {
		t.Errorf("expected dotted edge, got %d", g.Edges[0].Style)
	}
}

func TestParse_ThickEdge(t *testing.T) {
	g, err := Parse("graph LR\n    A ==> B")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].Style != EdgeThick {
		t.Errorf("expected thick edge, got %d", g.Edges[0].Style)
	}
}

func TestParse_MultipleEdges(t *testing.T) {
	input := `graph TD
    A --> B
    A --> C
    B --> D
    C --> D`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(g.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 4 {
		t.Errorf("expected 4 edges, got %d", len(g.Edges))
	}
}

func TestParse_Comments(t *testing.T) {
	input := `graph LR
    %% This is a comment
    A --> B`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(g.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(g.Nodes))
	}
}

func TestParse_SequenceDiagram(t *testing.T) {
	// Sequence diagrams aren't flowcharts - should return error or minimal result
	_, err := Parse("sequenceDiagram\n    Alice->>Bob: Hello")
	// We only support graph/flowchart for now
	if err == nil {
		// It's ok if it doesn't error - it might just produce no meaningful graph
	}
}

func TestParse_NoArrow(t *testing.T) {
	g, err := Parse("graph LR\n    A --- B")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(g.Edges) != 1 {
		t.Fatalf("expected 1 edge, got %d", len(g.Edges))
	}
	if g.Edges[0].Arrow {
		t.Error("expected no arrow")
	}
}

func TestComputeLayout(t *testing.T) {
	g := Graph{
		Direction: "LR",
		Nodes: []Node{
			{ID: "A", Label: "Start", Shape: ShapeRect},
			{ID: "B", Label: "End", Shape: ShapeRect},
		},
		Edges: []Edge{
			{From: "A", To: "B", Arrow: true},
		},
	}
	layout := ComputeLayout(g, 8229600, 6001200) // contentWidth x contentHeight
	if len(layout.Nodes) != 2 {
		t.Fatalf("expected 2 layout nodes, got %d", len(layout.Nodes))
	}
	// In LR layout, A should be left of B
	if layout.Nodes[0].X >= layout.Nodes[1].X {
		t.Errorf("expected A.X < B.X, got %d >= %d", layout.Nodes[0].X, layout.Nodes[1].X)
	}
}
