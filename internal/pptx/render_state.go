package pptx

import (
	"fmt"
	"strings"

	"github.com/pspoerri/marp2pptx/internal/mermaid"
)

func renderStateDiagramShapes(layout mermaid.Layout, startID, offX, offY int) (string, int) {
	sl := layout.State
	var sb strings.Builder
	id := startID

	nodeIDMap := make(map[string]int)
	for _, ln := range sl.Nodes {
		nodeIDMap[ln.ID] = id
		if sl.Stars[ln.ID] {
			sb.WriteString(renderStarNode(ln, id, offX, offY))
		} else {
			sb.WriteString(renderStateNode(ln, id, offX, offY))
		}
		id++
	}

	for _, le := range sl.Edges {
		fromShapeID := nodeIDMap[le.From]
		toShapeID := nodeIDMap[le.To]
		sb.WriteString(renderDiagramEdge(le, id, offX, offY, fromShapeID, toShapeID))
		id++
		if le.Label != "" {
			sb.WriteString(renderEdgeLabel(le, id, offX, offY))
			id++
		}
	}

	return sb.String(), id
}

func renderStateNode(ln mermaid.LayoutNode, id, offX, offY int) string {
	fontSize := 11
	if len(ln.Label) > 20 {
		fontSize = 9
	}

	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="State %s"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="roundRect"><a:avLst/></a:prstGeom>
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
`, id, escapeXML(ln.ID), offX+ln.X, offY+ln.Y, ln.W, ln.H, halfPt(fontSize), escapeXML(ln.Label))
}

func renderStarNode(ln mermaid.LayoutNode, id, offX, offY int) string {
	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Star %s"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="ellipse"><a:avLst/></a:prstGeom>
          <a:solidFill><a:srgbClr val="2F5496"/></a:solidFill>
          <a:ln w="12700"><a:solidFill><a:srgbClr val="2F5496"/></a:solidFill></a:ln>
        </p:spPr>
      </p:sp>
`, id, escapeXML(ln.ID), offX+ln.X, offY+ln.Y, ln.W, ln.H)
}
