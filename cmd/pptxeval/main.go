package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

func main() {
	lintMode := flag.Bool("lint", false, "lint mode: validate PPTX structure and exit with non-zero on errors")
	slideNum := flag.Int("slide", 0, "show detailed XML for a specific slide (1-based)")
	textOnly := flag.Bool("text", false, "extract text content only")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pptxeval [flags] <file.pptx>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	pptxPath := flag.Arg(0)
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s: %v\n", pptxPath, err)
		os.Exit(1)
	}
	defer r.Close()

	files := indexFiles(r)

	if *lintMode {
		os.Exit(runLint(files, pptxPath))
	}
	if *textOnly {
		printText(files)
		return
	}
	if *slideNum > 0 {
		printSlideDetail(files, *slideNum)
		return
	}
	printReport(files, pptxPath)
}

// indexFiles creates a map from path to zip.File.
func indexFiles(r *zip.ReadCloser) map[string]*zip.File {
	m := make(map[string]*zip.File)
	for _, f := range r.File {
		m[f.Name] = f
	}
	return m
}

func readFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// --- Report mode ---

func printReport(files map[string]*zip.File, pptxPath string) {
	slideCount := countSlides(files)
	mediaFiles := collectPrefix(files, "ppt/media/")

	fmt.Printf("=== %s ===\n", pptxPath)
	fmt.Printf("Slides: %d\n", slideCount)
	fmt.Printf("Media:  %d\n", len(mediaFiles))
	fmt.Println()

	// Per-slide details
	for i := 1; i <= slideCount; i++ {
		fmt.Printf("--- Slide %d ---\n", i)
		slidePath := fmt.Sprintf("ppt/slides/slide%d.xml", i)
		relsPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", i)

		if sf, ok := files[slidePath]; ok {
			data, _ := readFile(sf)
			xml := string(data)

			// Text content
			texts := extractAllText(xml)
			if len(texts) > 0 {
				fmt.Printf("  Text: %s\n", strings.Join(texts, " | "))
			}

			// Image references
			imageRefs := findPattern(xml, `r:embed="(rId\d+)"`)
			phTypes := findPattern(xml, `<p:ph[^>]*type="([^"]*)"`)
			if len(phTypes) > 0 {
				fmt.Printf("  Placeholders: %s\n", strings.Join(phTypes, ", "))
			}

			// Check for blip references (images in slide)
			blipRefs := findPattern(xml, `<a:blip[^>]*r:embed="([^"]*)"`)
			if len(blipRefs) > 0 {
				fmt.Printf("  Image blips: %s\n", strings.Join(blipRefs, ", "))
			}

			_ = imageRefs
		}

		// Relationships
		if rf, ok := files[relsPath]; ok {
			data, _ := readFile(rf)
			rels := parseRels(string(data))
			for _, rel := range rels {
				relType := path.Base(rel.relType)
				target := resolveRelTarget("ppt/slides/", rel.target)
				exists := "OK"
				if _, ok := files[target]; !ok {
					exists = "MISSING"
				}
				fmt.Printf("  Rel %s -> %s [%s] %s\n", rel.id, rel.target, relType, exists)
			}
		}
		fmt.Println()
	}

	// Media files
	if len(mediaFiles) > 0 {
		fmt.Println("--- Media ---")
		for _, name := range mediaFiles {
			f := files[name]
			fmt.Printf("  %s (%d bytes)\n", name, f.UncompressedSize64)
		}
		fmt.Println()
	}
}

// --- Text mode ---

func printText(files map[string]*zip.File) {
	slideCount := countSlides(files)
	for i := 1; i <= slideCount; i++ {
		slidePath := fmt.Sprintf("ppt/slides/slide%d.xml", i)
		if sf, ok := files[slidePath]; ok {
			data, _ := readFile(sf)
			texts := extractAllText(string(data))
			if len(texts) > 0 {
				fmt.Printf("Slide %d: %s\n", i, strings.Join(texts, " | "))
			}
		}
	}
}

// --- Slide detail mode ---

func printSlideDetail(files map[string]*zip.File, num int) {
	slidePath := fmt.Sprintf("ppt/slides/slide%d.xml", num)
	relsPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", num)

	if sf, ok := files[slidePath]; ok {
		data, _ := readFile(sf)
		fmt.Printf("=== Slide %d XML ===\n", num)
		fmt.Println(string(data))
	} else {
		fmt.Fprintf(os.Stderr, "Slide %d not found\n", num)
	}

	if rf, ok := files[relsPath]; ok {
		data, _ := readFile(rf)
		fmt.Printf("\n=== Slide %d Rels ===\n", num)
		fmt.Println(string(data))
	}
}

// --- Lint mode ---

func runLint(files map[string]*zip.File, pptxPath string) int {
	var errors []string
	var warnings []string

	slideCount := countSlides(files)
	if slideCount == 0 {
		errors = append(errors, "no slides found")
	}

	// Check required files
	required := []string{
		"[Content_Types].xml",
		"_rels/.rels",
		"ppt/presentation.xml",
		"ppt/_rels/presentation.xml.rels",
	}
	for _, req := range required {
		if _, ok := files[req]; !ok {
			errors = append(errors, fmt.Sprintf("missing required file: %s", req))
		}
	}

	// Check content types
	if ct, ok := files["[Content_Types].xml"]; ok {
		data, _ := readFile(ct)
		ctXML := string(data)

		// Check that slide content types are declared
		for i := 1; i <= slideCount; i++ {
			slidePart := fmt.Sprintf("/ppt/slides/slide%d.xml", i)
			if !strings.Contains(ctXML, slidePart) {
				errors = append(errors, fmt.Sprintf("content type not declared for slide %d", i))
			}
		}

		// Check image extension defaults
		mediaFiles := collectPrefix(files, "ppt/media/")
		for _, name := range mediaFiles {
			ext := strings.TrimPrefix(path.Ext(name), ".")
			if ext == "svg" {
				// SVG needs specific content type
				if !strings.Contains(ctXML, "image/svg+xml") && !strings.Contains(ctXML, `Extension="svg"`) {
					warnings = append(warnings, fmt.Sprintf("SVG media %s but no SVG content type declared", name))
				}
			} else if ext == "png" || ext == "jpeg" || ext == "jpg" || ext == "gif" {
				if !strings.Contains(ctXML, fmt.Sprintf(`Extension="%s"`, ext)) {
					warnings = append(warnings, fmt.Sprintf("media %s but no default content type for .%s", name, ext))
				}
			}
		}
	}

	// Check presentation.xml.rels
	if presRels, ok := files["ppt/_rels/presentation.xml.rels"]; ok {
		data, _ := readFile(presRels)
		rels := parseRels(string(data))

		// Check that all slide rels point to existing files
		for _, rel := range rels {
			target := resolveRelTarget("ppt/", rel.target)
			if _, ok := files[target]; !ok {
				errors = append(errors, fmt.Sprintf("presentation.xml.rels: %s -> %s (target missing)", rel.id, target))
			}
		}

		// Check slide count matches
		slideRels := 0
		for _, rel := range rels {
			if strings.Contains(rel.relType, "/slide") && !strings.Contains(rel.relType, "Master") && !strings.Contains(rel.relType, "Layout") {
				slideRels++
			}
		}
		if slideRels != slideCount {
			warnings = append(warnings, fmt.Sprintf("presentation rels declare %d slides but found %d slide XML files", slideRels, slideCount))
		}
	}

	// Per-slide validation
	for i := 1; i <= slideCount; i++ {
		slidePath := fmt.Sprintf("ppt/slides/slide%d.xml", i)
		relsPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", i)

		// Check rels file exists
		if _, ok := files[relsPath]; !ok {
			errors = append(errors, fmt.Sprintf("slide %d: missing rels file %s", i, relsPath))
			continue
		}

		// Read slide XML
		var slideXML string
		if sf, ok := files[slidePath]; ok {
			data, _ := readFile(sf)
			slideXML = string(data)
		}

		// Read rels
		rf := files[relsPath]
		data, _ := readFile(rf)
		rels := parseRels(string(data))

		// Check layout relationship exists
		hasLayout := false
		for _, rel := range rels {
			if strings.Contains(rel.relType, "/slideLayout") {
				hasLayout = true
				target := resolveRelTarget("ppt/slides/", rel.target)
				if _, ok := files[target]; !ok {
					errors = append(errors, fmt.Sprintf("slide %d: layout target %s missing", i, target))
				}
			}
		}
		if !hasLayout {
			errors = append(errors, fmt.Sprintf("slide %d: no slideLayout relationship", i))
		}

		// Check image references match rels
		blipRefs := findPattern(slideXML, `<a:blip[^>]*r:embed="([^"]*)"`)
		relMap := make(map[string]string)
		for _, rel := range rels {
			relMap[rel.id] = rel.target
		}
		for _, ref := range blipRefs {
			target, ok := relMap[ref]
			if !ok {
				errors = append(errors, fmt.Sprintf("slide %d: image ref %s has no matching relationship", i, ref))
				continue
			}
			mediaPath := resolveRelTarget("ppt/slides/", target)
			if _, ok := files[mediaPath]; !ok {
				errors = append(errors, fmt.Sprintf("slide %d: image ref %s -> %s (media file missing)", i, ref, mediaPath))
			}
		}

		// Check for HTML entities in text (common typographer bug)
		texts := extractAllText(slideXML)
		for _, t := range texts {
			if strings.Contains(t, "&ldquo;") || strings.Contains(t, "&rdquo;") ||
				strings.Contains(t, "&lsquo;") || strings.Contains(t, "&rsquo;") ||
				strings.Contains(t, "&hellip;") || strings.Contains(t, "&ndash;") ||
				strings.Contains(t, "&mdash;") {
				errors = append(errors, fmt.Sprintf("slide %d: text contains unresolved HTML entities: %q", i, t))
			}
		}

		// Check for zero-dimension shapes
		zeroDims := findPattern(slideXML, `<a:ext cx="0" cy="0"/>`)
		// Filter out the group shape transform which always has 0,0
		nonGroupZero := 0
		// Count occurrences of zero ext outside grpSpPr
		zeroExtRe := regexp.MustCompile(`<a:ext cx="0" cy="0"/>`)
		matches := zeroExtRe.FindAllStringIndex(slideXML, -1)
		grpSpPrRe := regexp.MustCompile(`<p:grpSpPr>[\s\S]*?</p:grpSpPr>`)
		grpRanges := grpSpPrRe.FindAllStringIndex(slideXML, -1)
		for _, m := range matches {
			inGroup := false
			for _, g := range grpRanges {
				if m[0] >= g[0] && m[1] <= g[1] {
					inGroup = true
					break
				}
			}
			if !inGroup {
				nonGroupZero++
			}
		}
		if nonGroupZero > 0 {
			warnings = append(warnings, fmt.Sprintf("slide %d: %d shape(s) with zero dimensions", i, nonGroupZero))
		}
		_ = zeroDims
	}

	// Check for orphaned media
	mediaFiles := collectPrefix(files, "ppt/media/")
	referencedMedia := make(map[string]bool)
	for i := 1; i <= slideCount; i++ {
		relsPath := fmt.Sprintf("ppt/slides/_rels/slide%d.xml.rels", i)
		if rf, ok := files[relsPath]; ok {
			data, _ := readFile(rf)
			rels := parseRels(string(data))
			for _, rel := range rels {
				if strings.Contains(rel.relType, "/image") {
					target := resolveRelTarget("ppt/slides/", rel.target)
					referencedMedia[target] = true
				}
			}
		}
	}
	// Also check slide master/layout rels for template media
	for name := range files {
		if (strings.Contains(name, "slideMasters/_rels/") || strings.Contains(name, "slideLayouts/_rels/")) && strings.HasSuffix(name, ".rels") {
			if f, ok := files[name]; ok {
				data, _ := readFile(f)
				rels := parseRels(string(data))
				base := strings.TrimSuffix(strings.TrimPrefix(name, "ppt/"), ".rels")
				base = "ppt/" + strings.TrimSuffix(base, "_rels/")
				for _, rel := range rels {
					if strings.Contains(rel.relType, "/image") {
						dir := path.Dir(name)
						dir = strings.TrimSuffix(dir, "/_rels")
						target := resolveRelTarget(dir+"/", rel.target)
						referencedMedia[target] = true
					}
				}
			}
		}
	}

	// Output
	fmt.Printf("=== PPTX Lint: %s ===\n", pptxPath)
	fmt.Printf("Slides: %d, Media: %d\n\n", slideCount, len(mediaFiles))

	if len(errors) > 0 {
		fmt.Println("ERRORS:")
		for _, e := range errors {
			fmt.Printf("  [ERROR] %s\n", e)
		}
		fmt.Println()
	}
	if len(warnings) > 0 {
		fmt.Println("WARNINGS:")
		for _, w := range warnings {
			fmt.Printf("  [WARN]  %s\n", w)
		}
		fmt.Println()
	}
	if len(errors) == 0 && len(warnings) == 0 {
		fmt.Println("All checks passed.")
	}

	if len(errors) > 0 {
		return 1
	}
	return 0
}

// --- Helpers ---

func countSlides(files map[string]*zip.File) int {
	count := 0
	for name := range files {
		if strings.HasPrefix(name, "ppt/slides/slide") && strings.HasSuffix(name, ".xml") {
			count++
		}
	}
	return count
}

func collectPrefix(files map[string]*zip.File, prefix string) []string {
	var result []string
	for name := range files {
		if strings.HasPrefix(name, prefix) {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

var textRe = regexp.MustCompile(`<a:t>([^<]*)</a:t>`)

func extractAllText(xml string) []string {
	matches := textRe.FindAllStringSubmatch(xml, -1)
	var texts []string
	for _, m := range matches {
		t := strings.TrimSpace(m[1])
		if t != "" {
			texts = append(texts, unescapeXML(t))
		}
	}
	return texts
}

func findPattern(xml, pattern string) []string {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(xml, -1)
	var result []string
	for _, m := range matches {
		if len(m) > 1 {
			result = append(result, m[1])
		}
	}
	return result
}

type rel struct {
	id      string
	relType string
	target  string
}

var relRe = regexp.MustCompile(`<Relationship[^>]*Id="([^"]*)"[^>]*Type="([^"]*)"[^>]*Target="([^"]*)"`)

func parseRels(xml string) []rel {
	matches := relRe.FindAllStringSubmatch(xml, -1)
	var rels []rel
	for _, m := range matches {
		rels = append(rels, rel{id: m[1], relType: m[2], target: m[3]})
	}
	return rels
}

func resolveRelTarget(basePath, target string) string {
	if strings.HasPrefix(target, "/") {
		return strings.TrimPrefix(target, "/")
	}
	// Resolve relative path
	base := basePath
	for strings.HasPrefix(target, "../") {
		target = strings.TrimPrefix(target, "../")
		base = path.Dir(strings.TrimSuffix(base, "/")) + "/"
	}
	return base + target
}

func unescapeXML(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&apos;", "'")
	return s
}
