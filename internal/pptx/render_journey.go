package pptx

import (
	"fmt"
	"strings"

	"github.com/pspoerri/marp2pptx/internal/mermaid"
)

// Score-to-color mapping (1=red, 3=yellow, 5=green)
var journeyScoreColors = []string{
	"E74C3C", // 1 - red
	"E67E22", // 2 - orange
	"F1C40F", // 3 - yellow
	"2ECC71", // 4 - light green
	"27AE60", // 5 - green
}

func renderJourneyDiagramShapes(layout mermaid.Layout, startID, offX, offY int) (string, int) {
	jl := layout.Journey
	var sb strings.Builder
	id := startID

	// Title
	if jl.Title != "" {
		sb.WriteString(renderJourneyTitle(jl, id, offX, offY))
		id++
	}

	// Sections and tasks
	for _, sec := range jl.Sections {
		if sec.Name != "" {
			sb.WriteString(renderJourneySection(sec, id, offX, offY))
			id++
		}
		for _, task := range sec.Tasks {
			sb.WriteString(renderJourneyTaskBar(task, id, offX, offY))
			id++
			sb.WriteString(renderJourneyTaskLabel(task, id, offX, offY))
			id++
		}
	}

	return sb.String(), id
}

func renderJourneyTitle(jl *mermaid.JourneyLayout, id, offX, offY int) string {
	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Journey Title"/>
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
            <a:pPr algn="l"/>
            <a:r><a:rPr lang="en-US" sz="%d" b="1" dirty="0"><a:solidFill><a:srgbClr val="2F5496"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
`, id, offX+jl.TitleX, offY+jl.TitleY, jl.TitleW, jl.TitleH, halfPt(14), escapeXML(jl.Title))
}

func renderJourneySection(sec mermaid.JourneySectionLayout, id, offX, offY int) string {
	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="Section %s"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="rect"><a:avLst/></a:prstGeom>
          <a:solidFill><a:srgbClr val="D6DCE4"/></a:solidFill>
          <a:ln w="0"><a:noFill/></a:ln>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0" anchor="ctr" lIns="91440"/>
          <a:lstStyle/>
          <a:p>
            <a:pPr algn="l"/>
            <a:r><a:rPr lang="en-US" sz="%d" b="1" dirty="0"><a:solidFill><a:srgbClr val="44546A"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
`, id, escapeXML(sec.Name), offX+sec.X, offY+sec.Y, sec.W, sec.H, halfPt(10), escapeXML(sec.Name))
}

func renderJourneyTaskBar(task mermaid.JourneyTaskLayout, id, offX, offY int) string {
	colorIdx := task.Score - 1
	if colorIdx < 0 {
		colorIdx = 0
	}
	if colorIdx >= len(journeyScoreColors) {
		colorIdx = len(journeyScoreColors) - 1
	}
	color := journeyScoreColors[colorIdx]

	// Background bar (gray)
	bgColor := "E8E8E8"

	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="TaskBg"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="roundRect"><a:avLst><a:gd name="adj" fmla="val 50000"/></a:avLst></a:prstGeom>
          <a:solidFill><a:srgbClr val="%s"/></a:solidFill>
          <a:ln w="0"><a:noFill/></a:ln>
        </p:spPr>
      </p:sp>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="TaskBar"/>
          <p:cNvSpPr/>
          <p:nvPr/>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
          <a:prstGeom prst="roundRect"><a:avLst><a:gd name="adj" fmla="val 50000"/></a:avLst></a:prstGeom>
          <a:solidFill><a:srgbClr val="%s"/></a:solidFill>
          <a:ln w="0"><a:noFill/></a:ln>
        </p:spPr>
      </p:sp>
`, id, offX+task.X, offY+task.Y, task.W, task.H, bgColor,
		id+10000, offX+task.X, offY+task.Y, task.BarW, task.H, color)
}

func renderJourneyTaskLabel(task mermaid.JourneyTaskLayout, id, offX, offY int) string {
	labelW := task.X // use the space to the left of the bar
	return fmt.Sprintf(`      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="%d" name="TaskLabel"/>
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
          <a:bodyPr wrap="square" rtlCol="0" anchor="ctr" rIns="91440"/>
          <a:lstStyle/>
          <a:p>
            <a:pPr algn="r"/>
            <a:r><a:rPr lang="en-US" sz="%d" dirty="0"><a:solidFill><a:srgbClr val="44546A"/></a:solidFill></a:rPr><a:t>%s</a:t></a:r>
          </a:p>
        </p:txBody>
      </p:sp>
`, id, offX, offY+task.Y, labelW, task.H, halfPt(9), escapeXML(task.Name))
}
