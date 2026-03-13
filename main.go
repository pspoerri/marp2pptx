package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pascal/marp2pptx/internal/markdown"
	"github.com/pascal/marp2pptx/internal/marp"
	"github.com/pascal/marp2pptx/internal/pptx"
)

var version = "dev"

func main() {
	output := flag.String("o", "", "output .pptx file path")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: marp2pptx [flags] <input.md>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	inputPath := flag.Arg(0)
	outputPath := *output
	if outputPath == "" {
		// Default: replace .md with .pptx
		outputPath = inputPath
		if len(outputPath) > 3 && outputPath[len(outputPath)-3:] == ".md" {
			outputPath = outputPath[:len(outputPath)-3] + ".pptx"
		} else {
			outputPath += ".pptx"
		}
	}

	if err := run(inputPath, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created %s\n", outputPath)
}

func run(inputPath, outputPath string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	pres, err := marp.Parse(string(data))
	if err != nil {
		return fmt.Errorf("parsing marp: %w", err)
	}

	baseDir := filepath.Dir(inputPath)

	var slides []pptx.SlideContent
	for _, slide := range pres.Slides {
		blocks, err := markdown.Convert(slide.RawMarkdown)
		if err != nil {
			return fmt.Errorf("converting markdown: %w", err)
		}
		resolveImages(blocks, baseDir)
		slides = append(slides, pptx.SlideContent{
			Blocks:     blocks,
			Directives: slide.Directives,
		})
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output: %w", err)
	}
	defer f.Close()

	if err := pptx.Write(f, pres.Meta, slides); err != nil {
		return fmt.Errorf("writing pptx: %w", err)
	}

	return nil
}

// resolveImages reads local image files and attaches data to Image blocks.
func resolveImages(blocks []markdown.ContentBlock, baseDir string) {
	for i, block := range blocks {
		img, ok := block.(markdown.Image)
		if !ok || img.URL == "" {
			continue
		}
		// Skip remote URLs
		if strings.HasPrefix(img.URL, "http://") || strings.HasPrefix(img.URL, "https://") {
			continue
		}
		imgPath := filepath.Join(baseDir, img.URL)
		imgData, err := os.ReadFile(imgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read image %s: %v\n", img.URL, err)
			continue
		}
		img.Data = imgData
		blocks[i] = img
	}
}
