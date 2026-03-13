package pptx

import (
	"fmt"
	"strings"

	"github.com/pspoerri/marp2pptx/internal/mermaid"
)

func renderClassDiagramShapes(layout mermaid.Layout, startID, offX, offY int) (string, int) {
	cl := layout.Class
	var sb strings.Builder
	id := startID

	classIDMap := make(map[string]int)
	for _, cn := range cl.Classes {
		classIDMap[cn.Name] = id
		sb.WriteString(renderClassNode(cn, id, offX, offY))
		id++
	}

	for _, rel := range cl.Relations {
		fromShapeID := classIDMap[rel.FromNode.Name]
		toShapeID := classIDMap[rel.ToNode.Name]
		sb.WriteString(renderClassRelation(rel, id, offX, offY, fromShapeID, toShapeID))
		id++
		if rel.Label != "" {
			sb.WriteString(renderClassRelLabel(rel, id, offX, offY))
			id++
		}
	}

	return sb.String(), id
}

func renderClassNode(cn mermaid.ClassLayoutNode, id, offX, offY int) string {
	x := offX + cn.X
	y := offY + cn.Y

	var sb strings.Builder

	// Class box (outer rectangle with header fill)
	sb.WriteString(fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Class %s"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
          <a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill>
          <a:ln w="12700"><a:solidFill><a:srgbClr val="2F5496"/></a:solidFill></a:ln>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0" anchor="t" lIns="45720" rIns="45720" tIns="0" bIns="0"/>
          <a:lstStyle/>
`, id, escapeXML(cn.Name), x, y, cn.W, cn.H))

	// Class name (bold, centered, with background)
	fontSize := 11
	if len(cn.Name) > 20 {
		fontSize = 9
	}
	sb.WriteString(fmt.Sprintf(`          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="en-US" sz="%d" b="1" dirty="0"><a:solidFill><a:srgbClr val="2F5496"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
`, halfPt(fontSize), escapeXML(cn.Name)))

	// Separator line (using a thin paragraph)
	sb.WriteString(`          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="en-US" sz="200" dirty="0"/><a:t>───────────────</a:t></a:r>
          </a:p>
`)

	// Members
	memFontSize := 9
	for _, m := range cn.Members {
		text := formatClassMember(m)
		sb.WriteString(fmt.Sprintf(`          <a:p>
            <a:pPr algn="l"/>
            <a:r><a:rPr lang="en-US" sz="%d" dirty="0"><a:latin typeface="Courier New"/><a:cs typeface="Courier New"/></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
`, halfPt(memFontSize), escapeXML(text)))
	}

	if len(cn.Members) == 0 {
		sb.WriteString(`          <a:p><a:endParaRPr lang="en-US"/></a:p>
`)
	}

	sb.WriteString(`        </p:txBody>
      </p:sp>
`)

	return sb.String()
}

func formatClassMember(m mermaid.ClassMember) string {
	s := m.Visibility
	if m.IsMethod {
		s += m.Name + "()"
		if m.Type != "" {
			s += " : " + m.Type
		}
	} else {
		s += m.Name
		if m.Type != "" {
			s += " : " + m.Type
		}
	}
	return s
}

func renderClassRelation(rel mermaid.ClassLayoutRelation, id, offX, offY, fromShapeID, toShapeID int) string {
	from := rel.FromNode
	to := rel.ToNode

	// Compute connection points
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
	lineW := 12700
	dashXML := ""
	if rel.Dashed {
		dashXML = `<a:prstDash val="dash"/>`
	}

	// Markers
	headEnd := markerToXML(rel.FromMarker, "headEnd")
	tailEnd := markerToXML(rel.ToMarker, "tailEnd")

	minX := x1
	if x2 < minX {
		minX = x2
	}
	minY := y1
	if y2 < minY {
		minY = y2
	}
	cx := abs(x2 - x1)
	cy := abs(y2 - y1)
	if cx == 0 {
		cx = 1
	}
	if cy == 0 {
		cy = 1
	}

	flipH := ""
	flipV := ""
	if x2 < x1 {
		flipH = ` flipH="1"`
	}
	if y2 < y1 {
		flipV = ` flipV="1"`
	}

	geom := connectorGeom(cx, cy)

	return fmt.Sprintf(`      <p:cxnSp>
        <p:nvCxnSpPr>
          <p:cNvPr id="%d" name="Relation %d"/>
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
            %s%s%s
          </a:ln>
        </p:spPr>
      </p:cxnSp>
`, id, id, fromShapeID, fromIdx, toShapeID, toIdx,
		flipH, flipV, minX, minY, cx, cy, geom, lineW, dashXML, headEnd, tailEnd)
}

func markerToXML(marker mermaid.RelMarker, tag string) string {
	switch marker {
	case mermaid.MarkerArrow:
		return fmt.Sprintf(`<%s type="arrow" w="med" len="med"/>`, tag)
	case mermaid.MarkerTriangle:
		return fmt.Sprintf(`<%s type="triangle" w="med" len="med"/>`, tag)
	case mermaid.MarkerDiamond:
		return fmt.Sprintf(`<%s type="diamond" w="med" len="med"/>`, tag)
	case mermaid.MarkerCircle:
		return fmt.Sprintf(`<%s type="oval" w="med" len="med"/>`, tag)
	default:
		return ""
	}
}

func renderClassRelLabel(rel mermaid.ClassLayoutRelation, id, offX, offY int) string {
	from := rel.FromNode
	to := rel.ToNode

	cx1, cy1, cx2, cy2 := computeConnectionPoints(
		from.X, from.Y, from.W, from.H,
		to.X, to.Y, to.W, to.H,
	)
	midX := offX + (cx1+cx2)/2
	midY := offY + (cy1+cy2)/2

	labelW := len(rel.Label)*emuPerPoint*8 + emuPerInch/4
	lH := emuPerInch / 4

	edgeDX := cx2 - cx1
	edgeDY := cy2 - cy1
	dx, dy := labelOffset(edgeDX, edgeDY, labelW, lH)

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
`, id, midX-labelW/2+dx, midY-lH/2+dy, labelW, lH, halfPt(9), escapeXML(rel.Label))
}

// connectionSideIdx returns the OOXML connection-site indices (top=0, right=1,
// bottom=2, left=3) for the attachment edges of two boxes.  The indices
// correspond to the edges chosen by computeConnectionPoints.
func connectionSideIdx(x1, y1, w1, h1, x2, y2, w2, h2 int) (fromIdx, toIdx int) {
	c1x := x1 + w1/2
	c1y := y1 + h1/2
	c2x := x2 + w2/2
	c2y := y2 + h2/2

	dx := c2x - c1x
	dy := c2y - c1y

	if abs(dx) > abs(dy) {
		if dx > 0 {
			return 1, 3 // source right → target left
		}
		return 3, 1 // source left → target right
	}
	if dy > 0 {
		return 2, 0 // source bottom → target top
	}
	return 0, 2 // source top → target bottom
}

// computeConnectionPoints calculates the best edge attachment points between two boxes.
func computeConnectionPoints(x1, y1, w1, h1, x2, y2, w2, h2 int) (cx1, cy1, cx2, cy2 int) {
	c1x := x1 + w1/2
	c1y := y1 + h1/2
	c2x := x2 + w2/2
	c2y := y2 + h2/2

	dx := c2x - c1x
	dy := c2y - c1y

	if abs(dx) > abs(dy) {
		if dx > 0 {
			cx1 = x1 + w1
			cy1 = c1y
			cx2 = x2
			cy2 = c2y
		} else {
			cx1 = x1
			cy1 = c1y
			cx2 = x2 + w2
			cy2 = c2y
		}
	} else {
		if dy > 0 {
			cx1 = c1x
			cy1 = y1 + h1
			cx2 = c2x
			cy2 = y2
		} else {
			cx1 = c1x
			cy1 = y1
			cx2 = c2x
			cy2 = y2 + h2
		}
	}
	return
}
