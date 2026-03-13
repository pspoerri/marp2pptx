package markdown

import (
	"strings"

	"github.com/pspoerri/marp2pptx/internal/mermaid"
)

// BlockKind identifies the type of content block.
type BlockKind int

const (
	BlockHeading BlockKind = iota
	BlockParagraph
	BlockList
	BlockCodeBlock
	BlockImage
	BlockTable
	BlockThematicBreak
	BlockDefinitionList
	BlockDiagram
)

// ContentBlock is a piece of slide content.
type ContentBlock interface {
	BlockKind() BlockKind
}

// Heading is a heading (h1-h6).
type Heading struct {
	Level int
	Runs  []Run
}

func (h Heading) BlockKind() BlockKind { return BlockHeading }

// Paragraph is a block of formatted text.
type Paragraph struct {
	Runs []Run
}

func (p Paragraph) BlockKind() BlockKind { return BlockParagraph }

// Run is a span of text with formatting.
type Run struct {
	Text          string
	Bold          bool
	Italic        bool
	Code          bool
	Link          string
	Strikethrough bool
	Superscript   bool
}

// List is an ordered or unordered list.
type List struct {
	Ordered bool
	Items   []ListItem
}

func (l List) BlockKind() BlockKind { return BlockList }

// ListItem is a single list entry.
type ListItem struct {
	Runs    []Run
	Checked *bool // nil = not a task item; true/false = checkbox state
}

// CodeBlock is a fenced code block.
type CodeBlock struct {
	Language string
	Code     string
}

func (c CodeBlock) BlockKind() BlockKind { return BlockCodeBlock }

// Image represents an image reference.
type Image struct {
	AltText    string
	URL        string
	Background bool
	Position   string
	Data       []byte // Resolved image data (populated by caller)
}

func (i Image) BlockKind() BlockKind { return BlockImage }

// Table represents a markdown table.
type Table struct {
	Headers []TableCell
	Rows    [][]TableCell
}

func (t Table) BlockKind() BlockKind { return BlockTable }

// TableCell is a single cell containing formatted runs.
type TableCell struct {
	Runs []Run
}

// Text returns the concatenated plain text of the cell.
func (c TableCell) Text() string {
	var s strings.Builder
	for _, r := range c.Runs {
		s.WriteString(r.Text)
	}
	return s.String()
}

// ThematicBreak is a horizontal rule (not a slide separator).
type ThematicBreak struct{}

func (t ThematicBreak) BlockKind() BlockKind { return BlockThematicBreak }

// DefinitionList is a list of term-definition pairs.
type DefinitionList struct {
	Items []DefinitionItem
}

func (d DefinitionList) BlockKind() BlockKind { return BlockDefinitionList }

// DefinitionItem is a term with one or more descriptions.
type DefinitionItem struct {
	Term         []Run
	Descriptions [][]Run
}

// Diagram represents a parsed mermaid diagram rendered as native shapes.
type Diagram struct {
	Graph mermaid.Graph
}

func (d Diagram) BlockKind() BlockKind { return BlockDiagram }
