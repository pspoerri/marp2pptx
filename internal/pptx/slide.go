package pptx

import (
	"fmt"
	"strings"

	"github.com/pascal/marp2pptx/internal/markdown"
)

// headingFontSizes maps heading levels to font sizes in points.
var headingFontSizes = map[int]int{
	1: 44,
	2: 32,
	3: 28,
	4: 24,
	5: 20,
	6: 18,
}

// segment is a group of content blocks that share a shape.
// Consecutive non-table blocks form one segment; each table is its own segment.
type segment struct {
	isTable bool
	blocks  []markdown.ContentBlock
	table   markdown.Table
}

// splitSegments groups blocks into text segments and table segments.
func splitSegments(blocks []markdown.ContentBlock) []segment {
	var segments []segment
	var textBlocks []markdown.ContentBlock

	for _, block := range blocks {
		if tbl, ok := block.(markdown.Table); ok {
			if len(textBlocks) > 0 {
				segments = append(segments, segment{blocks: textBlocks})
				textBlocks = nil
			}
			segments = append(segments, segment{isTable: true, table: tbl})
		} else {
			textBlocks = append(textBlocks, block)
		}
	}
	if len(textBlocks) > 0 {
		segments = append(segments, segment{blocks: textBlocks})
	}
	return segments
}

// generateSlideXML produces the XML for a single slide.
func generateSlideXML(blocks []markdown.ContentBlock, bgColor string) string {
	segments := splitSegments(blocks)
	if len(segments) == 0 {
		segments = []segment{{blocks: nil}}
	}

	segHeight := contentHeight / len(segments)

	var shapes strings.Builder
	shapeID := 2
	for i, seg := range segments {
		y := marginTop + i*segHeight
		if seg.isTable {
			shapes.WriteString(renderTableFrame(seg.table, shapeID, marginLeft, y, contentWidth, segHeight))
		} else {
			shapes.WriteString(renderTextShape(seg.blocks, shapeID, marginLeft, y, contentWidth, segHeight))
		}
		shapeID++
	}

	bgXML := ""
	if bgColor != "" {
		bgXML = fmt.Sprintf(`<p:bg><p:bgPr><a:solidFill><a:srgbClr val="%s"/></a:solidFill><a:effectLst/></p:bgPr></p:bg>`, strings.TrimPrefix(bgColor, "#"))
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
       xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
       xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>%s
    <p:spTree>
      <p:nvGrpSpPr>
        <p:cNvPr id="1" name=""/>
        <p:cNvGrpSpPr/>
        <p:nvPr/>
      </p:nvGrpSpPr>
      <p:grpSpPr>
        <a:xfrm>
          <a:off x="0" y="0"/>
          <a:ext cx="0" cy="0"/>
          <a:chOff x="0" y="0"/>
          <a:chExt cx="0" cy="0"/>
        </a:xfrm>
      </p:grpSpPr>
%s    </p:spTree>
  </p:cSld>
</p:sld>`, bgXML, shapes.String())
}

// renderTextShape renders non-table blocks into a text shape.
func renderTextShape(blocks []markdown.ContentBlock, id, x, y, cx, cy int) string {
	var body strings.Builder
	for _, block := range blocks {
		switch b := block.(type) {
		case markdown.Heading:
			body.WriteString(renderHeading(b))
		case markdown.Paragraph:
			body.WriteString(renderParagraph(b.Runs, 0, false))
		case markdown.List:
			body.WriteString(renderList(b))
		case markdown.CodeBlock:
			body.WriteString(renderCodeBlock(b))
		}
	}

	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Content"/>
          <p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0" anchor="t"/>
          <a:lstStyle/>
%s        </p:txBody>
      </p:sp>
`, id, x, y, cx, cy, body.String())
}

// renderTableFrame renders a table as a PPTX graphicFrame.
func renderTableFrame(tbl markdown.Table, id, x, y, cx, cy int) string {
	numCols := len(tbl.Headers)
	if numCols == 0 {
		return ""
	}
	colWidth := cx / numCols
	numRows := 1 + len(tbl.Rows)
	rowHeight := cy / numRows

	var grid strings.Builder
	for i := 0; i < numCols; i++ {
		grid.WriteString(fmt.Sprintf(`              <a:gridCol w="%d"/>
`, colWidth))
	}

	var rows strings.Builder

	// Header row
	rows.WriteString(fmt.Sprintf(`            <a:tr h="%d">
`, rowHeight))
	for _, cell := range tbl.Headers {
		rows.WriteString(renderTableCell(cell, true))
	}
	rows.WriteString(`            </a:tr>
`)

	// Data rows
	for _, row := range tbl.Rows {
		rows.WriteString(fmt.Sprintf(`            <a:tr h="%d">
`, rowHeight))
		for _, cell := range row {
			rows.WriteString(renderTableCell(cell, false))
		}
		rows.WriteString(`            </a:tr>
`)
	}

	return fmt.Sprintf(`      <p:graphicFrame>
        <p:nvGraphicFramePr>
          <p:cNvPr id="%d" name="Table %d"/>
          <p:cNvGraphicFramePr><a:graphicFrameLocks noGrp="1"/></p:cNvGraphicFramePr>
          <p:nvPr/>
        </p:nvGraphicFramePr>
        <p:xfrm>
          <a:off x="%d" y="%d"/>
          <a:ext cx="%d" cy="%d"/>
        </p:xfrm>
        <a:graphic>
          <a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/table">
            <a:tbl>
              <a:tblPr firstRow="1" bandRow="1">
                <a:tblStyle val="{5C22544A-7EE6-4342-B048-85BDC9FD1C3A}"/>
              </a:tblPr>
              <a:tblGrid>
%s              </a:tblGrid>
%s            </a:tbl>
          </a:graphicData>
        </a:graphic>
      </p:graphicFrame>
`, id, id, x, y, cx, cy, grid.String(), rows.String())
}

// renderTableCell renders a single table cell.
func renderTableCell(cell markdown.TableCell, header bool) string {
	var runs strings.Builder
	for _, r := range cell.Runs {
		fontSize := 14
		bold := r.Bold || header
		attrs := fmt.Sprintf(` sz="%d" dirty="0"`, halfPt(fontSize))
		if bold {
			attrs += ` b="1"`
		}
		if r.Italic {
			attrs += ` i="1"`
		}
		runs.WriteString(fmt.Sprintf(`                    <a:r><a:rPr lang="en-US"%s/><a:t>%s</a:t></a:r>
`, attrs, escapeXML(r.Text)))
	}

	return fmt.Sprintf(`              <a:tc>
                <a:txBody>
                  <a:bodyPr/>
                  <a:lstStyle/>
                  <a:p>
%s                  </a:p>
                </a:txBody>
                <a:tcPr/>
              </a:tc>
`, runs.String())
}

func renderHeading(h markdown.Heading) string {
	fontSize := headingFontSizes[h.Level]
	if fontSize == 0 {
		fontSize = 18
	}

	var runs strings.Builder
	for _, r := range h.Runs {
		runs.WriteString(renderRun(r, fontSize, true))
	}

	return fmt.Sprintf(`          <a:p>
            <a:pPr algn="l"/>
%s          </a:p>
`, runs.String())
}

func renderParagraph(runs []markdown.Run, indent int, bullet bool) string {
	var sb strings.Builder

	pprAttrs := ""
	bulletXML := ""
	if indent > 0 {
		pprAttrs = fmt.Sprintf(` lvl="%d"`, indent-1)
	}
	if bullet {
		bulletXML = `<a:buChar char="&#x2022;"/>`
	}

	sb.WriteString(fmt.Sprintf(`          <a:p>
            <a:pPr%s>%s</a:pPr>
`, pprAttrs, bulletXML))

	for _, r := range runs {
		sb.WriteString(renderRun(r, 18, false))
	}
	sb.WriteString(`          </a:p>
`)
	return sb.String()
}

func renderRun(r markdown.Run, fontSize int, bold bool) string {
	attrs := fmt.Sprintf(` sz="%d" dirty="0"`, halfPt(fontSize))
	if r.Bold || bold {
		attrs += ` b="1"`
	}
	if r.Italic {
		attrs += ` i="1"`
	}

	fontXML := ""
	if r.Code {
		fontXML = `<a:latin typeface="Courier New"/><a:cs typeface="Courier New"/>`
	}

	linkStart := ""
	linkEnd := ""
	if r.Link != "" {
		// Hyperlinks in PPTX require relationship IDs; for now, mark as underlined
		attrs += ` u="sng"`
	}

	text := escapeXML(r.Text)

	return fmt.Sprintf(`            %s<a:r><a:rPr lang="en-US"%s>%s</a:rPr><a:t>%s</a:t></a:r>%s
`, linkStart, attrs, fontXML, text, linkEnd)
}

func renderList(l markdown.List) string {
	var sb strings.Builder
	for i, item := range l.Items {
		if l.Ordered {
			// For ordered lists, prefix with number
			prefixedRuns := make([]markdown.Run, 0, len(item.Runs)+1)
			prefixedRuns = append(prefixedRuns, markdown.Run{
				Text: fmt.Sprintf("%d. ", i+1),
			})
			prefixedRuns = append(prefixedRuns, item.Runs...)
			sb.WriteString(renderParagraph(prefixedRuns, 1, false))
		} else {
			sb.WriteString(renderParagraph(item.Runs, 1, true))
		}
	}
	return sb.String()
}

func renderCodeBlock(cb markdown.CodeBlock) string {
	lines := strings.Split(cb.Code, "\n")
	var sb strings.Builder
	for _, line := range lines {
		run := markdown.Run{Text: line, Code: true}
		sb.WriteString(renderParagraph([]markdown.Run{run}, 0, false))
	}
	return sb.String()
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
