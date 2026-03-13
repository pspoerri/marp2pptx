package mermaid

// Graph represents a parsed mermaid flowchart/graph.
type Graph struct {
	Direction string // "LR", "RL", "TD", "BT"
	Nodes     []Node
	Edges     []Edge
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
	ShapeRect      NodeShape = iota // [text]
	ShapeRound                      // (text)
	ShapeDiamond                    // {text}
	ShapeCircle                     // ((text))
	ShapeStadium                    // ([text])
	ShapeHexagon                    // {{text}}
	ShapeParallel                   // [/text/]
	ShapeTrapezoid                  // [/text\]
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
	Nodes []LayoutNode
	Edges []LayoutEdge
	W, H  int // total bounding box in EMU
}
