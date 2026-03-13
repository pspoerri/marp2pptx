package mermaid

// DiagramType distinguishes between diagram kinds.
type DiagramType int

const (
	DiagramFlowchart DiagramType = iota
	DiagramSequence
	DiagramClass
	DiagramState
	DiagramJourney
	DiagramER
)

// Graph represents a parsed mermaid diagram.
type Graph struct {
	Type      DiagramType
	Direction string // "LR", "RL", "TD", "BT" (flowchart only)
	Nodes     []Node // flowchart nodes
	Edges     []Edge // flowchart edges
	Sequence  *SequenceDiagram
	Class     *ClassDiagram
	State     *StateDiagram
	Journey   *JourneyDiagram
	ER        *ERDiagram
}

// Node is a diagram node with a shape and label.
type Node struct {
	ID    string
	Label string
	Shape NodeShape
}

// NodeShape defines the visual shape of a node.
type NodeShape int

const (
	ShapeRect         NodeShape = iota // [text]
	ShapeRound                         // (text)
	ShapeDiamond                       // {text}
	ShapeCircle                        // ((text))
	ShapeDoubleCircle                  // (((text)))
	ShapeStadium                       // ([text])
	ShapeSubroutine                    // [[text]]
	ShapeCylinder                      // [(text)]
	ShapeAsymmetric                    // >text]
	ShapeHexagon                       // {{text}}
	ShapeParallel                      // [/text/]
	ShapeParallelAlt                   // [\text\]
	ShapeTrapezoid                     // [/text\]
	ShapeTrapezoidAlt                  // [\text/]
)

// Edge connects two nodes.
type Edge struct {
	From  string
	To    string
	Label string
	Style EdgeStyle
	Arrow bool
}

// EdgeStyle defines the line style.
type EdgeStyle int

const (
	EdgeSolid  EdgeStyle = iota // ---
	EdgeDotted                  // -.-
	EdgeThick                   // ===
)

// SequenceDiagram holds parsed sequence diagram data.
type SequenceDiagram struct {
	Participants []Participant
	Messages     []Message
}

// Participant is an actor in a sequence diagram.
type Participant struct {
	ID    string
	Label string
}

// MessageStyle defines the arrow style for sequence messages.
type MessageStyle int

const (
	MsgSolid       MessageStyle = iota // ->
	MsgDotted                          // -->
	MsgSolidArrow                      // ->>
	MsgDottedArrow                     // -->>
	MsgSolidCross                      // -x
	MsgDottedCross                     // --x
	MsgSolidAsync                      // -)
	MsgDottedAsync                     // --)
)

// Message is an arrow between two participants.
type Message struct {
	From  string
	To    string
	Label string
	Style MessageStyle
}

// ---------------------------------------------------------------------------
// Class diagram types
// ---------------------------------------------------------------------------

// ClassDiagram holds parsed class diagram data.
type ClassDiagram struct {
	Classes   []ClassDef
	Relations []ClassRelation
}

// ClassDef is a class with members.
type ClassDef struct {
	Name    string
	Members []ClassMember
}

// ClassMember is a field or method.
type ClassMember struct {
	Visibility string // "+", "-", "#", "~"
	Name       string
	Type       string
	IsMethod   bool
}

// RelMarker defines the visual marker at a relationship endpoint.
type RelMarker int

const (
	MarkerNone     RelMarker = iota
	MarkerArrow              // > or <
	MarkerTriangle           // |> or <|
	MarkerDiamond            // *
	MarkerCircle             // o
)

// ClassRelation is a relationship between two classes.
type ClassRelation struct {
	From       string
	To         string
	Label      string
	FromMarker RelMarker
	ToMarker   RelMarker
	Dashed     bool
}

// ---------------------------------------------------------------------------
// State diagram types
// ---------------------------------------------------------------------------

// StateDiagram holds parsed state diagram data.
type StateDiagram struct {
	States      []StateDef
	Transitions []StateTransition
}

// StateType distinguishes state node kinds.
type StateType int

const (
	StateNormal StateType = iota
	StateStar             // [*] start/end
)

// StateDef is a state in a state diagram.
type StateDef struct {
	ID    string
	Label string
	Type  StateType
}

// StateTransition is a transition between states.
type StateTransition struct {
	From  string
	To    string
	Label string
}

// ---------------------------------------------------------------------------
// Journey diagram types
// ---------------------------------------------------------------------------

// JourneyDiagram holds parsed user journey data.
type JourneyDiagram struct {
	Title    string
	Sections []JourneySection
}

// JourneySection is a named group of tasks.
type JourneySection struct {
	Name  string
	Tasks []JourneyTask
}

// JourneyTask is a single task with a score.
type JourneyTask struct {
	Name   string
	Score  int
	Actors []string
}

// ---------------------------------------------------------------------------
// ER diagram types
// ---------------------------------------------------------------------------

// ERDiagram holds parsed ER diagram data.
type ERDiagram struct {
	Entities      []EREntity
	Relationships []ERRelationship
}

// EREntity is an entity with attributes.
type EREntity struct {
	Name       string
	Attributes []ERAttribute
}

// ERAttribute is an entity attribute.
type ERAttribute struct {
	Type string
	Name string
	Keys []string // "PK", "FK", "UK"
}

// ERCardinality defines relationship cardinality.
type ERCardinality int

const (
	CardExactlyOne ERCardinality = iota // ||
	CardZeroOrOne                       // o| or |o
	CardOneOrMore                       // }| or |{
	CardZeroOrMore                      // }o or o{
)

// ERRelationship is a relationship between two entities.
type ERRelationship struct {
	EntityA      string
	CardinalityA ERCardinality
	EntityB      string
	CardinalityB ERCardinality
	Label        string
	Identifying  bool // -- (identifying) vs .. (non-identifying)
}

// LayoutNode is a node with computed position for rendering.
type LayoutNode struct {
	Node
	X, Y, W, H int // position and size in EMU
}

// LayoutEdge is an edge with source/target layout info.
type LayoutEdge struct {
	Edge
	FromNode, ToNode LayoutNode
}

// Layout holds the computed positions of all nodes and edges.
type Layout struct {
	Type     DiagramType
	Nodes    []LayoutNode
	Edges    []LayoutEdge
	Sequence *SequenceLayout
	Class    *ClassLayout
	State    *StateLayout
	Journey  *JourneyLayout
	ER       *ERLayout
	W, H     int // total bounding box in EMU
}

// SequenceLayout holds computed positions for a sequence diagram.
type SequenceLayout struct {
	Participants []SeqParticipantLayout
	Messages     []SeqMessageLayout
}

// SeqParticipantLayout is a participant with position info.
type SeqParticipantLayout struct {
	Participant
	X, Y, W, H   int // box position/size
	LifelineX    int // center X of lifeline
	LifelineTopY int // top of lifeline (below box)
	LifelineBotY int // bottom of lifeline
}

// SeqMessageLayout is a message with position info.
type SeqMessageLayout struct {
	Message
	FromX, ToX int // X coordinates of arrow endpoints
	Y          int // Y coordinate of arrow
}

// ---------------------------------------------------------------------------
// Class diagram layout types
// ---------------------------------------------------------------------------

// ClassLayout holds computed positions for a class diagram.
type ClassLayout struct {
	Classes   []ClassLayoutNode
	Relations []ClassLayoutRelation
}

// ClassLayoutNode is a class with position info.
type ClassLayoutNode struct {
	ClassDef
	X, Y, W, H int
	HeaderH    int // height of the class name row
	RowH       int // height of each member row
}

// ClassLayoutRelation is a relationship with endpoint info.
type ClassLayoutRelation struct {
	ClassRelation
	FromNode, ToNode ClassLayoutNode
}

// ---------------------------------------------------------------------------
// State diagram layout types
// ---------------------------------------------------------------------------

// StateLayout holds computed positions for a state diagram.
type StateLayout struct {
	Nodes []LayoutNode
	Edges []LayoutEdge
	Stars map[string]bool // node IDs that are [*]
}

// ---------------------------------------------------------------------------
// Journey diagram layout types
// ---------------------------------------------------------------------------

// JourneyLayout holds computed positions for a journey diagram.
type JourneyLayout struct {
	Title                          string
	TitleX, TitleY, TitleW, TitleH int
	Sections                       []JourneySectionLayout
}

// JourneySectionLayout is a section with position info.
type JourneySectionLayout struct {
	Name       string
	X, Y, W, H int
	Tasks      []JourneyTaskLayout
}

// JourneyTaskLayout is a task with position info.
type JourneyTaskLayout struct {
	JourneyTask
	X, Y, W, H int
	BarW       int // width of the score bar
}

// ---------------------------------------------------------------------------
// ER diagram layout types
// ---------------------------------------------------------------------------

// ERLayout holds computed positions for an ER diagram.
type ERLayout struct {
	Entities      []EREntityLayout
	Relationships []ERRelationshipLayout
}

// EREntityLayout is an entity with position info.
type EREntityLayout struct {
	EREntity
	X, Y, W, H int
	HeaderH    int // height of the entity name row
	RowH       int // height of each attribute row
}

// ERRelationshipLayout is a relationship with endpoint info.
type ERRelationshipLayout struct {
	ERRelationship
	FromEntity, ToEntity EREntityLayout
}
