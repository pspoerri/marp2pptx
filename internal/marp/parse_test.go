package marp

import (
	"testing"
)

func TestParse_BasicPresentation(t *testing.T) {
	input := `---
marp: true
theme: gaia
paginate: true
---

# Title Slide

---

## Second Slide

Some content here.
`

	pres, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if pres.Meta.Theme != "gaia" {
		t.Errorf("expected theme 'gaia', got %q", pres.Meta.Theme)
	}
	if !pres.Meta.Paginate {
		t.Error("expected paginate to be true")
	}
	if len(pres.Slides) != 2 {
		t.Fatalf("expected 2 slides, got %d", len(pres.Slides))
	}
	if pres.Slides[0].RawMarkdown != "# Title Slide" {
		t.Errorf("slide 0 content: %q", pres.Slides[0].RawMarkdown)
	}
}

func TestParse_NoFrontmatter(t *testing.T) {
	input := `# Just a slide

---

## Another slide
`

	pres, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if pres.Meta.Theme != "" {
		t.Errorf("expected empty theme, got %q", pres.Meta.Theme)
	}
	if len(pres.Slides) != 2 {
		t.Fatalf("expected 2 slides, got %d", len(pres.Slides))
	}
}

func TestParse_Directives(t *testing.T) {
	input := `---
marp: true
---

<!-- _class: lead -->
<!-- _backgroundColor: #000 -->

# Dark Slide
`

	pres, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(pres.Slides) != 1 {
		t.Fatalf("expected 1 slide, got %d", len(pres.Slides))
	}

	d := pres.Slides[0].Directives
	if d.Class != "lead" {
		t.Errorf("expected class 'lead', got %q", d.Class)
	}
	if d.BackgroundColor != "#000" {
		t.Errorf("expected backgroundColor '#000', got %q", d.BackgroundColor)
	}
}
