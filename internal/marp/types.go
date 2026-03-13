package marp

// Presentation represents a parsed Marp document.
type Presentation struct {
	Meta   Meta
	Slides []Slide
}

// Meta holds the frontmatter directives.
type Meta struct {
	Theme           string `yaml:"theme"`
	Paginate        bool   `yaml:"paginate"`
	Header          string `yaml:"header"`
	Footer          string `yaml:"footer"`
	BackgroundColor string `yaml:"backgroundColor"`
	Color           string `yaml:"color"`
}

// Slide represents a single slide with its directives and raw markdown.
type Slide struct {
	Directives  SlideDirectives
	RawMarkdown string
}

// SlideDirectives are per-slide HTML comment directives.
type SlideDirectives struct {
	Class           string
	BackgroundImage string
	BackgroundColor string
	Color           string
	Header          string
	Footer          string
}
