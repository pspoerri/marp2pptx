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

// generateSlideXML produces the XML for a single slide.
func generateSlideXML(blocks []markdown.ContentBlock, bgColor string) string {
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
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="2" name="Content"/>
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
    </p:spTree>
  </p:cSld>
</p:sld>`, bgXML, marginLeft, marginTop, contentWidth, contentHeight, body.String())
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
