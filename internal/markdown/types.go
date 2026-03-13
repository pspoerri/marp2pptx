package markdown

// BlockKind identifies the type of content block.
type BlockKind int

const (
	BlockHeading BlockKind = iota
	BlockParagraph
	BlockList
	BlockCodeBlock
	BlockImage
	BlockThematicBreak
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
	Text   string
	Bold   bool
	Italic bool
	Code   bool
	Link   string
}

// List is an ordered or unordered list.
type List struct {
	Ordered bool
	Items   []ListItem
}

func (l List) BlockKind() BlockKind { return BlockList }

// ListItem is a single list entry.
type ListItem struct {
	Runs []Run
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
}

func (i Image) BlockKind() BlockKind { return BlockImage }

// ThematicBreak is a horizontal rule (not a slide separator).
type ThematicBreak struct{}

func (t ThematicBreak) BlockKind() BlockKind { return BlockThematicBreak }
