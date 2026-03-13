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

// Write creates a PPTX file from converted slide content.
func Write(w io.Writer, meta marp.Meta, slides []SlideContent) error {
	zw := zip.NewWriter(w)
	defer zw.Close()

	// First pass: collect all images and assign media paths / relationship IDs
	type slideImageData struct {
		bgRef  *ImageRef
		fgRefs []ImageRef
	}
	allSlideImages := make([]slideImageData, len(slides))
	mediaCounter := 0

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

	// Slide master and layout
	if err := addFile(zw, "ppt/slideMasters/slideMaster1.xml", slideMasterXML()); err != nil {
		return err
	}
	if err := addFile(zw, "ppt/slideMasters/_rels/slideMaster1.xml.rels", slideMasterRelsXML()); err != nil {
		return err
	}
	if err := addFile(zw, "ppt/slideLayouts/slideLayout1.xml", slideLayoutXML()); err != nil {
		return err
	}
	if err := addFile(zw, "ppt/slideLayouts/_rels/slideLayout1.xml.rels", slideLayoutRelsXML()); err != nil {
		return err
	}

	// Slides
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
		slideXML := generateSlideXML(slide.Blocks, bgColor, sid.bgRef, sid.fgRefs)
		slidePath := fmt.Sprintf("ppt/slides/slide%d.xml", i+1)
		if err := addFile(zw, slidePath, slideXML); err != nil {
			return err
		}

		relsPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", i+1)
		if err := addFile(zw, relsPath, slideRelsXMLWithImages(sid.bgRef, sid.fgRefs)); err != nil {
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

func slideRelsXMLWithImages(bgRef *ImageRef, fgRefs []ImageRef) string {
	var rels strings.Builder
	rels.WriteString(`  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
`)
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
  <p:sldLayoutIdLst>
    <p:sldLayoutId id="2147483649" r:id="rId1"/>
  </p:sldLayoutIdLst>
</p:sldMaster>`
}

func slideMasterRelsXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideLayout" Target="../slideLayouts/slideLayout1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/theme" Target="../theme/theme1.xml"/>
</Relationships>`
}

func slideLayoutXML() string {
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
