package pptx

import (
	"fmt"
	"strings"

	"github.com/pspoerri/marp2pptx/internal/mermaid"
)

func renderERDiagramShapes(layout mermaid.Layout, startID, offX, offY int) (string, int) {
	el := layout.ER
	var sb strings.Builder
	id := startID

	for _, en := range el.Entities {
		sb.WriteString(renderEREntity(en, id, offX, offY))
		id++
	}

	for _, rel := range el.Relationships {
		sb.WriteString(renderERRelation(rel, id, offX, offY))
		id++
		// Cardinality labels
		sb.WriteString(renderERCardLabel(rel, id, offX, offY, true))
		id++
		sb.WriteString(renderERCardLabel(rel, id, offX, offY, false))
		id++
		if rel.Label != "" {
			sb.WriteString(renderERRelLabel(rel, id, offX, offY))
			id++
		}
	}

	return sb.String(), id
}

func renderEREntity(en mermaid.EREntityLayout, id, offX, offY int) string {
	x := offX + en.X
	y := offY + en.Y

	var sb strings.Builder

	// Entity box
	sb.WriteString(fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Entity %s"/>
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
`, id, escapeXML(en.Name), x, y, en.W, en.H))

	// Entity name
	fontSize := 11
	if len(en.Name) > 20 {
		fontSize = 9
	}
	sb.WriteString(fmt.Sprintf(`          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="en-US" sz="%d" b="1" dirty="0"><a:solidFill><a:srgbClr val="2F5496"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
`, halfPt(fontSize), escapeXML(en.Name)))

	if len(en.Attributes) > 0 {
		sb.WriteString(`          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="en-US" sz="200" dirty="0"/><a:t>───────────────</a:t></a:r>
          </a:p>
`)
		attrFontSize := 8
		for _, attr := range en.Attributes {
			text := attr.Type + " " + attr.Name
			if len(attr.Keys) > 0 {
				text += " " + strings.Join(attr.Keys, ",")
			}
			sb.WriteString(fmt.Sprintf(`          <a:p>
            <a:pPr algn="l"/>
            <a:r><a:rPr lang="en-US" sz="%d" dirty="0"><a:latin typeface="Courier New"/><a:cs typeface="Courier New"/></a:rPr><a:t>  %s</a:t></a:r>
          </a:p>
`, halfPt(attrFontSize), escapeXML(text)))
		}
	} else {
		sb.WriteString(`          <a:p><a:endParaRPr lang="en-US"/></a:p>
`)
	}

	sb.WriteString(`        </p:txBody>
      </p:sp>
`)
	return sb.String()
}

func renderERRelation(rel mermaid.ERRelationshipLayout, id, offX, offY int) string {
	from := rel.FromEntity
	to := rel.ToEntity

	x1, y1, x2, y2 := computeConnectionPoints(
		from.X, from.Y, from.W, from.H,
		to.X, to.Y, to.W, to.H,
	)
	x1 += offX
	y1 += offY
	x2 += offX
	y2 += offY

	lineW := 12700
	dashXML := ""
	if !rel.Identifying {
		dashXML = `<a:prstDash val="dash"/>`
	}

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
          <p:cNvPr id="%d" name="ERRel %d"/>
          <p:cNvCxnSpPr/>
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
            %s
          </a:ln>
        </p:spPr>
      </p:cxnSp>
`, id, id, flipH, flipV, minX, minY, cx, cy, geom, lineW, dashXML)
}

func cardinalityText(c mermaid.ERCardinality) string {
	switch c {
	case mermaid.CardExactlyOne:
		return "1"
	case mermaid.CardZeroOrOne:
		return "0..1"
	case mermaid.CardOneOrMore:
		return "1..*"
	case mermaid.CardZeroOrMore:
		return "0..*"
	default:
		return ""
	}
}

func renderERCardLabel(rel mermaid.ERRelationshipLayout, id, offX, offY int, isFrom bool) string {
	var card mermaid.ERCardinality
	var entity mermaid.EREntityLayout
	var other mermaid.EREntityLayout
	if isFrom {
		card = rel.CardinalityA
		entity = rel.FromEntity
		other = rel.ToEntity
	} else {
		card = rel.CardinalityB
		entity = rel.ToEntity
		other = rel.FromEntity
	}

	text := cardinalityText(card)
	if text == "" {
		return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Card"/>
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

	// Position near the entity end of the line
	cx1, cy1, _, _ := computeConnectionPoints(
		entity.X, entity.Y, entity.W, entity.H,
		other.X, other.Y, other.W, other.H,
	)
	labelX := offX + cx1
	labelY := offY + cy1

	labelW := emuPerInch / 2
	labelH := emuPerInch / 4

	// Offset slightly toward the other entity
	dx := other.X + other.W/2 - entity.X - entity.W/2
	dy := other.Y + other.H/2 - entity.Y - entity.H/2
	if abs(dx) > abs(dy) {
		if dx > 0 {
			labelX += emuPerInch / 16
		} else {
			labelX -= labelW + emuPerInch/16
		}
		labelY -= labelH + emuPerInch/16
	} else {
		if dy > 0 {
			labelY += emuPerInch / 16
		} else {
			labelY -= labelH + emuPerInch/16
		}
		labelX += emuPerInch / 16
	}

	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Card"/>
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
          <a:bodyPr wrap="square" rtlCol="0" anchor="ctr"/>
          <a:lstStyle/>
          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="en-US" sz="%d" b="1" dirty="0"><a:solidFill><a:srgbClr val="2F5496"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
`, id, labelX, labelY, labelW, labelH, halfPt(8), escapeXML(text))
}

func renderERRelLabel(rel mermaid.ERRelationshipLayout, id, offX, offY int) string {
	from := rel.FromEntity
	to := rel.ToEntity

	cx1, cy1, cx2, cy2 := computeConnectionPoints(
		from.X, from.Y, from.W, from.H,
		to.X, to.Y, to.W, to.H,
	)
	midX := offX + (cx1+cx2)/2
	midY := offY + (cy1+cy2)/2

	labelW := len(rel.Label)*emuPerPoint*7 + emuPerInch/4
	lH := emuPerInch / 4

	edgeDX := cx2 - cx1
	edgeDY := cy2 - cy1
	dx, dy := labelOffset(edgeDX, edgeDY, labelW, lH)

	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="RelLabel"/>
          <p:cNvSpPr txBox="1"/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
          <a:solidFill><a:srgbClr val="FFFFFF"/></a:solidFill>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0" anchor="ctr" anchorCtr="1"/>
          <a:lstStyle/>
          <a:p>
            <a:pPr algn="ctr"/>
            <a:r><a:rPr lang="en-US" sz="%d" i="1" dirty="0"><a:solidFill><a:srgbClr val="44546A"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
`, id, midX-labelW/2+dx, midY-lH/2+dy, labelW, lH, halfPt(9), escapeXML(rel.Label))
}
