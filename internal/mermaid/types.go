package mermaid

// DiagramType distinguishes between diagram kinds.
type DiagramType int

const (
	DiagramFlowchart DiagramType = iota
	DiagramSequence
)

// Graph represents a parsed mermaid diagram.
type Graph struct {
	Type      DiagramType
	Direction string // "LR", "RL", "TD", "BT" (flowchart only)
	Nodes     []Node // flowchart nodes
	Edges     []Edge // flowchart edges
	Sequence  *SequenceDiagram
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
