package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/pascal/marp2pptx/internal/markdown"
	"github.com/pascal/marp2pptx/internal/marp"
	"github.com/pascal/marp2pptx/internal/pptx"
)

func main() {
	output := flag.String("o", "", "output .pptx file path")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: marp2pptx [flags] <input.md>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

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

	var slides []pptx.SlideContent
	for _, slide := range pres.Slides {
		blocks, err := markdown.Convert(slide.RawMarkdown)
		if err != nil {
			return fmt.Errorf("converting markdown: %w", err)
		}
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
