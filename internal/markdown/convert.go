package markdown

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// Convert parses markdown text and returns content blocks.
func Convert(source string) ([]ContentBlock, error) {
	src := []byte(source)
	md := goldmark.New(goldmark.WithExtensions(
		extension.GFM,
		extension.DefinitionList,
		extension.Footnote,
		extension.Typographer,
	))
	reader := text.NewReader(src)
	doc := md.Parser().Parse(reader)

	var blocks []ContentBlock
	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		b := convertNode(child, src)
		if b != nil {
			blocks = append(blocks, b...)
		}
	}
	return blocks, nil
}

func convertNode(n ast.Node, src []byte) []ContentBlock {
	switch node := n.(type) {
	case *ast.Heading:
		return []ContentBlock{Heading{
			Level: node.Level,
			Runs:  extractRuns(node, src),
		}}
	case *ast.Paragraph:
		// Check if this paragraph contains only an image
		if img := extractSingleImage(node, src); img != nil {
			return []ContentBlock{*img}
		}
		return []ContentBlock{Paragraph{
			Runs: extractRuns(node, src),
		}}
	case *ast.List:
		return []ContentBlock{convertList(node, src)}
	case *ast.FencedCodeBlock:
		lang := ""
		if node.Info != nil {
			lang = string(node.Info.Segment.Value(src))
		}
		var buf bytes.Buffer
		for i := 0; i < node.Lines().Len(); i++ {
			line := node.Lines().At(i)
			buf.Write(line.Value(src))
		}
		return []ContentBlock{CodeBlock{
			Language: lang,
			Code:     strings.TrimRight(buf.String(), "\n"),
		}}
	case *ast.ThematicBreak:
		return []ContentBlock{ThematicBreak{}}
	default:
		switch n.Kind() {
		case east.KindTable:
			return []ContentBlock{convertTable(n, src)}
		case east.KindDefinitionList:
			return []ContentBlock{convertDefinitionList(n, src)}
		case east.KindFootnoteList:
			return convertFootnoteList(n, src)
		}
		return nil
	}
}

func convertList(node *ast.List, src []byte) List {
	l := List{Ordered: node.IsOrdered()}
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if li, ok := child.(*ast.ListItem); ok {
			item := ListItem{}

			// Check for task checkbox
			if first := li.FirstChild(); first != nil {
				for c := first.FirstChild(); c != nil; c = c.NextSibling() {
					if cb, ok := c.(*east.TaskCheckBox); ok {
						checked := cb.IsChecked
						item.Checked = &checked
						break
					}
				}
			}

			// Extract runs from the list item's children
			var runs []Run
			for p := li.FirstChild(); p != nil; p = p.NextSibling() {
				if para, ok := p.(*ast.Paragraph); ok {
					runs = append(runs, extractRuns(para, src)...)
				} else if tb, ok := p.(*ast.TextBlock); ok {
					runs = append(runs, extractRuns(tb, src)...)
				}
			}
			item.Runs = runs
			l.Items = append(l.Items, item)
		}
	}
	return l
}

func convertTable(n ast.Node, src []byte) Table {
	tbl := Table{}
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		var cells []TableCell
		for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
			if cell.Kind() == east.KindTableCell {
				cells = append(cells, TableCell{Runs: extractRuns(cell, src)})
			}
		}
		if child.Kind() == east.KindTableHeader {
			tbl.Headers = cells
		} else if child.Kind() == east.KindTableRow {
			tbl.Rows = append(tbl.Rows, cells)
		}
	}
	return tbl
}

func convertDefinitionList(n ast.Node, src []byte) DefinitionList {
	dl := DefinitionList{}
	var current *DefinitionItem
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case east.KindDefinitionTerm:
			if current != nil {
				dl.Items = append(dl.Items, *current)
			}
			current = &DefinitionItem{
				Term: extractRuns(child, src),
			}
		case east.KindDefinitionDescription:
			if current == nil {
				current = &DefinitionItem{}
			}
			var runs []Run
			for p := child.FirstChild(); p != nil; p = p.NextSibling() {
				if para, ok := p.(*ast.Paragraph); ok {
					runs = append(runs, extractRuns(para, src)...)
				} else if tb, ok := p.(*ast.TextBlock); ok {
					runs = append(runs, extractRuns(tb, src)...)
				}
			}
			current.Descriptions = append(current.Descriptions, runs)
		}
	}
	if current != nil {
		dl.Items = append(dl.Items, *current)
	}
	return dl
}

func convertFootnoteList(n ast.Node, src []byte) []ContentBlock {
	var blocks []ContentBlock
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() == east.KindFootnote {
			fn, ok := child.(*east.Footnote)
			if !ok {
				continue
			}
			var runs []Run
			runs = append(runs, Run{
				Text:        fmt.Sprintf("%d. ", fn.Index),
				Superscript: true,
			})
			for p := child.FirstChild(); p != nil; p = p.NextSibling() {
				if para, ok := p.(*ast.Paragraph); ok {
					runs = append(runs, extractRuns(para, src)...)
				}
			}
			blocks = append(blocks, Paragraph{Runs: runs})
		}
	}
	return blocks
}

func extractRuns(n ast.Node, src []byte) []Run {
	var runs []Run
	walkInlines(n, src, false, false, false, false, "", &runs)
	return runs
}

func walkInlines(n ast.Node, src []byte, bold, italic, code, strikethrough bool, link string, runs *[]Run) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Text:
			t := string(node.Segment.Value(src))
			if t != "" {
				*runs = append(*runs, Run{
					Text: t, Bold: bold, Italic: italic, Code: code, Link: link,
					Strikethrough: strikethrough,
				})
			}
			if node.SoftLineBreak() {
				*runs = append(*runs, Run{Text: " "})
			}
		case *ast.String:
			t := string(node.Value)
			if t != "" {
				*runs = append(*runs, Run{
					Text: t, Bold: bold, Italic: italic, Code: code, Link: link,
					Strikethrough: strikethrough,
				})
			}
		case *ast.Emphasis:
			if node.Level == 2 {
				walkInlines(node, src, true, italic, code, strikethrough, link, runs)
			} else {
				walkInlines(node, src, bold, true, code, strikethrough, link, runs)
			}
		case *ast.CodeSpan:
			for c := node.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					text := string(t.Segment.Value(src))
					if text != "" {
						*runs = append(*runs, Run{
							Text: text, Bold: bold, Italic: italic, Code: true, Link: link,
							Strikethrough: strikethrough,
						})
					}
				}
			}
		case *ast.Link:
			dest := string(node.Destination)
			walkInlines(node, src, bold, italic, code, strikethrough, dest, runs)
		case *ast.Image:
			// Images in inline context are handled at the block level
		case *east.Strikethrough:
			walkInlines(node, src, bold, italic, code, true, link, runs)
		case *east.TaskCheckBox:
			// Handled at list item level via Checked field
		case *east.FootnoteLink:
			*runs = append(*runs, Run{
				Text:        fmt.Sprintf("%d", node.Index),
				Superscript: true,
			})
		case *east.FootnoteBacklink:
			// Skip backlinks in presentation context
		default:
			walkInlines(child, src, bold, italic, code, strikethrough, link, runs)
		}
	}
}

// imageAltText extracts alt text from an Image node's children.
func imageAltText(node *ast.Image, src []byte) string {
	var buf strings.Builder
	for c := node.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			buf.Write(t.Segment.Value(src))
		}
	}
	return buf.String()
}

// extractSingleImage checks if a paragraph contains only a single image.
func extractSingleImage(para *ast.Paragraph, src []byte) *Image {
	count := 0
	var img *Image
	for child := para.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Image:
			count++
			alt := imageAltText(node, src)
			url := string(node.Destination)
			bg := strings.HasPrefix(alt, "bg")
			position := ""
			if bg {
				parts := strings.Fields(alt)
				if len(parts) > 1 {
					position = strings.Join(parts[1:], " ")
				}
			}
			img = &Image{
				AltText:    alt,
				URL:        url,
				Background: bg,
				Position:   position,
			}
		case *ast.Text:
			if strings.TrimSpace(string(node.Segment.Value(src))) != "" {
				return nil
			}
		default:
			return nil
		}
	}
	if count == 1 {
		return img
	}
	return nil
}
