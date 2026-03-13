package mermaid

import "testing"

func TestParse_SimpleGraph(t *testing.T) {
	g, err := Parse("graph LR\n    A --> B")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if g.Type != DiagramFlowchart {
		t.Errorf("expected flowchart type, got %d", g.Type)
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

func TestParse_ExtendedNodeShapes(t *testing.T) {
	tests := []struct {
		input string
		id    string
		shape NodeShape
		label string
	}{
		{"A([Stadium])", "A", ShapeStadium, "Stadium"},
		{"B[[Subroutine]]", "B", ShapeSubroutine, "Subroutine"},
		{"C[(Database)]", "C", ShapeCylinder, "Database"},
		{"D(((Double)))", "D", ShapeDoubleCircle, "Double"},
		{"E{{Hexagon}}", "E", ShapeHexagon, "Hexagon"},
	}
	for _, tt := range tests {
		g, err := Parse("graph LR\n    " + tt.input)
		if err != nil {
			t.Fatalf("Parse %q failed: %v", tt.input, err)
		}
		if len(g.Nodes) == 0 {
			t.Fatalf("Parse %q: no nodes", tt.input)
		}
		n := g.Nodes[0]
		if n.ID != tt.id {
			t.Errorf("Parse %q: expected id %q, got %q", tt.input, tt.id, n.ID)
		}
		if n.Shape != tt.shape {
			t.Errorf("Parse %q: expected shape %d, got %d", tt.input, tt.shape, n.Shape)
		}
		if n.Label != tt.label {
			t.Errorf("Parse %q: expected label %q, got %q", tt.input, tt.label, n.Label)
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

// ---------------------------------------------------------------------------
// Sequence diagram tests
// ---------------------------------------------------------------------------

func TestParse_SequenceDiagram(t *testing.T) {
	input := `sequenceDiagram
    Alice->>Bob: Hello Bob
    Bob-->>Alice: Hi Alice`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if g.Type != DiagramSequence {
		t.Fatalf("expected sequence diagram type, got %d", g.Type)
	}
	if g.Sequence == nil {
		t.Fatal("expected non-nil Sequence")
	}
	if len(g.Sequence.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(g.Sequence.Participants))
	}
	if g.Sequence.Participants[0].ID != "Alice" {
		t.Errorf("expected first participant 'Alice', got %q", g.Sequence.Participants[0].ID)
	}
	if g.Sequence.Participants[1].ID != "Bob" {
		t.Errorf("expected second participant 'Bob', got %q", g.Sequence.Participants[1].ID)
	}
	if len(g.Sequence.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(g.Sequence.Messages))
	}
	m := g.Sequence.Messages[0]
	if m.From != "Alice" || m.To != "Bob" || m.Label != "Hello Bob" {
		t.Errorf("unexpected message: %+v", m)
	}
	if m.Style != MsgSolidArrow {
		t.Errorf("expected solid arrow, got %d", m.Style)
	}
	m2 := g.Sequence.Messages[1]
	if m2.Style != MsgDottedArrow {
		t.Errorf("expected dotted arrow, got %d", m2.Style)
	}
}

func TestParse_SequenceWithParticipants(t *testing.T) {
	input := `sequenceDiagram
    participant A as Alice
    participant B as Bob
    A->>B: Hello`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(g.Sequence.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(g.Sequence.Participants))
	}
	if g.Sequence.Participants[0].Label != "Alice" {
		t.Errorf("expected label 'Alice', got %q", g.Sequence.Participants[0].Label)
	}
	if g.Sequence.Participants[1].Label != "Bob" {
		t.Errorf("expected label 'Bob', got %q", g.Sequence.Participants[1].Label)
	}
}

func TestParse_SequenceArrowTypes(t *testing.T) {
	tests := []struct {
		arrow string
		style MessageStyle
	}{
		{"->", MsgSolid},
		{"-->", MsgDotted},
		{"->>", MsgSolidArrow},
		{"-->>", MsgDottedArrow},
		{"-x", MsgSolidCross},
		{"--x", MsgDottedCross},
		{"-)", MsgSolidAsync},
		{"--)", MsgDottedAsync},
	}
	for _, tt := range tests {
		input := "sequenceDiagram\n    A" + tt.arrow + "B: msg"
		g, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse %q failed: %v", tt.arrow, err)
		}
		if len(g.Sequence.Messages) != 1 {
			t.Fatalf("Parse %q: expected 1 message, got %d", tt.arrow, len(g.Sequence.Messages))
		}
		if g.Sequence.Messages[0].Style != tt.style {
			t.Errorf("Parse %q: expected style %d, got %d", tt.arrow, tt.style, g.Sequence.Messages[0].Style)
		}
	}
}

func TestParse_SequenceSkipsControlFlow(t *testing.T) {
	input := `sequenceDiagram
    Alice->>Bob: Hello
    loop Every minute
        Bob->>Alice: Ping
    end
    Alice-->>Bob: Done`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	// Should have 3 messages (loop/end skipped)
	if len(g.Sequence.Messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(g.Sequence.Messages))
	}
}

func TestComputeLayout_Flowchart(t *testing.T) {
	g := Graph{
		Type:      DiagramFlowchart,
		Direction: "LR",
		Nodes: []Node{
			{ID: "A", Label: "Start", Shape: ShapeRect},
			{ID: "B", Label: "End", Shape: ShapeRect},
		},
		Edges: []Edge{
			{From: "A", To: "B", Arrow: true},
		},
	}
	layout := ComputeLayout(g, 8229600, 6001200)
	if len(layout.Nodes) != 2 {
		t.Fatalf("expected 2 layout nodes, got %d", len(layout.Nodes))
	}
	if layout.Nodes[0].X >= layout.Nodes[1].X {
		t.Errorf("expected A.X < B.X, got %d >= %d", layout.Nodes[0].X, layout.Nodes[1].X)
	}
}

func TestComputeLayout_Sequence(t *testing.T) {
	g := Graph{
		Type: DiagramSequence,
		Sequence: &SequenceDiagram{
			Participants: []Participant{
				{ID: "A", Label: "Alice"},
				{ID: "B", Label: "Bob"},
			},
			Messages: []Message{
				{From: "A", To: "B", Label: "Hello", Style: MsgSolidArrow},
			},
		},
	}
	layout := ComputeLayout(g, 8229600, 6001200)
	if layout.Sequence == nil {
		t.Fatal("expected non-nil sequence layout")
	}
	if len(layout.Sequence.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(layout.Sequence.Participants))
	}
	// Alice should be to the left of Bob
	if layout.Sequence.Participants[0].X >= layout.Sequence.Participants[1].X {
		t.Error("expected Alice.X < Bob.X")
	}
	if len(layout.Sequence.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(layout.Sequence.Messages))
	}
	// Message should go from Alice's lifeline to Bob's lifeline
	m := layout.Sequence.Messages[0]
	if m.FromX >= m.ToX {
		t.Error("expected message FromX < ToX")
	}
}
