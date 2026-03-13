package marp

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse reads a Marp markdown document and returns a Presentation.
func Parse(input string) (*Presentation, error) {
	meta, body, err := splitFrontmatter(input)
	if err != nil {
		return nil, err
	}

	slides := splitSlides(body)

	pres := &Presentation{
		Meta:   meta,
		Slides: slides,
	}
	return pres, nil
}

// splitFrontmatter extracts YAML frontmatter delimited by "---" at the start.
func splitFrontmatter(input string) (Meta, string, error) {
	var meta Meta

	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "---") {
		return meta, input, nil
	}

	// Find the closing "---"
	rest := trimmed[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return meta, input, nil
	}

	yamlBlock := rest[:idx]
	body := rest[idx+4:] // skip "\n---"

	if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
		return meta, body, err
	}

	return meta, body, nil
}

// splitSlides splits the body into slides on "---" line separators.
func splitSlides(body string) []Slide {
	lines := strings.Split(body, "\n")
	var slides []Slide
	var current []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "---" {
			slide := buildSlide(strings.Join(current, "\n"))
			slides = append(slides, slide)
			current = nil
		} else {
			current = append(current, line)
		}
	}

	// Last slide
	if len(current) > 0 || len(slides) == 0 {
		slide := buildSlide(strings.Join(current, "\n"))
		slides = append(slides, slide)
	}

	return slides
}

func buildSlide(raw string) Slide {
	directives := extractDirectives(raw)
	cleaned := removeDirectiveComments(raw)
	return Slide{
		Directives:  directives,
		RawMarkdown: strings.TrimSpace(cleaned),
	}
}
