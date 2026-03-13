package main

import (
	"archive/zip"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
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

	// 7 slides in mermaid.md
	count := pptxSlideCount(t, outPath)
	if count != 7 {
		t.Errorf("expected 7 slides, got %d", count)
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
