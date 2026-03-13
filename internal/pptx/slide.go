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

	switch layout.Type {
	case mermaid.DiagramSequence:
		if layout.Sequence != nil {
			return renderSequenceDiagram(layout, startID, x, y)
		}
	case mermaid.DiagramClass:
		if layout.Class != nil {
			return renderClassDiagramShapes(layout, startID, x, y)
		}
	case mermaid.DiagramState:
		if layout.State != nil {
			return renderStateDiagramShapes(layout, startID, x, y)
		}
	case mermaid.DiagramJourney:
		if layout.Journey != nil {
			return renderJourneyDiagramShapes(layout, startID, x, y)
		}
	case mermaid.DiagramER:
		if layout.ER != nil {
			return renderERDiagramShapes(layout, startID, x, y)
		}
	}
	return renderFlowchartDiagram(layout, startID, x, y)
}

func renderFlowchartDiagram(layout mermaid.Layout, startID, x, y int) (string, int) {
	var sb strings.Builder
	id := startID

	nodeIDMap := make(map[string]int)
	for _, ln := range layout.Nodes {
		nodeIDMap[ln.ID] = id
		sb.WriteString(renderDiagramNode(ln, id, x, y))
		id++
	}

	for _, le := range layout.Edges {
		fromShapeID := nodeIDMap[le.From]
		toShapeID := nodeIDMap[le.To]
		sb.WriteString(renderDiagramEdge(le, id, x, y, fromShapeID, toShapeID))
		id++
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
	case mermaid.ShapeRound, mermaid.ShapeStadium:
		return "roundRect"
	case mermaid.ShapeDiamond:
		return "diamond"
	case mermaid.ShapeCircle, mermaid.ShapeDoubleCircle:
		return "ellipse"
	case mermaid.ShapeSubroutine:
		return "flowChartPredefinedProcess"
	case mermaid.ShapeCylinder:
		return "can"
	case mermaid.ShapeAsymmetric:
		return "homePlate"
	case mermaid.ShapeHexagon:
		return "hexagon"
	case mermaid.ShapeParallel, mermaid.ShapeParallelAlt:
		return "parallelogram"
	case mermaid.ShapeTrapezoid, mermaid.ShapeTrapezoidAlt:
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

// backEdgeOffset is the horizontal routing offset for back-edges (1/3 inch).
const backEdgeOffset = emuPerInch / 3

// isBackEdge returns true if the edge goes "upward" (target above source)
// and the two nodes overlap horizontally, meaning a straight connector
// would pass through intermediate nodes.
func isBackEdge(from, to mermaid.LayoutNode) bool {
	fromCY := from.Y + from.H/2
	toCY := to.Y + to.H/2
	if toCY >= fromCY {
		return false
	}
	// Check horizontal overlap
	overlapL := from.X
	if to.X > overlapL {
		overlapL = to.X
	}
	overlapR := from.X + from.W
	if to.X+to.W < overlapR {
		overlapR = to.X + to.W
	}
	return overlapR > overlapL
}

// renderBackEdgeShape renders a back-edge that routes around the right side
// of intermediate nodes using a custom geometry path with rounded corners
// and an explicit arrowhead triangle.
func renderBackEdgeShape(le mermaid.LayoutEdge, id, offX, offY int) string {
	from := le.FromNode
	to := le.ToNode

	// Connect from right side of source to right side of target
	startX := from.X + from.W
	startY := from.Y + from.H/2
	endX := to.X + to.W
	endY := to.Y + to.H/2

	rightMax := startX
	if endX > rightMax {
		rightMax = endX
	}
	routeX := rightMax + backEdgeOffset

	// Arrowhead dimensions
	arrowLen := 0
	arrowHW := 0 // half-width
	if le.Arrow {
		arrowLen = emuPerInch / 12
		arrowHW = emuPerInch / 24
	}

	// Bounding box (extended for arrowhead wings)
	bbX := startX
	if endX < bbX {
		bbX = endX
	}
	bbY := endY - arrowHW
	bbCX := routeX - bbX
	bbCY := startY - bbY

	// Path coordinates relative to bounding box
	// Y coords shifted by arrowHW to accommodate arrowhead wings
	p1x := startX - bbX
	p1y := bbCY // bottom (source level)
	p2x := bbCX // right edge (route point)
	p2y := bbCY
	p3x := bbCX
	p3y := arrowHW // top (target level, shifted by arrowHW)

	// Line endpoint: stop at arrowhead base (arrowLen right of target edge)
	lineEndX := endX - bbX + arrowLen
	lineEndY := arrowHW

	// Corner radius
	r := emuPerInch / 8
	if r > bbCX/3 {
		r = bbCX / 3
	}
	if vertSpan := bbCY - 2*arrowHW; r > vertSpan/6 {
		r = vertSpan / 6
	}

	// Line style
	lineW := 12700 // 1pt
	dashXML := ""
	switch le.Style {
	case mermaid.EdgeDotted:
		dashXML = `<a:prstDash val="dash"/>`
	case mermaid.EdgeThick:
		lineW = 25400 // 2pt
	}

	// Build the routing line path (unfilled, stroked only)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="BackEdge %d"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:custGeom>
            <a:avLst/>
            <a:gdLst/>
            <a:ahLst/>
            <a:cxnLst/>
            <a:rect l="0" t="0" r="0" b="0"/>
            <a:pathLst>
              <a:path w="%d" h="%d" fill="none">
                <a:moveTo><a:pt x="%d" y="%d"/></a:moveTo>
                <a:lnTo><a:pt x="%d" y="%d"/></a:lnTo>
                <a:quadBezTo><a:pt x="%d" y="%d"/><a:pt x="%d" y="%d"/></a:quadBezTo>
                <a:lnTo><a:pt x="%d" y="%d"/></a:lnTo>
                <a:quadBezTo><a:pt x="%d" y="%d"/><a:pt x="%d" y="%d"/></a:quadBezTo>
                <a:lnTo><a:pt x="%d" y="%d"/></a:lnTo>
              </a:path>
`, id, id,
		offX+bbX, offY+bbY, bbCX, bbCY,
		bbCX, bbCY,
		p1x, p1y, // moveTo: start at source right edge
		p2x-r, p2y, // lineTo: approach bottom-right corner
		p2x, p2y, p2x, p2y-r, // quadBezTo: round bottom-right corner
		p3x, p3y+r, // lineTo: go up, approach top-right corner
		p3x, p3y, p3x-r, p3y, // quadBezTo: round top-right corner
		lineEndX, lineEndY)) // lineTo: go left to arrowhead base

	// Add filled arrowhead triangle pointing left at target's right edge
	if le.Arrow {
		tipX := endX - bbX
		tipY := arrowHW
		sb.WriteString(fmt.Sprintf(`              <a:path w="%d" h="%d">
                <a:moveTo><a:pt x="%d" y="%d"/></a:moveTo>
                <a:lnTo><a:pt x="%d" y="%d"/></a:lnTo>
                <a:lnTo><a:pt x="%d" y="%d"/></a:lnTo>
                <a:close/>
              </a:path>
`, bbCX, bbCY,
			tipX, tipY, // arrow tip at target right edge
			tipX+arrowLen, tipY-arrowHW, // upper wing
			tipX+arrowLen, tipY+arrowHW)) // lower wing
	}

	// Shape fill: needed for the arrowhead triangle; line path uses fill="none"
	fillXML := `<a:noFill/>`
	if le.Arrow {
		fillXML = `<a:solidFill><a:srgbClr val="2F5496"/></a:solidFill>`
	}

	sb.WriteString(fmt.Sprintf(`            </a:pathLst>
          </a:custGeom>
          %s
          <a:ln w="%d">
            <a:solidFill><a:srgbClr val="2F5496"/></a:solidFill>
            %s
          </a:ln>
        </p:spPr>
      </p:sp>
`, fillXML, lineW, dashXML))

	return sb.String()
}

func renderDiagramEdge(le mermaid.LayoutEdge, id, offX, offY, fromShapeID, toShapeID int) string {
	from := le.FromNode
	to := le.ToNode

	// Route back-edges around the side to avoid passing through intermediate nodes
	if isBackEdge(from, to) {
		return renderBackEdgeShape(le, id, offX, offY)
	}

	x1, y1, x2, y2 := computeConnectionPoints(
		from.X, from.Y, from.W, from.H,
		to.X, to.Y, to.W, to.H,
	)
	fromIdx, toIdx := connectionSideIdx(
		from.X, from.Y, from.W, from.H,
		to.X, to.Y, to.W, to.H,
	)
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

	geom := connectorGeom(cxLine, cyLine)

	return fmt.Sprintf(`      <p:cxnSp>
        <p:nvCxnSpPr>
          <p:cNvPr id="%d" name="Connector %d"/>
          <p:cNvCxnSpPr>
            <a:stCxn id="%d" idx="%d"/>
            <a:endCxn id="%d" idx="%d"/>
          </p:cNvCxnSpPr>
          <p:nvPr/>
        </p:nvCxnSpPr>
        <p:spPr>
          <a:xfrm%s%s>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="%s"><a:avLst/></a:prstGeom>
          <a:ln w="%d">
            <a:solidFill><a:srgbClr val="2F5496"/></a:solidFill>
            %s%s
          </a:ln>
        </p:spPr>
      </p:cxnSp>
`, id, id, fromShapeID, fromIdx, toShapeID, toIdx,
		flipH, flipV, minX, minY, cxLine, cyLine,
		geom, lineW, dashXML, tailEnd)
}

func renderEdgeLabel(le mermaid.LayoutEdge, id, offX, offY int) string {
	from := le.FromNode
	to := le.ToNode

	labelW := len(le.Label)*emuPerPoint*8 + emuPerInch/4
	lH := emuPerInch / 4

	var labelX, labelY int

	if isBackEdge(from, to) {
		// Center label on the vertical segment of the back-edge route
		rightX := from.X + from.W
		if to.X+to.W > rightX {
			rightX = to.X + to.W
		}
		rightX += backEdgeOffset
		midY := (from.Y + from.H/2 + to.Y + to.H/2) / 2
		labelX = offX + rightX - labelW/2
		labelY = offY + midY - lH/2
	} else {
		// Compute actual connection points for accurate midpoint
		cx1, cy1, cx2, cy2 := computeConnectionPoints(
			from.X, from.Y, from.W, from.H,
			to.X, to.Y, to.W, to.H,
		)
		midX := offX + (cx1+cx2)/2
		midY := offY + (cy1+cy2)/2

		// Offset perpendicular to edge so label doesn't overlap the connector
		edgeDX := cx2 - cx1
		edgeDY := cy2 - cy1
		dx, dy := labelOffset(edgeDX, edgeDY, labelW, lH)
		labelX = midX - labelW/2 + dx
		labelY = midY - lH/2 + dy
	}

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
`, id, labelX, labelY, labelW, lH, halfPt(10), escapeXML(le.Label))
}

// ---------------------------------------------------------------------------
// Sequence diagram rendering
// ---------------------------------------------------------------------------

func renderSequenceDiagram(layout mermaid.Layout, startID, offX, offY int) (string, int) {
	seq := layout.Sequence
	var sb strings.Builder
	id := startID

	// Render participant boxes
	for _, p := range seq.Participants {
		sb.WriteString(renderSeqParticipant(p, id, offX, offY))
		id++
	}

	// Render lifelines (dashed vertical lines)
	for _, p := range seq.Participants {
		sb.WriteString(renderSeqLifeline(p, id, offX, offY))
		id++
	}

	// Render messages (horizontal arrows with labels)
	for _, m := range seq.Messages {
		sb.WriteString(renderSeqMessage(m, id, offX, offY))
		id++
		// Message label
		sb.WriteString(renderSeqMessageLabel(m, id, offX, offY))
		id++
	}

	return sb.String(), id
}

func renderSeqParticipant(p mermaid.SeqParticipantLayout, id, offX, offY int) string {
	fontSize := 12
	if len(p.Label) > 15 {
		fontSize = 10
	}

	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Participant %s"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
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
`, id, escapeXML(p.ID), offX+p.X, offY+p.Y, p.W, p.H, halfPt(fontSize), escapeXML(p.Label))
}

func renderSeqLifeline(p mermaid.SeqParticipantLayout, id, offX, offY int) string {
	x := offX + p.LifelineX
	y1 := offY + p.LifelineTopY
	y2 := offY + p.LifelineBotY
	cy := y2 - y1
	if cy <= 0 {
		cy = 1
	}

	return fmt.Sprintf(`      <p:cxnSp>
        <p:nvCxnSpPr>
          <p:cNvPr id="%d" name="Lifeline %s"/>
          <p:cNvCxnSpPr/>
          <p:nvPr/>
        </p:nvCxnSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="0" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="straightConnector1"><a:avLst/></a:prstGeom>
          <a:ln w="9525">
            <a:solidFill><a:srgbClr val="808080"/></a:solidFill>
            <a:prstDash val="dash"/>
          </a:ln>
        </p:spPr>
      </p:cxnSp>
`, id, escapeXML(p.ID), x, y1, cy)
}

func renderSeqMessage(m mermaid.SeqMessageLayout, id, offX, offY int) string {
	x1 := offX + m.FromX
	x2 := offX + m.ToX
	y := offY + m.Y

	minX := x1
	cx := x2 - x1
	flipH := ""
	if cx < 0 {
		minX = x2
		cx = -cx
		flipH = ` flipH="1"`
	}
	if cx == 0 {
		cx = 1
	}

	// Line style based on message type
	lineW := 12700
	dashXML := ""
	tailEnd := ""

	switch m.Style {
	case mermaid.MsgSolid:
		// solid, no arrow
	case mermaid.MsgDotted:
		dashXML = `<a:prstDash val="dash"/>`
	case mermaid.MsgSolidArrow:
		tailEnd = `<a:tailEnd type="triangle" w="med" len="med"/>`
	case mermaid.MsgDottedArrow:
		dashXML = `<a:prstDash val="dash"/>`
		tailEnd = `<a:tailEnd type="triangle" w="med" len="med"/>`
	case mermaid.MsgSolidCross:
		tailEnd = `<a:tailEnd type="diamond" w="med" len="med"/>`
	case mermaid.MsgDottedCross:
		dashXML = `<a:prstDash val="dash"/>`
		tailEnd = `<a:tailEnd type="diamond" w="med" len="med"/>`
	case mermaid.MsgSolidAsync:
		tailEnd = `<a:tailEnd type="arrow" w="med" len="med"/>`
	case mermaid.MsgDottedAsync:
		dashXML = `<a:prstDash val="dash"/>`
		tailEnd = `<a:tailEnd type="arrow" w="med" len="med"/>`
	}

	return fmt.Sprintf(`      <p:cxnSp>
        <p:nvCxnSpPr>
          <p:cNvPr id="%d" name="Message %d"/>
          <p:cNvCxnSpPr/>
          <p:nvPr/>
        </p:nvCxnSpPr>
        <p:spPr>
          <a:xfrm%s>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="0"/>
          </a:xfrm>
          <a:prstGeom prst="straightConnector1"><a:avLst/></a:prstGeom>
          <a:ln w="%d">
            <a:solidFill><a:srgbClr val="2F5496"/></a:solidFill>
            %s%s
          </a:ln>
        </p:spPr>
      </p:cxnSp>
`, id, id, flipH, minX, y, cx, lineW, dashXML, tailEnd)
}

func renderSeqMessageLabel(m mermaid.SeqMessageLayout, id, offX, offY int) string {
	if m.Label == "" {
		return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Label"/>
          <p:cNvSpPr txBox="1"/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="0" y="0"/>
            <a:ext cx="0" cy="0"/>
          </a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
          <a:noFill/>
        </p:spPr>
        <p:txBody>
          <a:bodyPr/>
          <a:lstStyle/>
          <a:p><a:endParaRPr lang="en-US"/></a:p>
        </p:txBody>
      </p:sp>
`, id)
	}

	midX := offX + (m.FromX+m.ToX)/2
	labelW := len(m.Label)*emuPerPoint*7 + emuPerInch/4
	labelH := emuPerInch / 4
	y := offY + m.Y - labelH - emuPerPoint*4

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
          <a:bodyPr wrap="square" rtlCol="0" anchor="b" anchorCtr="1"/>
          <a:lstStyle/>
          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="en-US" sz="%d" dirty="0"/><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
`, id, midX-labelW/2, y, labelW, labelH, halfPt(10), escapeXML(m.Label))
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
