package pptx

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// TemplateData holds files and metadata extracted from a .pptx/.potx template.
type TemplateData struct {
	// Files to copy into the output (theme, masters, layouts, media, etc.)
	// Excludes slides, notesSlides, presentation.xml, presentation.xml.rels, [Content_Types].xml, _rels/.rels.
	Files map[string][]byte

	// Layout mapping: our layout type → template layout filename (e.g., "slideLayout8.xml")
	layouts map[LayoutType]string

	// Non-slide relationships from the template's presentation.xml.rels
	nonSlideRels []parsedRel

	// Slide size from the template
	slideSizeXML string

	// Default text style from the template's presentation.xml
	defaultTextStyleXML string

	// Presentation element attributes (e.g. saveSubsetFonts, autoCompressPictures)
	presentationAttrs string

	// Original [Content_Types].xml bytes (for preserving all part overrides)
	origContentTypes []byte

	// Original _rels/.rels bytes (for preserving docProps references)
	origRootRels []byte
}

type parsedRel struct {
	ID     string
	Type   string
	Target string
}

// LayoutFile returns the layout filename for the given layout type.
func (t *TemplateData) LayoutFile(lt LayoutType) string {
	if f, ok := t.layouts[lt]; ok {
		return f
	}
	// Fallback: use the first available layout
	for _, f := range t.layouts {
		return f
	}
	return "slideLayout1.xml"
}

// maxMediaIndex scans template files for existing media and returns the highest
// image number, so new images can be numbered without collision.
func (t *TemplateData) maxMediaIndex() int {
	maxIdx := 0
	mediaRe := regexp.MustCompile(`^ppt/media/image(\d+)\.`)
	for name := range t.Files {
		if m := mediaRe.FindStringSubmatch(name); len(m) > 1 {
			if n, err := strconv.Atoi(m[1]); err == nil && n > maxIdx {
				maxIdx = n
			}
		}
	}
	return maxIdx
}

// LoadTemplate reads a .pptx or .potx file and extracts theme data.
func LoadTemplate(path string) (*TemplateData, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("opening template: %w", err)
	}
	defer r.Close()

	td := &TemplateData{
		Files:   make(map[string][]byte),
		layouts: make(map[LayoutType]string),
	}

	// Read all files from the template
	allFiles := make(map[string][]byte)
	for _, f := range r.File {
		data, err := readZipEntry(f)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f.Name, err)
		}
		allFiles[f.Name] = data
	}

	// Store originals before filtering
	if ct, ok := allFiles["[Content_Types].xml"]; ok {
		td.origContentTypes = ct
	}
	if rr, ok := allFiles["_rels/.rels"]; ok {
		td.origRootRels = rr
	}

	// Parse slide layouts to detect types by analyzing placeholders
	var layoutInfos []layoutInfo
	for name, data := range allFiles {
		if !strings.HasPrefix(name, "ppt/slideLayouts/slideLayout") ||
			!strings.HasSuffix(name, ".xml") ||
			strings.Contains(name, "_rels") {
			continue
		}
		base := name[len("ppt/slideLayouts/"):]
		layoutInfos = append(layoutInfos, parseLayoutInfo(base, data))
	}
	td.selectLayouts(layoutInfos)

	// Parse presentation.xml for slide size, default text style, and attributes
	if presXML, ok := allFiles["ppt/presentation.xml"]; ok {
		presStr := string(presXML)
		td.slideSizeXML = extractSlideSizeXML(presStr)
		td.defaultTextStyleXML = extractDefaultTextStyleXML(presStr)
		td.presentationAttrs = extractPresentationAttrs(presStr)
	}

	// Parse presentation.xml.rels for non-slide relationships
	if presRels, ok := allFiles["ppt/_rels/presentation.xml.rels"]; ok {
		td.nonSlideRels = parseNonSlideRels(presRels)
	}

	// Store all files except slides, notesSlides, and metadata we'll regenerate
	for name, data := range allFiles {
		if isTemplateSlideFile(name) || isMetadataFile(name) {
			continue
		}
		td.Files[name] = data
	}

	return td, nil
}

// ---------------------------------------------------------------------------
// Layout detection — estimates layout type from placeholder text boxes
// ---------------------------------------------------------------------------

// layoutInfo holds parsed information about a slide layout.
type layoutInfo struct {
	filename     string
	name         string // from <p:cSld name="...">
	ooxmlType    string // from type="..." on <p:sldLayout>
	placeholders []phInfo
}

// phInfo describes a single placeholder in a layout.
type phInfo struct {
	phType string // "title", "ctrTitle", "subTitle", "body", "sldNum", "pic", etc.
	idx    string // idx attribute value, empty if not present
}

var (
	phRegex      = regexp.MustCompile(`<p:ph\b([^>]*?)(?:/>|>)`)
	layoutTypeRe = regexp.MustCompile(`<p:sldLayout[^>]*\btype="([^"]+)"`)
	layoutNameRe = regexp.MustCompile(`<p:cSld[^>]*\bname="([^"]+)"`)
	attrTypeRe   = regexp.MustCompile(`\btype="([^"]+)"`)
	attrIdxRe    = regexp.MustCompile(`\bidx="([^"]+)"`)
)

func parseLayoutInfo(filename string, data []byte) layoutInfo {
	s := string(data)
	li := layoutInfo{filename: filename}

	if m := layoutTypeRe.FindStringSubmatch(s); len(m) > 1 {
		li.ooxmlType = m[1]
	}
	if m := layoutNameRe.FindStringSubmatch(s); len(m) > 1 {
		li.name = m[1]
	}

	for _, m := range phRegex.FindAllStringSubmatch(s, -1) {
		attrs := m[1]
		ph := phInfo{phType: "body"} // no type attr = body placeholder
		if tm := attrTypeRe.FindStringSubmatch(attrs); len(tm) > 1 {
			ph.phType = tm[1]
		}
		if im := attrIdxRe.FindStringSubmatch(attrs); len(im) > 1 {
			ph.idx = im[1]
		}
		li.placeholders = append(li.placeholders, ph)
	}

	return li
}

// selectLayouts maps template layouts to our LayoutType enum using multi-level detection.
// Pass 1: standard OOXML type attribute (works for standard templates).
// Pass 2: placeholder composition — analyzes text boxes to estimate layout type.
// Pass 3: name heuristics — matches by layout name patterns.
func (t *TemplateData) selectLayouts(layouts []layoutInfo) {
	// Pass 1: OOXML type attribute (standard templates)
	ooxmlMap := map[string]LayoutType{
		"ctrTitle": LayoutTitleSlide,
		"title":    LayoutTitleSlide,
		"secHead":  LayoutTitleSlide,
		"obj":      LayoutTitleContent,
		"twoObj":   LayoutTitleContent,
		"blank":    LayoutBlank,
	}
	for _, li := range layouts {
		if li.ooxmlType == "" {
			continue
		}
		if lt, ok := ooxmlMap[li.ooxmlType]; ok {
			if _, already := t.layouts[lt]; !already {
				t.layouts[lt] = li.filename
			}
		}
	}
	if len(t.layouts) >= 3 {
		return
	}

	// Pass 2: estimate layout type from placeholder text boxes.
	// Score each candidate — prefer layouts with fewer extraneous placeholders.
	type candidate struct {
		filename string
		score    int // higher = better (100 - total placeholders)
	}
	var bestTC, bestTS *candidate
	for _, li := range layouts {
		hasCtrTitle, hasTitle, hasSubTitle, hasBody1 := false, false, false, false
		for _, ph := range li.placeholders {
			switch ph.phType {
			case "ctrTitle":
				hasCtrTitle = true
			case "title":
				hasTitle = true
			case "subTitle":
				hasSubTitle = true
			case "body":
				if ph.idx == "1" {
					hasBody1 = true
				}
			}
		}
		score := 100 - len(li.placeholders) // simpler layouts score higher
		if _, ok := t.layouts[LayoutTitleContent]; !ok {
			if hasTitle && hasBody1 {
				if bestTC == nil || score > bestTC.score {
					bestTC = &candidate{li.filename, score}
				}
			}
		}
		if _, ok := t.layouts[LayoutTitleSlide]; !ok {
			if hasCtrTitle || hasSubTitle {
				if bestTS == nil || score > bestTS.score {
					bestTS = &candidate{li.filename, score}
				}
			}
		}
	}
	if bestTC != nil {
		t.layouts[LayoutTitleContent] = bestTC.filename
	}
	if bestTS != nil {
		t.layouts[LayoutTitleSlide] = bestTS.filename
	}

	// Pass 3: name heuristics
	nameLower := func(li layoutInfo) string { return strings.ToLower(strings.TrimSpace(li.name)) }

	if _, ok := t.layouts[LayoutTitleContent]; !ok {
		for _, li := range layouts {
			n := nameLower(li)
			if strings.Contains(n, "content") {
				t.layouts[LayoutTitleContent] = li.filename
				break
			}
		}
	}
	if _, ok := t.layouts[LayoutTitleSlide]; !ok {
		// Prefer "subtitle" layouts
		for _, li := range layouts {
			n := nameLower(li)
			if strings.Contains(n, "subtitle") {
				t.layouts[LayoutTitleSlide] = li.filename
				break
			}
		}
	}
	if _, ok := t.layouts[LayoutTitleSlide]; !ok {
		// Fall back to layouts with "title" but not "content"
		for _, li := range layouts {
			n := nameLower(li)
			hasTitle := false
			for _, ph := range li.placeholders {
				if ph.phType == "title" {
					hasTitle = true
				}
			}
			if hasTitle && strings.Contains(n, "title") && !strings.Contains(n, "content") {
				t.layouts[LayoutTitleSlide] = li.filename
				break
			}
		}
	}
	if _, ok := t.layouts[LayoutBlank]; !ok {
		for _, li := range layouts {
			n := nameLower(li)
			if strings.Contains(n, "blank") {
				t.layouts[LayoutBlank] = li.filename
				break
			}
		}
	}
	// For Blank, also try title-only layouts (just title + sldNum, no body/pic)
	if _, ok := t.layouts[LayoutBlank]; !ok {
		var bestBlank *candidate
		for _, li := range layouts {
			hasTitle, hasBody, hasPic := false, false, false
			for _, ph := range li.placeholders {
				switch ph.phType {
				case "title":
					hasTitle = true
				case "body":
					hasBody = true
				case "pic":
					hasPic = true
				}
			}
			if hasTitle && !hasBody && !hasPic {
				score := 100 - len(li.placeholders)
				if bestBlank == nil || score > bestBlank.score {
					bestBlank = &candidate{li.filename, score}
				}
			}
		}
		if bestBlank != nil {
			t.layouts[LayoutBlank] = bestBlank.filename
		}
	}

	// Final fallback: use TitleContent layout if available, else first layout by name
	for _, lt := range []LayoutType{LayoutTitleSlide, LayoutTitleContent, LayoutBlank} {
		if t.layouts[lt] != "" {
			continue
		}
		// Prefer TitleContent as fallback (most generic)
		if tc, ok := t.layouts[LayoutTitleContent]; ok {
			t.layouts[lt] = tc
			continue
		}
		// Last resort: pick first layout alphabetically for determinism
		best := ""
		for _, li := range layouts {
			if best == "" || li.filename < best {
				best = li.filename
			}
		}
		if best != "" {
			t.layouts[lt] = best
		}
	}
}

// ---------------------------------------------------------------------------
// Content types — preserves template's extensions and non-slide overrides
// ---------------------------------------------------------------------------

type xmlContentTypes struct {
	XMLName   xml.Name        `xml:"Types"`
	Defaults  []xmlCTDefault  `xml:"Default"`
	Overrides []xmlCTOverride `xml:"Override"`
}

type xmlCTDefault struct {
	Extension   string `xml:"Extension,attr"`
	ContentType string `xml:"ContentType,attr"`
}

type xmlCTOverride struct {
	PartName    string `xml:"PartName,attr"`
	ContentType string `xml:"ContentType,attr"`
}

// contentTypesXML generates [Content_Types].xml preserving the template's
// extensions and non-slide overrides, replacing slides with ours.
func (t *TemplateData) contentTypesXML(slideCount int) string {
	var types xmlContentTypes
	if err := xml.Unmarshal(t.origContentTypes, &types); err != nil {
		return fallbackContentTypesXML(t, slideCount)
	}

	var sb strings.Builder
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" standalone=\"yes\"?>\n")
	sb.WriteString("<Types xmlns=\"http://schemas.openxmlformats.org/package/2006/content-types\">")

	for _, d := range types.Defaults {
		sb.WriteString(fmt.Sprintf(`<Default Extension="%s" ContentType="%s"/>`,
			d.Extension, d.ContentType))
	}

	for _, o := range types.Overrides {
		// Skip template's slide and notesSlide entries (we generate our own slides)
		if strings.HasPrefix(o.PartName, "/ppt/slides/") ||
			strings.HasPrefix(o.PartName, "/ppt/notesSlides/") {
			continue
		}
		ct := o.ContentType
		// Convert template content type to presentation
		if strings.Contains(ct, "template.main") {
			ct = "application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"
		}
		sb.WriteString(fmt.Sprintf(`<Override PartName="%s" ContentType="%s"/>`,
			o.PartName, ct))
	}

	for i := 1; i <= slideCount; i++ {
		sb.WriteString(fmt.Sprintf(`<Override PartName="/ppt/slides/slide%d.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.slide+xml"/>`, i))
	}

	sb.WriteString("</Types>")
	return sb.String()
}

func fallbackContentTypesXML(t *TemplateData, slideCount int) string {
	var overrides strings.Builder
	for i := 1; i <= slideCount; i++ {
		overrides.WriteString(fmt.Sprintf("  <Override PartName=\"/ppt/slides/slide%d.xml\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.slide+xml\"/>\n", i))
	}
	for name := range t.Files {
		if strings.HasPrefix(name, "ppt/theme/") && strings.HasSuffix(name, ".xml") && !strings.Contains(name, "_rels") {
			overrides.WriteString(fmt.Sprintf("  <Override PartName=\"/%s\" ContentType=\"application/vnd.openxmlformats-officedocument.theme+xml\"/>\n", name))
		} else if strings.HasPrefix(name, "ppt/slideMasters/") && strings.HasSuffix(name, ".xml") && !strings.Contains(name, "_rels") {
			overrides.WriteString(fmt.Sprintf("  <Override PartName=\"/%s\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.slideMaster+xml\"/>\n", name))
		} else if strings.HasPrefix(name, "ppt/slideLayouts/") && strings.HasSuffix(name, ".xml") && !strings.Contains(name, "_rels") {
			overrides.WriteString(fmt.Sprintf("  <Override PartName=\"/%s\" ContentType=\"application/vnd.openxmlformats-officedocument.presentationml.slideLayout+xml\"/>\n", name))
		}
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="png" ContentType="image/png"/>
  <Default Extension="jpeg" ContentType="image/jpeg"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
%s</Types>`, overrides.String())
}

// ---------------------------------------------------------------------------
// Presentation metadata — properly coordinates master rIds with rels
// ---------------------------------------------------------------------------

const slideMasterRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slideMaster"

// presentationXML generates presentation.xml with properly coordinated rIds.
// The master rId references are computed from the position of slideMaster entries
// in nonSlideRels, matching the IDs assigned in presentationRelsXML.
const notesMasterRelType = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/notesMaster"

func (t *TemplateData) presentationXML(slideCount int) string {
	var sldIdLst strings.Builder
	for i := 1; i <= slideCount; i++ {
		sldIdLst.WriteString(fmt.Sprintf("    <p:sldId id=\"%d\" r:id=\"rId%d\"/>\n", 255+i, i))
	}

	// Build master ID list by finding slideMaster rels and computing their new rIds.
	// Non-slide rels are assigned rId(slideCount+1+i) in presentationRelsXML.
	var masterList strings.Builder
	masterList.WriteString("<p:sldMasterIdLst>\n")
	masterID := 2147483648
	for i, rel := range t.nonSlideRels {
		if rel.Type == slideMasterRelType {
			newRId := slideCount + 1 + i
			masterList.WriteString(fmt.Sprintf("    <p:sldMasterId id=\"%d\" r:id=\"rId%d\"/>\n", masterID, newRId))
			masterID++
		}
	}
	masterList.WriteString("  </p:sldMasterIdLst>")

	// Build notesMasterIdLst if template has a notesMaster relationship
	var notesMasterLst string
	for i, rel := range t.nonSlideRels {
		if rel.Type == notesMasterRelType {
			newRId := slideCount + 1 + i
			notesMasterLst = fmt.Sprintf("\n  <p:notesMasterIdLst><p:notesMasterId r:id=\"rId%d\"/></p:notesMasterIdLst>", newRId)
			break
		}
	}

	sizeXML := t.slideSizeXML
	if sizeXML == "" {
		sizeXML = fmt.Sprintf("<p:sldSz cx=\"%d\" cy=\"%d\"/>\n  <p:notesSz cx=\"%d\" cy=\"%d\"/>",
			slideWidth, slideHeight, slideWidth, slideHeight)
	}

	// Include default text style if present in the template
	defaultTextStyle := ""
	if t.defaultTextStyleXML != "" {
		defaultTextStyle = "\n  " + t.defaultTextStyleXML
	}

	// Include presentation attributes from the template
	presAttrs := ""
	if t.presentationAttrs != "" {
		presAttrs = " " + t.presentationAttrs
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
                xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"
                xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"%s>
  %s
  <p:sldIdLst>
%s  </p:sldIdLst>%s
  %s%s
</p:presentation>`, presAttrs, masterList.String(), sldIdLst.String(), notesMasterLst, sizeXML, defaultTextStyle)
}

// presentationRelsXML generates presentation.xml.rels combining our slides with template rels.
func (t *TemplateData) presentationRelsXML(slideCount int) string {
	var rels strings.Builder
	// Our slide relationships: rId1..rIdN
	for i := 1; i <= slideCount; i++ {
		rels.WriteString(fmt.Sprintf("  <Relationship Id=\"rId%d\" Type=\"http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide\" Target=\"slides/slide%d.xml\"/>\n", i, i))
	}
	// Template's non-slide relationships with remapped IDs: rId(N+1)..
	for i, rel := range t.nonSlideRels {
		newID := fmt.Sprintf("rId%d", slideCount+1+i)
		rels.WriteString(fmt.Sprintf("  <Relationship Id=\"%s\" Type=\"%s\" Target=\"%s\"/>\n",
			newID, rel.Type, rel.Target))
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
%s</Relationships>`, rels.String())
}

// ---------------------------------------------------------------------------
// XML parsing helpers
// ---------------------------------------------------------------------------

// xmlRelationships for parsing .rels files.
type xmlRelationships struct {
	XMLName xml.Name          `xml:"Relationships"`
	Rels    []xmlRelationship `xml:"Relationship"`
}

type xmlRelationship struct {
	ID     string `xml:"Id,attr"`
	Type   string `xml:"Type,attr"`
	Target string `xml:"Target,attr"`
}

func parseNonSlideRels(data []byte) []parsedRel {
	var rels xmlRelationships
	if err := xml.Unmarshal(data, &rels); err != nil {
		return nil
	}
	var result []parsedRel
	slideRelType := "http://schemas.openxmlformats.org/officeDocument/2006/relationships/slide"
	for _, r := range rels.Rels {
		if r.Type != slideRelType {
			result = append(result, parsedRel{ID: r.ID, Type: r.Type, Target: r.Target})
		}
	}
	return result
}

func isTemplateSlideFile(name string) bool {
	return strings.HasPrefix(name, "ppt/slides/") ||
		strings.HasPrefix(name, "ppt/notesSlides/")
}

func isMetadataFile(name string) bool {
	switch name {
	case "[Content_Types].xml",
		"ppt/presentation.xml",
		"ppt/_rels/presentation.xml.rels",
		"_rels/.rels":
		return true
	}
	return false
}

func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

var sldSzRe = regexp.MustCompile(`<p:sldSz[^/]*/?>`)
var notesSzRe = regexp.MustCompile(`<p:notesSz[^/]*/?>`)
var defaultTextStyleRe = regexp.MustCompile(`<p:defaultTextStyle>[\s\S]*?</p:defaultTextStyle>`)
var presAttrsRe = regexp.MustCompile(`<p:presentation\b[^>]*>`)

func extractSlideSizeXML(xmlStr string) string {
	sldSz := sldSzRe.FindString(xmlStr)
	notesSz := notesSzRe.FindString(xmlStr)
	if sldSz == "" {
		return ""
	}
	result := sldSz
	if notesSz != "" {
		result += "\n  " + notesSz
	}
	return result
}

func extractDefaultTextStyleXML(xmlStr string) string {
	return defaultTextStyleRe.FindString(xmlStr)
}

func extractPresentationAttrs(xmlStr string) string {
	m := presAttrsRe.FindString(xmlStr)
	if m == "" {
		return ""
	}
	// Extract individual attributes we want to preserve
	var attrs []string
	attrRe := regexp.MustCompile(`\b(saveSubsetFonts|autoCompressPictures)="([^"]*)"`)
	for _, am := range attrRe.FindAllStringSubmatch(m, -1) {
		attrs = append(attrs, fmt.Sprintf(`%s="%s"`, am[1], am[2]))
	}
	return strings.Join(attrs, " ")
}
