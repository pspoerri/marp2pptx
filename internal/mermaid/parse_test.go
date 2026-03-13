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

// ---------------------------------------------------------------------------
// Class diagram tests
// ---------------------------------------------------------------------------

func TestParse_ClassDiagram(t *testing.T) {
	input := `classDiagram
    class Animal {
        +String name
        +int age
        +makeSound()
    }
    class Dog {
        +String breed
        +bark()
    }
    Animal <|-- Dog`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if g.Type != DiagramClass {
		t.Fatalf("expected class diagram type, got %d", g.Type)
	}
	if g.Class == nil {
		t.Fatal("expected non-nil Class")
	}
	if len(g.Class.Classes) != 2 {
		t.Fatalf("expected 2 classes, got %d", len(g.Class.Classes))
	}
	if g.Class.Classes[0].Name != "Animal" {
		t.Errorf("expected first class 'Animal', got %q", g.Class.Classes[0].Name)
	}
	if len(g.Class.Classes[0].Members) != 3 {
		t.Errorf("expected 3 members for Animal, got %d", len(g.Class.Classes[0].Members))
	}
	if len(g.Class.Relations) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(g.Class.Relations))
	}
	rel := g.Class.Relations[0]
	if rel.From != "Animal" || rel.To != "Dog" {
		t.Errorf("expected relation Animal-Dog, got %s-%s", rel.From, rel.To)
	}
	if rel.FromMarker != MarkerTriangle {
		t.Errorf("expected triangle marker on From side, got %d", rel.FromMarker)
	}
}

func TestParse_ClassRelationTypes(t *testing.T) {
	tests := []struct {
		op         string
		fromMarker RelMarker
		toMarker   RelMarker
		dashed     bool
	}{
		{"<|--", MarkerTriangle, MarkerNone, false},
		{"--|>", MarkerNone, MarkerTriangle, false},
		{"..|>", MarkerNone, MarkerTriangle, true},
		{"*--", MarkerDiamond, MarkerNone, false},
		{"--*", MarkerNone, MarkerDiamond, false},
		{"o--", MarkerCircle, MarkerNone, false},
		{"-->", MarkerNone, MarkerArrow, false},
		{"..>", MarkerNone, MarkerArrow, true},
		{"--", MarkerNone, MarkerNone, false},
	}
	for _, tt := range tests {
		input := "classDiagram\n    A " + tt.op + " B"
		g, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse %q failed: %v", tt.op, err)
		}
		if len(g.Class.Relations) != 1 {
			t.Fatalf("Parse %q: expected 1 relation, got %d", tt.op, len(g.Class.Relations))
		}
		rel := g.Class.Relations[0]
		if rel.FromMarker != tt.fromMarker {
			t.Errorf("Parse %q: expected fromMarker %d, got %d", tt.op, tt.fromMarker, rel.FromMarker)
		}
		if rel.ToMarker != tt.toMarker {
			t.Errorf("Parse %q: expected toMarker %d, got %d", tt.op, tt.toMarker, rel.ToMarker)
		}
		if rel.Dashed != tt.dashed {
			t.Errorf("Parse %q: expected dashed=%v, got %v", tt.op, tt.dashed, rel.Dashed)
		}
	}
}

func TestParse_ClassMember(t *testing.T) {
	tests := []struct {
		input      string
		visibility string
		name       string
		isMethod   bool
	}{
		{"+String name", "+", "name", false},
		{"-makeSound()", "-", "makeSound", true},
		{"#int age", "#", "age", false},
		{"~process()", "~", "process", true},
	}
	for _, tt := range tests {
		m, ok := parseClassMember(tt.input)
		if !ok {
			t.Fatalf("parseClassMember(%q) failed", tt.input)
		}
		if m.Visibility != tt.visibility {
			t.Errorf("parseClassMember(%q): expected visibility %q, got %q", tt.input, tt.visibility, m.Visibility)
		}
		if m.Name != tt.name {
			t.Errorf("parseClassMember(%q): expected name %q, got %q", tt.input, tt.name, m.Name)
		}
		if m.IsMethod != tt.isMethod {
			t.Errorf("parseClassMember(%q): expected isMethod=%v, got %v", tt.input, tt.isMethod, m.IsMethod)
		}
	}
}

// ---------------------------------------------------------------------------
// State diagram tests
// ---------------------------------------------------------------------------

func TestParse_StateDiagram(t *testing.T) {
	input := `stateDiagram-v2
    [*] --> Idle
    Idle --> Processing : start
    Processing --> Done : finish
    Done --> [*]`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if g.Type != DiagramState {
		t.Fatalf("expected state diagram type, got %d", g.Type)
	}
	if g.State == nil {
		t.Fatal("expected non-nil State")
	}
	// [*] appears twice → 2 star nodes + 3 regular states = 5 total
	if len(g.State.States) != 5 {
		t.Fatalf("expected 5 states, got %d", len(g.State.States))
	}
	if len(g.State.Transitions) != 4 {
		t.Fatalf("expected 4 transitions, got %d", len(g.State.Transitions))
	}
	// Check star nodes
	starCount := 0
	for _, s := range g.State.States {
		if s.Type == StateStar {
			starCount++
		}
	}
	if starCount != 2 {
		t.Errorf("expected 2 star states, got %d", starCount)
	}
}

func TestParse_StateDiagramLabeled(t *testing.T) {
	input := `stateDiagram-v2
    A --> B : transition`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(g.State.Transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(g.State.Transitions))
	}
	if g.State.Transitions[0].Label != "transition" {
		t.Errorf("expected label 'transition', got %q", g.State.Transitions[0].Label)
	}
}

// ---------------------------------------------------------------------------
// Journey diagram tests
// ---------------------------------------------------------------------------

func TestParse_JourneyDiagram(t *testing.T) {
	input := `journey
    title My Working Day
    section Morning
        Make coffee: 5: Me
        Commute: 2: Me, Bus
    section Work
        Write code: 5: Me`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if g.Type != DiagramJourney {
		t.Fatalf("expected journey diagram type, got %d", g.Type)
	}
	if g.Journey == nil {
		t.Fatal("expected non-nil Journey")
	}
	if g.Journey.Title != "My Working Day" {
		t.Errorf("expected title 'My Working Day', got %q", g.Journey.Title)
	}
	if len(g.Journey.Sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(g.Journey.Sections))
	}
	if g.Journey.Sections[0].Name != "Morning" {
		t.Errorf("expected section 'Morning', got %q", g.Journey.Sections[0].Name)
	}
	if len(g.Journey.Sections[0].Tasks) != 2 {
		t.Fatalf("expected 2 tasks in Morning, got %d", len(g.Journey.Sections[0].Tasks))
	}
	task := g.Journey.Sections[0].Tasks[0]
	if task.Name != "Make coffee" {
		t.Errorf("expected task name 'Make coffee', got %q", task.Name)
	}
	if task.Score != 5 {
		t.Errorf("expected score 5, got %d", task.Score)
	}
	if len(task.Actors) != 1 || task.Actors[0] != "Me" {
		t.Errorf("expected actors [Me], got %v", task.Actors)
	}
	task2 := g.Journey.Sections[0].Tasks[1]
	if len(task2.Actors) != 2 {
		t.Errorf("expected 2 actors for Commute, got %d", len(task2.Actors))
	}
}

func TestParse_JourneyScoreClamping(t *testing.T) {
	input := `journey
    section Test
        Low: 0: A
        High: 10: A`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if g.Journey.Sections[0].Tasks[0].Score != 1 {
		t.Errorf("expected score clamped to 1, got %d", g.Journey.Sections[0].Tasks[0].Score)
	}
	if g.Journey.Sections[0].Tasks[1].Score != 5 {
		t.Errorf("expected score clamped to 5, got %d", g.Journey.Sections[0].Tasks[1].Score)
	}
}

// ---------------------------------------------------------------------------
// ER diagram tests
// ---------------------------------------------------------------------------

func TestParse_ERDiagram(t *testing.T) {
	input := `erDiagram
    CUSTOMER ||--o{ ORDER : places
    ORDER ||--|{ LINE-ITEM : contains`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if g.Type != DiagramER {
		t.Fatalf("expected ER diagram type, got %d", g.Type)
	}
	if g.ER == nil {
		t.Fatal("expected non-nil ER")
	}
	if len(g.ER.Entities) != 3 {
		t.Fatalf("expected 3 entities, got %d", len(g.ER.Entities))
	}
	if len(g.ER.Relationships) != 2 {
		t.Fatalf("expected 2 relationships, got %d", len(g.ER.Relationships))
	}
	rel := g.ER.Relationships[0]
	if rel.EntityA != "CUSTOMER" || rel.EntityB != "ORDER" {
		t.Errorf("expected CUSTOMER-ORDER, got %s-%s", rel.EntityA, rel.EntityB)
	}
	if rel.CardinalityA != CardExactlyOne {
		t.Errorf("expected CardExactlyOne for A, got %d", rel.CardinalityA)
	}
	if rel.CardinalityB != CardZeroOrMore {
		t.Errorf("expected CardZeroOrMore for B, got %d", rel.CardinalityB)
	}
	if rel.Label != "places" {
		t.Errorf("expected label 'places', got %q", rel.Label)
	}
	if !rel.Identifying {
		t.Error("expected identifying relationship")
	}
}

func TestParse_ERDiagramWithAttributes(t *testing.T) {
	input := `erDiagram
    CUSTOMER {
        int id PK
        string name
    }
    CUSTOMER ||--o{ ORDER : places`
	g, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(g.ER.Entities) < 1 {
		t.Fatal("expected at least 1 entity")
	}
	cust := g.ER.Entities[0]
	if len(cust.Attributes) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(cust.Attributes))
	}
	if cust.Attributes[0].Type != "int" || cust.Attributes[0].Name != "id" {
		t.Errorf("unexpected first attribute: %+v", cust.Attributes[0])
	}
	if len(cust.Attributes[0].Keys) != 1 || cust.Attributes[0].Keys[0] != "PK" {
		t.Errorf("expected PK key, got %v", cust.Attributes[0].Keys)
	}
}

func TestParse_ERCardinalities(t *testing.T) {
	tests := []struct {
		left  string
		right string
		cardA ERCardinality
		cardB ERCardinality
	}{
		{"||", "||", CardExactlyOne, CardExactlyOne},
		{"||", "o{", CardExactlyOne, CardZeroOrMore},
		{"}|", "|{", CardOneOrMore, CardOneOrMore},
		{"o|", "|o", CardZeroOrOne, CardZeroOrOne},
	}
	for _, tt := range tests {
		input := "erDiagram\n    A " + tt.left + "--" + tt.right + " B : rel"
		g, err := Parse(input)
		if err != nil {
			t.Fatalf("Parse %q--%q failed: %v", tt.left, tt.right, err)
		}
		rel := g.ER.Relationships[0]
		if rel.CardinalityA != tt.cardA {
			t.Errorf("%q--%q: expected cardA %d, got %d", tt.left, tt.right, tt.cardA, rel.CardinalityA)
		}
		if rel.CardinalityB != tt.cardB {
			t.Errorf("%q--%q: expected cardB %d, got %d", tt.left, tt.right, tt.cardB, rel.CardinalityB)
		}
	}
}

// ---------------------------------------------------------------------------
// Layout tests for new diagram types
// ---------------------------------------------------------------------------

func TestComputeLayout_Class(t *testing.T) {
	g := Graph{
		Type: DiagramClass,
		Class: &ClassDiagram{
			Classes: []ClassDef{
				{Name: "Animal", Members: []ClassMember{{Name: "name", Type: "String", Visibility: "+"}}},
				{Name: "Dog", Members: []ClassMember{{Name: "bark", IsMethod: true, Visibility: "+"}}},
			},
			Relations: []ClassRelation{
				{From: "Animal", To: "Dog", FromMarker: MarkerTriangle},
			},
		},
	}
	layout := ComputeLayout(g, 8229600, 6001200)
	if layout.Class == nil {
		t.Fatal("expected non-nil class layout")
	}
	if len(layout.Class.Classes) != 2 {
		t.Fatalf("expected 2 classes, got %d", len(layout.Class.Classes))
	}
	// Animal (parent) should be above Dog (child)
	if layout.Class.Classes[0].Y >= layout.Class.Classes[1].Y {
		t.Error("expected Animal.Y < Dog.Y (parent above child)")
	}
}

func TestComputeLayout_State(t *testing.T) {
	g := Graph{
		Type: DiagramState,
		State: &StateDiagram{
			States: []StateDef{
				{ID: "__star_1__", Type: StateStar},
				{ID: "Idle", Label: "Idle", Type: StateNormal},
			},
			Transitions: []StateTransition{
				{From: "__star_1__", To: "Idle"},
			},
		},
	}
	layout := ComputeLayout(g, 8229600, 6001200)
	if layout.State == nil {
		t.Fatal("expected non-nil state layout")
	}
	if len(layout.State.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(layout.State.Nodes))
	}
	if !layout.State.Stars["__star_1__"] {
		t.Error("expected __star_1__ in Stars")
	}
}

func TestComputeLayout_Journey(t *testing.T) {
	g := Graph{
		Type: DiagramJourney,
		Journey: &JourneyDiagram{
			Title: "Test",
			Sections: []JourneySection{
				{Name: "S1", Tasks: []JourneyTask{
					{Name: "T1", Score: 5},
					{Name: "T2", Score: 3},
				}},
			},
		},
	}
	layout := ComputeLayout(g, 8229600, 6001200)
	if layout.Journey == nil {
		t.Fatal("expected non-nil journey layout")
	}
	if layout.Journey.Title != "Test" {
		t.Errorf("expected title 'Test', got %q", layout.Journey.Title)
	}
	if len(layout.Journey.Sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(layout.Journey.Sections))
	}
	if len(layout.Journey.Sections[0].Tasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(layout.Journey.Sections[0].Tasks))
	}
	// Task with score 5 should have larger bar than score 3
	t1 := layout.Journey.Sections[0].Tasks[0]
	t2 := layout.Journey.Sections[0].Tasks[1]
	if t1.BarW <= t2.BarW {
		t.Error("expected score-5 task bar wider than score-3")
	}
}

func TestComputeLayout_ER(t *testing.T) {
	g := Graph{
		Type: DiagramER,
		ER: &ERDiagram{
			Entities: []EREntity{
				{Name: "A"},
				{Name: "B"},
			},
			Relationships: []ERRelationship{
				{EntityA: "A", EntityB: "B", CardinalityA: CardExactlyOne, CardinalityB: CardZeroOrMore, Identifying: true, Label: "has"},
			},
		},
	}
	layout := ComputeLayout(g, 8229600, 6001200)
	if layout.ER == nil {
		t.Fatal("expected non-nil ER layout")
	}
	if len(layout.ER.Entities) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(layout.ER.Entities))
	}
	if len(layout.ER.Relationships) != 1 {
		t.Fatalf("expected 1 relationship, got %d", len(layout.ER.Relationships))
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
