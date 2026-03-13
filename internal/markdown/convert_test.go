package markdown

import (
	"strings"
	"testing"
)

// runsText concatenates all run text for comparison.
func runsText(runs []Run) string {
	var sb strings.Builder
	for _, r := range runs {
		sb.WriteString(r.Text)
	}
	return sb.String()
}

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
	if got := runsText(h.Runs); got != "Hello World" {
		t.Errorf("expected text 'Hello World', got %q", got)
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

func TestConvert_Table(t *testing.T) {
	input := "| Name | Age |\n| --- | --- |\n| Alice | 30 |\n| Bob | 25 |"
	blocks, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d: %+v", len(blocks), blocks)
	}
	tbl, ok := blocks[0].(Table)
	if !ok {
		t.Fatalf("expected Table, got %T", blocks[0])
	}
	if len(tbl.Headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(tbl.Headers))
	}
	if tbl.Headers[0].Text() != "Name" {
		t.Errorf("expected header 'Name', got %q", tbl.Headers[0].Text())
	}
	if tbl.Headers[1].Text() != "Age" {
		t.Errorf("expected header 'Age', got %q", tbl.Headers[1].Text())
	}
	if len(tbl.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(tbl.Rows))
	}
	if tbl.Rows[0][0].Text() != "Alice" {
		t.Errorf("expected 'Alice', got %q", tbl.Rows[0][0].Text())
	}
	if tbl.Rows[1][1].Text() != "25" {
		t.Errorf("expected '25', got %q", tbl.Rows[1][1].Text())
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

func TestConvert_Strikethrough(t *testing.T) {
	blocks, err := Convert("This is ~~deleted~~ text.")
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
	hasStrike := false
	for _, r := range p.Runs {
		if r.Strikethrough && r.Text == "deleted" {
			hasStrike = true
		}
	}
	if !hasStrike {
		t.Errorf("expected a strikethrough run, got %+v", p.Runs)
	}
}

func TestConvert_TaskList(t *testing.T) {
	input := "- [x] Done\n- [ ] Todo"
	blocks, err := Convert(input)
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
	if len(l.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(l.Items))
	}
	if l.Items[0].Checked == nil || !*l.Items[0].Checked {
		t.Error("expected first item to be checked")
	}
	if l.Items[1].Checked == nil || *l.Items[1].Checked {
		t.Error("expected second item to be unchecked")
	}
}

func TestConvert_DefinitionList(t *testing.T) {
	input := "Term\n:   Description one\n:   Description two"
	blocks, err := Convert(input)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	found := false
	for _, b := range blocks {
		if dl, ok := b.(DefinitionList); ok {
			found = true
			if len(dl.Items) != 1 {
				t.Fatalf("expected 1 definition item, got %d", len(dl.Items))
			}
			if runsText(dl.Items[0].Term) != "Term" {
				t.Errorf("expected term 'Term', got %q", runsText(dl.Items[0].Term))
			}
			if len(dl.Items[0].Descriptions) != 2 {
				t.Errorf("expected 2 descriptions, got %d", len(dl.Items[0].Descriptions))
			}
		}
	}
	if !found {
		t.Errorf("expected DefinitionList block, got %+v", blocks)
	}
}

func TestConvert_Typographer(t *testing.T) {
	blocks, err := Convert(`She said "hello" and he said 'hi'...`)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(blocks) < 1 {
		t.Fatal("expected at least 1 block")
	}
	p, ok := blocks[0].(Paragraph)
	if !ok {
		t.Fatalf("expected Paragraph, got %T", blocks[0])
	}
	text := runsText(p.Runs)
	// Typographer should convert straight quotes to smart quotes and ... to ellipsis
	if strings.Contains(text, `"hello"`) {
		t.Errorf("expected typographer to convert straight quotes, got %q", text)
	}
	if strings.Contains(text, "...") {
		t.Errorf("expected typographer to convert ellipsis, got %q", text)
	}
}
