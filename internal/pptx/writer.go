package pptx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"path"
	"strings"

	"github.com/pspoerri/marp2pptx/internal/markdown"
	"github.com/pspoerri/marp2pptx/internal/marp"
)

// SlideContent holds the converted content for a single slide.
type SlideContent struct {
	Blocks     []markdown.ContentBlock
	Directives marp.SlideDirectives
}

// slideImageData holds image references for a single slide.
type slideImageData struct {
	bgRef  *ImageRef
	fgRefs []ImageRef
}

// Write creates a PPTX file from converted slide content.
// If tpl is non-nil, the template's theme/master/layouts are used.
func Write(w io.Writer, meta marp.Meta, slides []SlideContent, tpl *TemplateData) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	// First pass: collect all images and assign media paths / relationship IDs
	allSlideImages := make([]slideImageData, len(slides))
	// When using a template, offset media counter to avoid colliding with template media files
	mediaCounter := 0
	if tpl != nil {
		mediaCounter = tpl.maxMediaIndex()
	}

	for i, slide := range slides {
		sid := &allSlideImages[i]
		relCounter := 2 // rId1 = slideLayout
		for _, block := range slide.Blocks {
			img, ok := block.(markdown.Image)
			if !ok || len(img.Data) == 0 {
				continue
			}
			mediaCounter++
			ext := imageExtFromURL(img.URL)
			ref := ImageRef{
				RelID:     fmt.Sprintf("rId%d", relCounter),
				MediaPath: fmt.Sprintf("ppt/media/image%d.%s", mediaCounter, ext),
				Image:     img,
			}
			ref.WidthPx, ref.HeightPx = imageDimensions(img.Data)
			relCounter++

			if img.Background {
				r := ref
				sid.bgRef = &r
			} else {
				sid.fgRefs = append(sid.fgRefs, ref)
			}
		}
	}

	// Determine layout for each slide
	slideLayouts := make([]LayoutType, len(slides))
	for i, slide := range slides {
		slideLayouts[i] = DetermineLayout(slide.Blocks, slide.Directives.Class)
	}

	if tpl != nil {
		return writeWithTemplate(zw, meta, slides, allSlideImages, slideLayouts, tpl)
	}
	return writeDefault(zw, meta, slides, allSlideImages, slideLayouts)
}

// writeDefault writes a PPTX with our built-in theme and layouts.
func writeDefault(zw *zip.Writer, meta marp.Meta, slides []SlideContent, allSlideImages []slideImageData, slideLayouts []LayoutType) error {
	// [Content_Types].xml
	if err := addFile(zw, "[Content_Types].xml", contentTypesXML(len(slides))); err != nil {
		return err
	}

	// _rels/.rels
	if err := addFile(zw, "_rels/.rels", topRelsXML()); err != nil {
		return err
	}

	// ppt/presentation.xml
	if err := addFile(zw, "ppt/presentation.xml", presentationXML(len(slides))); err != nil {
		return err
	}

	// ppt/_rels/presentation.xml.rels
	if err := addFile(zw, "ppt/_rels/presentation.xml.rels", presentationRelsXML(len(slides))); err != nil {
		return err
	}

	// Theme
	if err := addFile(zw, "ppt/theme/theme1.xml", themeXML()); err != nil {
		return err
	}

	// Slide master and layouts
	if err := addFile(zw, "ppt/slideMasters/slideMaster1.xml", slideMasterXML()); err != nil {
		return err
	}
	if err := addFile(zw, "ppt/slideMasters/_rels/slideMaster1.xml.rels", slideMasterRelsXML()); err != nil {
		return err
	}
	// Layout 1: Title Slide
	if err := addFile(zw, "ppt/slideLayouts/slideLayout1.xml", titleSlideLayoutXML()); err != nil {
		return err
	}
	if err := addFile(zw, "ppt/slideLayouts/_rels/slideLayout1.xml.rels", slideLayoutRelsXML()); err != nil {
		return err
	}
	// Layout 2: Title and Content
	if err := addFile(zw, "ppt/slideLayouts/slideLayout2.xml", titleContentLayoutXML()); err != nil {
		return err
	}
	if err := addFile(zw, "ppt/slideLayouts/_rels/slideLayout2.xml.rels", slideLayoutRelsXML()); err != nil {
		return err
	}
	// Layout 3: Blank
	if err := addFile(zw, "ppt/slideLayouts/slideLayout3.xml", blankLayoutXML()); err != nil {
		return err
	}
	if err := addFile(zw, "ppt/slideLayouts/_rels/slideLayout3.xml.rels", slideLayoutRelsXML()); err != nil {
		return err
	}

	// Slides
	return writeSlides(zw, meta, slides, allSlideImages, slideLayouts, func(lt LayoutType) string {
		return fmt.Sprintf("slideLayout%d.xml", int(lt))
	})
}

// writeWithTemplate writes a PPTX using files from a template.
func writeWithTemplate(zw *zip.Writer, meta marp.Meta, slides []SlideContent, allSlideImages []slideImageData, slideLayouts []LayoutType, tpl *TemplateData) error {
	// Write all template files (theme, masters, layouts, media, etc.)
	for name, data := range tpl.Files {
		if err := addFileBytes(zw, name, data); err != nil {
			return err
		}
	}

	// Generate metadata files incorporating template's non-slide entries
	if err := addFile(zw, "[Content_Types].xml", tpl.contentTypesXML(len(slides))); err != nil {
		return err
	}
	// Use template's original _rels/.rels (preserves docProps, thumbnail, etc.)
	if tpl.origRootRels != nil {
		if err := addFileBytes(zw, "_rels/.rels", tpl.origRootRels); err != nil {
			return err
		}
	} else {
		if err := addFile(zw, "_rels/.rels", topRelsXML()); err != nil {
			return err
		}
	}
	if err := addFile(zw, "ppt/presentation.xml", tpl.presentationXML(len(slides))); err != nil {
		return err
	}
	if err := addFile(zw, "ppt/_rels/presentation.xml.rels", tpl.presentationRelsXML(len(slides))); err != nil {
		return err
	}

	// Slides
	return writeSlides(zw, meta, slides, allSlideImages, slideLayouts, func(lt LayoutType) string {
		return tpl.LayoutFile(lt)
	})
}

// writeSlides writes slide XML, rels, and embedded images.
func writeSlides(zw *zip.Writer, meta marp.Meta, slides []SlideContent, allSlideImages []slideImageData, slideLayouts []LayoutType, layoutFile func(LayoutType) string) error {
	for i, slide := range slides {
		sid := allSlideImages[i]

		// Embed image files
		if sid.bgRef != nil {
			if err := addFileBytes(zw, sid.bgRef.MediaPath, sid.bgRef.Image.Data); err != nil {
				return err
			}
		}
		for _, ref := range sid.fgRefs {
			if err := addFileBytes(zw, ref.MediaPath, ref.Image.Data); err != nil {
				return err
			}
		}

		bgColor := slide.Directives.BackgroundColor
		if bgColor == "" {
			bgColor = meta.BackgroundColor
		}
		layout := slideLayouts[i]
		slideXML := generateSlideXML(slide.Blocks, bgColor, sid.bgRef, sid.fgRefs, layout)
		slidePath := fmt.Sprintf("ppt/slides/slide%d.xml", i+1)
		if err := addFile(zw, slidePath, slideXML); err != nil {
			return err
		}

		relsPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", i+1)
		lf := layoutFile(layout)
		if err := addFile(zw, relsPath, slideRelsXMLWithImages(lf, sid.bgRef, sid.fgRefs)); err != nil {
			return err
		}
	}
	return nil
}

func addFile(zw *zip.Writer, name, content string) error {
	f, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write([]byte(content))
	return err
}

func addFileBytes(zw *zip.Writer, name string, data []byte) error {
	f, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = f.Write(data)
	return err
}

func imageExtFromURL(url string) string {
	ext := strings.ToLower(path.Ext(url))
	ext = strings.TrimPrefix(ext, ".")
	switch ext {
	case "jpg":
		return "jpeg"
	case "png", "jpeg", "gif":
		return ext
	default:
		return "png"
	}
}

func imageDimensions(data []byte) (width, height int) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

// ---------------------------------------------------------------------------
// Default boilerplate XML generation
// ---------------------------------------------------------------------------

func contentTypesXML(slideCount int) string {
	var overrides strings.Builder
	for i := 1; i <= slideCount; i++ {
		overrides.WriteString(fmt.Sprintf(`  <Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>
`, i))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="png" ContentType="image/png"/>
  <Default Extension="jpeg" ContentType="image/jpeg"/>
  <Default Extension="jpg" ContentType="image/jpeg"/>
  <Default Extension="gif" ContentType="image/gif"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
  <Override PartName="/ppt/theme/theme1.xml" ContentType="application/vnd.openxmlformats-officedocument.theme+xml"/>
  <Override PartName="/ppt/slideMasters/slideMaster1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml"/>
  <Override PartName="/ppt/slideLayouts/slideLayout1.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
  <Override PartName="/ppt/slideLayouts/slideLayout2.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
  <Override PartName="/ppt/slideLayouts/slideLayout3.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml"/>
%s</Types>`, overrides.String())
}

func topRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>`
}

func presentationXML(slideCount int) string {
	var sldIdLst strings.Builder
	for i := 1; i <= slideCount; i++ {
		sldIdLst.WriteString(fmt.Sprintf(`    <p:sldId id="%d" r:id="rId%d"/>
`, 255+i, i))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
                xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
                xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:sldMasterIdLst>
    <p:sldMasterId id="2147483648" r:id="rId%d"/>
  </p:sldMasterIdLst>
  <p:sldIdLst>
%s  </p:sldIdLst>
  <p:sldSz cx="%d" cy="%d"/>
  <p:notesSz cx="%d" cy="%d"/>
</p:presentation>`, slideCount+1, sldIdLst.String(), slideWidth, slideHeight, slideWidth, slideHeight)
}

func presentationRelsXML(slideCount int) string {
	var rels strings.Builder
	for i := 1; i <= slideCount; i++ {
		rels.WriteString(fmt.Sprintf(`  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide" Target="slides/slide%d.xml"/>
`, i, i))
	}
	masterID := slideCount + 1
	themeID := slideCount + 2
	rels.WriteString(fmt.Sprintf(`  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="slideMasters/slideMaster1.xml"/>
`, masterID))
	rels.WriteString(fmt.Sprintf(`  <Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="theme/theme1.xml"/>
`, themeID))

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
%s</Relationships>`, rels.String())
}

func slideRelsXMLWithImages(layoutFile string, bgRef *ImageRef, fgRefs []ImageRef) string {
	var rels strings.Builder
	rels.WriteString(fmt.Sprintf(`  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/%s"/>
`, layoutFile))
	if bgRef != nil {
		rels.WriteString(fmt.Sprintf(`  <Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/%s"/>
`, bgRef.RelID, path.Base(bgRef.MediaPath)))
	}
	for _, ref := range fgRefs {
		rels.WriteString(fmt.Sprintf(`  <Relationship Id="%s" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/image" Target="../media/%s"/>
`, ref.RelID, path.Base(ref.MediaPath)))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
%s</Relationships>`, rels.String())
}

// ---------------------------------------------------------------------------
// Slide master and layouts
// ---------------------------------------------------------------------------

func slideMasterXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldMaster xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
             xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
             xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:cSld>
    <p:bg>
      <p:bgPr>
        <a:solidFill><a:schemeClr val="bg1"/></a:solidFill>
        <a:effectLst/>
      </p:bgPr>
    </p:bg>
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
    </p:spTree>
  </p:cSld>
  <p:clrMap bg1="lt1" tx1="dk1" bg2="lt2" tx2="dk2" accent1="accent1" accent2="accent2"
            accent3="accent3" accent4="accent4" accent5="accent5" accent6="accent6" hlink="hlink" folHlink="folHlink"/>
  <p:txStyles>
    <p:titleStyle>
      <a:lvl1pPr algn="l">
        <a:defRPr sz="4400" b="1" kern="1200">
          <a:solidFill><a:schemeClr val="tx1"/></a:solidFill>
          <a:latin typeface="+mj-lt"/>
          <a:ea typeface="+mj-ea"/>
          <a:cs typeface="+mj-cs"/>
        </a:defRPr>
      </a:lvl1pPr>
    </p:titleStyle>
    <p:bodyStyle>
      <a:lvl1pPr marL="228600" indent="-228600" algn="l">
        <a:buChar char="&#x2022;"/>
        <a:defRPr sz="1800" kern="1200">
          <a:solidFill><a:schemeClr val="tx1"/></a:solidFill>
          <a:latin typeface="+mn-lt"/>
          <a:ea typeface="+mn-ea"/>
          <a:cs typeface="+mn-cs"/>
        </a:defRPr>
      </a:lvl1pPr>
      <a:lvl2pPr marL="457200" indent="-228600" algn="l">
        <a:buChar char="&#x2013;"/>
        <a:defRPr sz="1600" kern="1200">
          <a:solidFill><a:schemeClr val="tx1"/></a:solidFill>
          <a:latin typeface="+mn-lt"/>
        </a:defRPr>
      </a:lvl2pPr>
    </p:bodyStyle>
  </p:txStyles>
  <p:sldLayoutIdLst>
    <p:sldLayoutId id="2147483649" r:id="rId1"/>
    <p:sldLayoutId id="2147483650" r:id="rId2"/>
    <p:sldLayoutId id="2147483651" r:id="rId3"/>
  </p:sldLayoutIdLst>
</p:sldMaster>`
}

func slideMasterRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout2.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout3.xml"/>
  <Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>`
}

// titleSlideLayoutXML returns layout 1: Title Slide (centered title + subtitle).
func titleSlideLayoutXML() string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
             xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
             xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
             type="ctrTitle">
  <p:cSld name="Title Slide">
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
          <p:cNvPr id="2" name="Title 1"/>
          <p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr>
          <p:nvPr><p:ph type="ctrTitle"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0" anchor="ctr"/>
          <a:lstStyle>
            <a:lvl1pPr algn="ctr">
              <a:defRPr sz="4400" b="1"/>
            </a:lvl1pPr>
          </a:lstStyle>
          <a:p><a:endParaRPr lang="en-US"/></a:p>
        </p:txBody>
      </p:sp>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="3" name="Subtitle 2"/>
          <p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr>
          <p:nvPr><p:ph type="subTitle" idx="1"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0" anchor="t"/>
          <a:lstStyle>
            <a:lvl1pPr algn="ctr">
              <a:defRPr sz="2000"/>
            </a:lvl1pPr>
          </a:lstStyle>
          <a:p><a:endParaRPr lang="en-US"/></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sldLayout>`, ctrTitleX, ctrTitleY, ctrTitleCX, ctrTitleCY,
		subTitleX, subTitleY, subTitleCX, subTitleCY)
}

// titleContentLayoutXML returns layout 2: Title and Content.
func titleContentLayoutXML() string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
             xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
             xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
             type="obj">
  <p:cSld name="Title and Content">
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
          <p:cNvPr id="2" name="Title 1"/>
          <p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr>
          <p:nvPr><p:ph type="title"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0" anchor="b"/>
          <a:lstStyle>
            <a:lvl1pPr algn="l">
              <a:defRPr sz="4400" b="1"/>
            </a:lvl1pPr>
          </a:lstStyle>
          <a:p><a:endParaRPr lang="en-US"/></a:p>
        </p:txBody>
      </p:sp>
      <p:sp>
        <p:nvSpPr>
          <p:cNvPr id="3" name="Content Placeholder 2"/>
          <p:cNvSpPr><a:spLocks noGrp="1"/></p:cNvSpPr>
          <p:nvPr><p:ph idx="1"/></p:nvPr>
        </p:nvSpPr>
        <p:spPr>
          <a:xfrm>
            <a:off x="%d" y="%d"/>
            <a:ext cx="%d" cy="%d"/>
          </a:xfrm>
        </p:spPr>
        <p:txBody>
          <a:bodyPr wrap="square" rtlCol="0" anchor="t"/>
          <a:lstStyle>
            <a:lvl1pPr>
              <a:defRPr sz="1800"/>
            </a:lvl1pPr>
          </a:lstStyle>
          <a:p><a:endParaRPr lang="en-US"/></a:p>
        </p:txBody>
      </p:sp>
    </p:spTree>
  </p:cSld>
</p:sldLayout>`, marginLeft, titlePlcY, contentWidth, titlePlcCY,
		marginLeft, bodyAreaY, contentWidth, bodyAreaCY)
}

// blankLayoutXML returns layout 3: Blank.
func blankLayoutXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sldLayout xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
             xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
             xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
             type="blank">
  <p:cSld name="Blank">
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
    </p:spTree>
  </p:cSld>
</p:sldLayout>`
}

func slideLayoutRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster" Target="../slideMasters/slideMaster1.xml"/>
</Relationships>`
}

func themeXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<a:theme xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" name="marp2pptx">
  <a:themeElements>
    <a:clrScheme name="Default">
      <a:dk1><a:srgbClr val="000000"/></a:dk1>
      <a:lt1><a:srgbClr val="FFFFFF"/></a:lt1>
      <a:dk2><a:srgbClr val="44546A"/></a:dk2>
      <a:lt2><a:srgbClr val="E7E6E6"/></a:lt2>
      <a:accent1><a:srgbClr val="4472C4"/></a:accent1>
      <a:accent2><a:srgbClr val="ED7D31"/></a:accent2>
      <a:accent3><a:srgbClr val="A5A5A5"/></a:accent3>
      <a:accent4><a:srgbClr val="FFC000"/></a:accent4>
      <a:accent5><a:srgbClr val="5B9BD5"/></a:accent5>
      <a:accent6><a:srgbClr val="70AD47"/></a:accent6>
      <a:hlink><a:srgbClr val="0563C1"/></a:hlink>
      <a:folHlink><a:srgbClr val="954F72"/></a:folHlink>
    </a:clrScheme>
    <a:fontScheme name="Default">
      <a:majorFont><a:latin typeface="Calibri"/><a:ea typeface=""/><a:cs typeface=""/></a:majorFont>
      <a:minorFont><a:latin typeface="Calibri"/><a:ea typeface=""/><a:cs typeface=""/></a:minorFont>
    </a:fontScheme>
    <a:fmtScheme name="Default">
      <a:fillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:fillStyleLst>
      <a:lnStyleLst>
        <a:ln w="6350"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
        <a:ln w="6350"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
        <a:ln w="6350"><a:solidFill><a:schemeClr val="phClr"/></a:solidFill></a:ln>
      </a:lnStyleLst>
      <a:effectStyleLst>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
        <a:effectStyle><a:effectLst/></a:effectStyle>
      </a:effectStyleLst>
      <a:bgFillStyleLst>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
        <a:solidFill><a:schemeClr val="phClr"/></a:solidFill>
      </a:bgFillStyleLst>
    </a:fmtScheme>
  </a:themeElements>
</a:theme>`
}
