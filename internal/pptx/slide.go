package pptx

import (
	"fmt"
	"strings"

	"github.com/pspoerri/marp2pptx/internal/markdown"
	"github.com/pspoerri/marp2pptx/internal/mermaid"
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

// ImageRef holds information about an embedded image for a slide.
type ImageRef struct {
	RelID     string
	MediaPath string
	Image     markdown.Image
	WidthPx   int
	HeightPx  int
}

// segment is a group of content blocks that share a shape.
type segment struct {
	isTable   bool
	isImage   bool
	isDiagram bool
	blocks    []markdown.ContentBlock
	table     markdown.Table
	image     markdown.Image
	diagram   markdown.Diagram
}

// splitSegments groups blocks into text segments, table segments, and image segments.
func splitSegments(blocks []markdown.ContentBlock) []segment {
	var segments []segment
	var textBlocks []markdown.ContentBlock

	for _, block := range blocks {
		switch b := block.(type) {
		case markdown.Table:
			if len(textBlocks) > 0 {
				segments = append(segments, segment{blocks: textBlocks})
				textBlocks = nil
			}
			segments = append(segments, segment{isTable: true, table: b})
		case markdown.Image:
			if !b.Background {
				if len(textBlocks) > 0 {
					segments = append(segments, segment{blocks: textBlocks})
					textBlocks = nil
				}
				segments = append(segments, segment{isImage: true, image: b})
			}
		case markdown.Diagram:
			if len(textBlocks) > 0 {
				segments = append(segments, segment{blocks: textBlocks})
				textBlocks = nil
			}
			segments = append(segments, segment{isDiagram: true, diagram: b})
		default:
			textBlocks = append(textBlocks, block)
		}
	}
	if len(textBlocks) > 0 {
		segments = append(segments, segment{blocks: textBlocks})
	}
	return segments
}

// generateSlideXML produces the XML for a single slide.
func generateSlideXML(blocks []markdown.ContentBlock, bgColor string, bgImageRef *ImageRef, fgImageRefs []ImageRef) string {
	segments := splitSegments(blocks)
	if len(segments) == 0 {
		segments = []segment{{blocks: nil}}
	}

	segHeight := contentHeight / len(segments)

	// Build a map from image URL to foreground image ref for lookup
	fgRefMap := make(map[string]ImageRef)
	for _, ref := range fgImageRefs {
		fgRefMap[ref.Image.URL] = ref
	}

	var shapes strings.Builder
	shapeID := 2
	for i, seg := range segments {
		y := marginTop + i*segHeight
		if seg.isTable {
			shapes.WriteString(renderTableFrame(seg.table, shapeID, marginLeft, y, contentWidth, segHeight))
			shapeID++
		} else if seg.isImage {
			if ref, ok := fgRefMap[seg.image.URL]; ok {
				shapes.WriteString(renderImageShape(ref, shapeID, marginLeft, y, contentWidth, segHeight))
			}
			shapeID++
		} else if seg.isDiagram {
			xml, nextID := renderDiagramShapes(seg.diagram, shapeID, marginLeft, y, contentWidth, segHeight)
			shapes.WriteString(xml)
			shapeID = nextID
		} else {
			shapes.WriteString(renderTextShape(seg.blocks, shapeID, marginLeft, y, contentWidth, segHeight))
			shapeID++
		}
	}

	bgXML := ""
	if bgImageRef != nil {
		bgXML = fmt.Sprintf(`<p:bg><p:bgPr><a:blipFill><a:blip r:embed="%s"/><a:stretch><a:fillRect/></a:stretch></a:blipFill><a:effectLst/></p:bgPr></p:bg>`, bgImageRef.RelID)
	} else if bgColor != "" {
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

// renderImageShape renders an image as a picture shape.
func renderImageShape(ref ImageRef, id, x, y, cx, cy int) string {
	// Fit image within available space maintaining aspect ratio
	imgCX, imgCY := fitImage(ref.WidthPx, ref.HeightPx, cx, cy)

	// Center within available space
	offX := x + (cx-imgCX)/2
	offY := y + (cy-imgCY)/2

	return fmt.Sprintf(`      <p:pic>
        <p:nvPicPr>
          <p:cNvPr id="%d" name="Picture %d"/>
          <p:cNvPicPr><a:picLocks noChangeAspect="1"/></p:cNvPicPr>
          <p:nvPr/>
        </p:nvPicPr>
        <p:blipFill>
          <a:blip r:embed="%s"/>
          <a:stretch><a:fillRect/></a:stretch>
        </p:blipFill>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
        </p:spPr>
      </p:pic>
`, id, id, ref.RelID, offX, offY, imgCX, imgCY)
}

// fitImage scales pixel dimensions to fit within maxCX/maxCY EMU, maintaining aspect ratio.
func fitImage(imgW, imgH, maxCX, maxCY int) (cx, cy int) {
	if imgW == 0 || imgH == 0 {
		return maxCX, maxCY
	}

	// Convert pixel dimensions to EMU (assume 96 DPI)
	emuW := imgW * emuPerInch / 96
	emuH := imgH * emuPerInch / 96

	scaleX := float64(maxCX) / float64(emuW)
	scaleY := float64(maxCY) / float64(emuH)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}
	if scale > 1 {
		scale = 1 // Don't upscale
	}

	return int(float64(emuW) * scale), int(float64(emuH) * scale)
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
		case markdown.DefinitionList:
			body.WriteString(renderDefinitionList(b))
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
		if r.Strikethrough {
			attrs += ` strike="sngStrike"`
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
	if r.Strikethrough {
		attrs += ` strike="sngStrike"`
	}
	if r.Superscript {
		attrs += ` baseline="30000"`
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
		// Prepend checkbox for task list items
		itemRuns := item.Runs
		if item.Checked != nil {
			checkbox := "\u2610 " // ☐ unchecked
			if *item.Checked {
				checkbox = "\u2611 " // ☑ checked
			}
			itemRuns = append([]markdown.Run{{Text: checkbox}}, itemRuns...)
		}

		if l.Ordered {
			prefixedRuns := make([]markdown.Run, 0, len(itemRuns)+1)
			prefixedRuns = append(prefixedRuns, markdown.Run{
				Text: fmt.Sprintf("%d. ", i+1),
			})
			prefixedRuns = append(prefixedRuns, itemRuns...)
			sb.WriteString(renderParagraph(prefixedRuns, 1, false))
		} else {
			sb.WriteString(renderParagraph(itemRuns, 1, true))
		}
	}
	return sb.String()
}

func renderDefinitionList(dl markdown.DefinitionList) string {
	var sb strings.Builder
	for _, item := range dl.Items {
		// Render term as bold paragraph
		boldRuns := make([]markdown.Run, len(item.Term))
		for i, r := range item.Term {
			boldRuns[i] = r
			boldRuns[i].Bold = true
		}
		sb.WriteString(renderParagraph(boldRuns, 0, false))

		// Render each description as indented paragraph
		for _, desc := range item.Descriptions {
			sb.WriteString(renderParagraph(desc, 1, false))
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

// renderDiagramShapes renders a mermaid diagram as native PPTX shapes.
// Returns the XML and the next available shape ID.
func renderDiagramShapes(d markdown.Diagram, startID, x, y, cx, cy int) (string, int) {
	layout := mermaid.ComputeLayout(d.Graph, cx, cy)
	var sb strings.Builder
	id := startID

	// Render nodes as shapes
	nodeIDMap := make(map[string]int) // node ID -> shape ID for connector refs
	for _, ln := range layout.Nodes {
		nodeIDMap[ln.ID] = id
		sb.WriteString(renderDiagramNode(ln, id, x, y))
		id++
	}

	// Render edges as connector shapes
	for _, le := range layout.Edges {
		fromShapeID := nodeIDMap[le.From]
		toShapeID := nodeIDMap[le.To]
		sb.WriteString(renderDiagramEdge(le, id, x, y, fromShapeID, toShapeID))
		id++
		// If edge has a label, render it as a text box
		if le.Label != "" {
			sb.WriteString(renderEdgeLabel(le, id, x, y))
			id++
		}
	}

	return sb.String(), id
}

// prstGeomForShape maps mermaid node shapes to OOXML preset geometry names.
func prstGeomForShape(s mermaid.NodeShape) string {
	switch s {
	case mermaid.ShapeRound:
		return "roundRect"
	case mermaid.ShapeDiamond:
		return "diamond"
	case mermaid.ShapeCircle:
		return "ellipse"
	case mermaid.ShapeStadium:
		return "roundRect"
	case mermaid.ShapeHexagon:
		return "hexagon"
	case mermaid.ShapeParallel:
		return "parallelogram"
	case mermaid.ShapeTrapezoid:
		return "trapezoid"
	default:
		return "rect"
	}
}

func renderDiagramNode(ln mermaid.LayoutNode, id, offX, offY int) string {
	prst := prstGeomForShape(ln.Shape)
	fontSize := 12
	if len(ln.Label) > 20 {
		fontSize = 10
	}

	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Node %s"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="%s"><a:avLst/></a:prstGeom>
          <a:solidFill><a:srgbClr val="4472C4"/></a:solidFill>
          <a:ln w="12700"><a:solidFill><a:srgbClr val="2F5496"/></a:solidFill></a:ln>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0" anchor="ctr" anchorCtr="1"/>
          <a:lstStyle/>
          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="en-US" sz="%d" dirty="0"><a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
`, id, escapeXML(ln.ID), offX+ln.X, offY+ln.Y, ln.W, ln.H, prst, halfPt(fontSize), escapeXML(ln.Label))
}

func renderDiagramEdge(le mermaid.LayoutEdge, id, offX, offY, fromShapeID, toShapeID int) string {
	// Compute connection points (center of each node's edge)
	from := le.FromNode
	to := le.ToNode

	// Determine which sides to connect based on relative position
	var x1, y1, x2, y2 int
	dx := (to.X + to.W/2) - (from.X + from.W/2)
	dy := (to.Y + to.H/2) - (from.Y + from.H/2)

	if abs(dx) > abs(dy) {
		// Horizontal connection
		if dx > 0 {
			x1 = from.X + from.W
			y1 = from.Y + from.H/2
			x2 = to.X
			y2 = to.Y + to.H/2
		} else {
			x1 = from.X
			y1 = from.Y + from.H/2
			x2 = to.X + to.W
			y2 = to.Y + to.H/2
		}
	} else {
		// Vertical connection
		if dy > 0 {
			x1 = from.X + from.W/2
			y1 = from.Y + from.H
			x2 = to.X + to.W/2
			y2 = to.Y
		} else {
			x1 = from.X + from.W/2
			y1 = from.Y
			x2 = to.X + to.W/2
			y2 = to.Y + to.H
		}
	}

	// Apply segment offset
	x1 += offX
	y1 += offY
	x2 += offX
	y2 += offY

	// Line style
	lineW := 12700 // 1pt
	dashXML := ""
	switch le.Style {
	case mermaid.EdgeDotted:
		dashXML = `<a:prstDash val="dash"/>`
	case mermaid.EdgeThick:
		lineW = 25400 // 2pt
	}

	// Arrow head
	tailEnd := ""
	if le.Arrow {
		tailEnd = `<a:tailEnd type="triangle" w="med" len="med"/>`
	}

	// Use a freeform connector (two-point line)
	// Compute bounding box
	minX := min(x1, x2)
	minY := min(y1, y2)
	cxLine := abs(x2 - x1)
	cyLine := abs(y2 - y1)
	if cxLine == 0 {
		cxLine = 1
	}
	if cyLine == 0 {
		cyLine = 1
	}

	// Flip flags for connector
	flipH := ""
	flipV := ""
	if x2 < x1 {
		flipH = ` flipH="1"`
	}
	if y2 < y1 {
		flipV = ` flipV="1"`
	}

	return fmt.Sprintf(`      <p:cxnSp>
        <p:nvCxnSpPr>
          <p:cNvPr id="%d" name="Connector %d"/>
          <p:cNvCxnSpPr>
            <a:stCxn id="%d" idx="0"/>
            <a:endCxn id="%d" idx="0"/>
          </p:cNvCxnSpPr>
          <p:nvPr/>
        </p:nvCxnSpPr>
        <p:spPr>
          <a:xfrm%s%s>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="straightConnector1"><a:avLst/></a:prstGeom>
          <a:ln w="%d">
            <a:solidFill><a:srgbClr val="2F5496"/></a:solidFill>
            %s%s
          </a:ln>
        </p:spPr>
      </p:cxnSp>
`, id, id, fromShapeID, toShapeID,
		flipH, flipV, minX, minY, cxLine, cyLine,
		lineW, dashXML, tailEnd)
}

func renderEdgeLabel(le mermaid.LayoutEdge, id, offX, offY int) string {
	// Place label at midpoint of the edge
	from := le.FromNode
	to := le.ToNode
	midX := offX + (from.X+from.W/2+to.X+to.W/2)/2
	midY := offY + (from.Y+from.H/2+to.Y+to.H/2)/2

	labelW := len(le.Label)*emuPerPoint*8 + emuPerInch/4
	labelH := emuPerInch / 4

	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Label"/>
          <p:cNvSpPr txBox="1"/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
          <a:noFill/>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0" anchor="ctr" anchorCtr="1"/>
          <a:lstStyle/>
          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="en-US" sz="%d" dirty="0"/><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
`, id, midX-labelW/2, midY-labelH/2, labelW, labelH, halfPt(10), escapeXML(le.Label))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
