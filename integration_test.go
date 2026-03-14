package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// requireBuild builds the binary and returns its path.
func requireBuild(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "marp2pptx")
	cmd := exec.Command("go", "build", "-o", binary, ".")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build failed: %v", err)
	}
	return binary
}

// runBinary executes the built binary with args and returns stdout+stderr and any error.
func runBinary(t *testing.T, binary string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// pptxSlideCount opens a .pptx file and counts slide XML entries.
func pptxSlideCount(t *testing.T, path string) int {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("opening pptx: %v", err)
	}
	defer r.Close()
	count := 0
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/slides/slide") && strings.HasSuffix(f.Name, ".xml") {
			count++
		}
	}
	return count
}

// pptxHasMedia checks if the .pptx contains any media files.
func pptxHasMedia(t *testing.T, path string) bool {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("opening pptx: %v", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/media/") {
			return true
		}
	}
	return false
}

// pptxMediaCount counts media files in the .pptx.
func pptxMediaCount(t *testing.T, path string) int {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("opening pptx: %v", err)
	}
	defer r.Close()
	count := 0
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "ppt/media/") {
			count++
		}
	}
	return count
}

func TestIntegration_BasicConversion(t *testing.T) {
	binary := requireBuild(t)
	outPath := filepath.Join(t.TempDir(), "output.pptx")

	output, err := runBinary(t, binary, "-o", outPath, "testdata/sample.md")
	if err != nil {
		t.Fatalf("conversion failed: %v\n%s", err, output)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	count := pptxSlideCount(t, outPath)
	if count != 5 {
		t.Errorf("expected 5 slides, got %d", count)
	}
}

func TestIntegration_DefaultOutputPath(t *testing.T) {
	binary := requireBuild(t)

	// Copy sample.md to a temp dir so the .pptx ends up there
	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "test.md")
	data, err := os.ReadFile("testdata/sample.md")
	if err != nil {
		t.Fatalf("reading sample: %v", err)
	}
	if err := os.WriteFile(input, data, 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}

	output, err := runBinary(t, binary, input)
	if err != nil {
		t.Fatalf("conversion failed: %v\n%s", err, output)
	}

	expected := filepath.Join(tmpDir, "test.pptx")
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected default output %s not created: %v", expected, err)
	}
}

func TestIntegration_ImageEmbed(t *testing.T) {
	binary := requireBuild(t)

	// Create a markdown file referencing the test image
	tmpDir := t.TempDir()
	md := `---
marp: true
---

# Image Test

![test](test.png)
`
	input := filepath.Join(tmpDir, "img.md")
	if err := os.WriteFile(input, []byte(md), 0644); err != nil {
		t.Fatal(err)
	}
	// Copy test.png next to the markdown
	imgData, err := os.ReadFile("testdata/test.png")
	if err != nil {
		t.Fatalf("reading test.png: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "test.png"), imgData, 0644); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(tmpDir, "img.pptx")
	output, runErr := runBinary(t, binary, "-o", outPath, input)
	if runErr != nil {
		t.Fatalf("conversion failed: %v\n%s", runErr, output)
	}

	if !pptxHasMedia(t, outPath) {
		t.Error("expected pptx to contain embedded media")
	}
}

func TestIntegration_MermaidDiagram(t *testing.T) {
	binary := requireBuild(t)
	outPath := filepath.Join(t.TempDir(), "mermaid.pptx")

	output, err := runBinary(t, binary, "-o", outPath, "testdata/mermaid.md")
	if err != nil {
		t.Fatalf("conversion failed: %v\n%s", err, output)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	// 17 slides in mermaid.md (flowcharts, sequences, class, state, journey, ER, edges, mixed)
	count := pptxSlideCount(t, outPath)
	if count != 17 {
		t.Errorf("expected 17 slides, got %d", count)
	}
}

func TestIntegration_MermaidNoExternalDeps(t *testing.T) {
	// Mermaid diagrams should work without any external tools
	binary := requireBuild(t)

	tmpDir := t.TempDir()
	md := "---\nmarp: true\n---\n\n```mermaid\ngraph LR\n    A --> B\n```\n"
	input := filepath.Join(tmpDir, "m.md")
	if err := os.WriteFile(input, []byte(md), 0644); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(tmpDir, "m.pptx")

	// Run with empty PATH to prove no external tools needed
	cmd := exec.Command(binary, "-o", outPath, input)
	cmd.Env = []string{"HOME=" + os.Getenv("HOME")}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		t.Fatalf("mermaid should work without external tools: %v\n%s", err, out.String())
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

func TestIntegration_NoMermaidNoError(t *testing.T) {
	// Files without mermaid should work even without mmdc
	binary := requireBuild(t)
	outPath := filepath.Join(t.TempDir(), "output.pptx")

	output, err := runBinary(t, binary, "-o", outPath, "testdata/sample.md")
	if err != nil {
		t.Fatalf("conversion failed: %v\n%s", err, output)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

func TestIntegration_Version(t *testing.T) {
	binary := requireBuild(t)
	output, err := runBinary(t, binary, "-version")
	if err != nil {
		t.Fatalf("version flag failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "dev") {
		t.Errorf("expected version output to contain 'dev', got %q", output)
	}
}

// --- PPTX inspection helpers ---

// pptxSlideXML extracts the XML content of a specific slide (1-based).
func pptxSlideXML(t *testing.T, path string, slideNum int) string {
	t.Helper()
	return pptxFileContent(t, path, fmt.Sprintf("ppt/slides/slide%d.xml", slideNum))
}

// pptxFileContent reads a file from the PPTX ZIP.
func pptxFileContent(t *testing.T, pptxPath, filename string) string {
	t.Helper()
	r, err := zip.OpenReader(pptxPath)
	if err != nil {
		t.Fatalf("opening pptx: %v", err)
	}
	defer r.Close()
	for _, f := range r.File {
		if f.Name == filename {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("opening %s: %v", filename, err)
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("reading %s: %v", filename, err)
			}
			return string(data)
		}
	}
	t.Fatalf("file %s not found in %s", filename, pptxPath)
	return ""
}

// pptxSlideTexts extracts all text from a slide's <a:t> elements.
func pptxSlideTexts(t *testing.T, path string, slideNum int) []string {
	t.Helper()
	xml := pptxSlideXML(t, path, slideNum)
	re := regexp.MustCompile(`<a:t>([^<]*)</a:t>`)
	matches := re.FindAllStringSubmatch(xml, -1)
	var texts []string
	for _, m := range matches {
		s := m[1]
		s = strings.ReplaceAll(s, "&amp;", "&")
		s = strings.ReplaceAll(s, "&lt;", "<")
		s = strings.ReplaceAll(s, "&gt;", ">")
		s = strings.ReplaceAll(s, "&quot;", "\"")
		s = strings.ReplaceAll(s, "&apos;", "'")
		if strings.TrimSpace(s) != "" {
			texts = append(texts, s)
		}
	}
	return texts
}

// pptxSlideImageDimensions extracts image shape dimensions (cx, cy in EMU).
func pptxSlideImageDimensions(t *testing.T, path string, slideNum int) (cx, cy int) {
	t.Helper()
	xml := pptxSlideXML(t, path, slideNum)
	// Find <p:pic> shapes and extract their extent
	picRe := regexp.MustCompile(`<p:pic>[\s\S]*?</p:pic>`)
	pics := picRe.FindAllString(xml, -1)
	if len(pics) == 0 {
		return 0, 0
	}
	extRe := regexp.MustCompile(`<a:ext cx="(\d+)" cy="(\d+)"/>`)
	if m := extRe.FindStringSubmatch(pics[0]); len(m) > 2 {
		fmt.Sscanf(m[1], "%d", &cx)
		fmt.Sscanf(m[2], "%d", &cy)
	}
	return
}

// --- Extensions integration tests ---

func TestIntegration_Extensions(t *testing.T) {
	binary := requireBuild(t)
	outPath := filepath.Join(t.TempDir(), "extensions.pptx")

	output, err := runBinary(t, binary, "-o", outPath, "testdata/extensions.md")
	if err != nil {
		t.Fatalf("conversion failed: %v\n%s", err, output)
	}

	count := pptxSlideCount(t, outPath)
	if count != 6 {
		t.Errorf("expected 6 slides, got %d", count)
	}
}

func TestIntegration_TypographerText(t *testing.T) {
	binary := requireBuild(t)
	outPath := filepath.Join(t.TempDir(), "extensions.pptx")

	output, err := runBinary(t, binary, "-o", outPath, "testdata/extensions.md")
	if err != nil {
		t.Fatalf("conversion failed: %v\n%s", err, output)
	}

	// Slide 5 is the Typographer slide
	texts := pptxSlideTexts(t, outPath, 5)
	allText := strings.Join(texts, "")

	// Must not contain HTML entities
	htmlEntities := []string{"&ldquo;", "&rdquo;", "&lsquo;", "&rsquo;", "&hellip;", "&ndash;", "&mdash;"}
	for _, ent := range htmlEntities {
		if strings.Contains(allText, ent) {
			t.Errorf("slide 5 contains unresolved HTML entity %s in text: %q", ent, allText)
		}
	}

	// Must contain proper Unicode characters
	if !strings.Contains(allText, "\u201c") { // left double quote
		t.Errorf("expected left double quote in text, got %q", allText)
	}
	if !strings.Contains(allText, "\u2026") { // ellipsis
		t.Errorf("expected ellipsis in text, got %q", allText)
	}
	if !strings.Contains(allText, "\u2013") { // en-dash
		t.Errorf("expected en-dash in text, got %q", allText)
	}
}

func TestIntegration_ImageVisible(t *testing.T) {
	binary := requireBuild(t)
	outPath := filepath.Join(t.TempDir(), "extensions.pptx")

	output, err := runBinary(t, binary, "-o", outPath, "testdata/extensions.md")
	if err != nil {
		t.Fatalf("conversion failed: %v\n%s", err, output)
	}

	if !pptxHasMedia(t, outPath) {
		t.Fatal("expected pptx to contain embedded media")
	}

	// Slide 4 is the Image slide - check that image has reasonable dimensions
	cx, cy := pptxSlideImageDimensions(t, outPath, 4)
	minSize := 100000 // ~0.1 inch in EMU — image must be visible
	if cx < minSize || cy < minSize {
		t.Errorf("image dimensions too small to be visible: cx=%d cy=%d (min=%d)", cx, cy, minSize)
	}
}

func TestIntegration_PptxLint(t *testing.T) {
	binary := requireBuild(t)

	tests := []struct {
		name  string
		input string
		args  []string // extra args before input
	}{
		{"sample", "testdata/sample.md", nil},
		{"extensions", "testdata/extensions.md", nil},
		{"mermaid", "testdata/mermaid.md", nil},
		{"darktheme", "testdata/sample.md", []string{"-theme", "testdata/darktheme.potx"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outPath := filepath.Join(t.TempDir(), tt.name+".pptx")
			args := append(tt.args, "-o", outPath, tt.input)
			output, err := runBinary(t, binary, args...)
			if err != nil {
				t.Fatalf("conversion failed: %v\n%s", err, output)
			}

			// Run lint tool
			cmd := exec.Command("go", "run", "./cmd/pptxeval", "-lint", outPath)
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			if err := cmd.Run(); err != nil {
				t.Errorf("lint failed for %s:\n%s", tt.name, out.String())
			}
		})
	}
}

func TestIntegration_DarkTheme(t *testing.T) {
	binary := requireBuild(t)
	outPath := filepath.Join(t.TempDir(), "dark.pptx")

	output, err := runBinary(t, binary, "-theme", "testdata/darktheme.potx", "-o", outPath, "testdata/sample.md")
	if err != nil {
		t.Fatalf("conversion failed: %v\n%s", err, output)
	}

	count := pptxSlideCount(t, outPath)
	if count != 5 {
		t.Errorf("expected 5 slides, got %d", count)
	}

	// Verify template theme is used (not built-in)
	themeXML := pptxFileContent(t, outPath, "ppt/theme/theme1.xml")
	if !strings.Contains(themeXML, "Calibri Light") {
		t.Error("expected template theme with Calibri Light font")
	}

	// Verify template media files are preserved
	mediaCount := pptxMediaCount(t, outPath)
	if mediaCount < 10 {
		t.Errorf("expected template media files to be preserved, got %d", mediaCount)
	}

	// Verify presentation.xml has defaultTextStyle from template
	presXML := pptxFileContent(t, outPath, "ppt/presentation.xml")
	if !strings.Contains(presXML, "<p:defaultTextStyle>") {
		t.Error("expected defaultTextStyle from template in presentation.xml")
	}
	if !strings.Contains(presXML, "<p:notesMasterIdLst>") {
		t.Error("expected notesMasterIdLst from template in presentation.xml")
	}
}
