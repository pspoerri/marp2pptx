package markdown

import (
	"testing"
)

func TestConvert_Heading(t *testing.T) {
	blocks, err := Convert("# Hello World")
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	h, ok := blocks[0].(Heading)
	if !ok {
		t.Fatalf("expected Heading, got %T", blocks[0])
	}
	if h.Level != 1 {
		t.Errorf("expected level 1, got %d", h.Level)
	}
	if len(h.Runs) != 1 || h.Runs[0].Text != "Hello World" {
		t.Errorf("unexpected runs: %+v", h.Runs)
	}
}

func TestConvert_FormattedParagraph(t *testing.T) {
	blocks, err := Convert("This is **bold** and *italic* text.")
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	p, ok := blocks[0].(Paragraph)
	if !ok {
		t.Fatalf("expected Paragraph, got %T", blocks[0])
	}

	// Check that bold and italic runs exist
	hasBold := false
	hasItalic := false
	for _, r := range p.Runs {
		if r.Bold && r.Text == "bold" {
			hasBold = true
		}
		if r.Italic && r.Text == "italic" {
			hasItalic = true
		}
	}
	if !hasBold {
		t.Error("expected a bold run")
	}
	if !hasItalic {
		t.Error("expected an italic run")
	}
}

func TestConvert_UnorderedList(t *testing.T) {
	blocks, err := Convert("- Item one\n- Item two\n- Item three")
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	l, ok := blocks[0].(List)
	if !ok {
		t.Fatalf("expected List, got %T", blocks[0])
	}
	if l.Ordered {
		t.Error("expected unordered list")
	}
	if len(l.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(l.Items))
	}
}

func TestConvert_CodeBlock(t *testing.T) {
	input := "```go\nfmt.Println(\"hello\")\n```"
	blocks, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	cb, ok := blocks[0].(CodeBlock)
	if !ok {
		t.Fatalf("expected CodeBlock, got %T", blocks[0])
	}
	if cb.Language != "go" {
		t.Errorf("expected language 'go', got %q", cb.Language)
	}
}

func TestConvert_BackgroundImage(t *testing.T) {
	blocks, err := Convert("![bg](image.jpg)")
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	img, ok := blocks[0].(Image)
	if !ok {
		t.Fatalf("expected Image, got %T", blocks[0])
	}
	if !img.Background {
		t.Error("expected background image")
	}
	if img.URL != "image.jpg" {
		t.Errorf("expected URL 'image.jpg', got %q", img.URL)
	}
}
