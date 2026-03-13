package marp

import (
	"regexp"
	"strings"
)

var directiveRe = regexp.MustCompile(`<!--\s*(\w+)\s*:\s*(.*?)\s*-->`)

// extractDirectives parses HTML comment directives from slide markdown.
// Supports both scoped (_directive) and global (directive) forms.
func extractDirectives(raw string) SlideDirectives {
	var d SlideDirectives
	matches := directiveRe.FindAllStringSubmatch(raw, -1)
	for _, m := range matches {
		key := strings.TrimPrefix(m[1], "_")
		value := m[2]
		switch key {
		case "class":
			d.Class = value
		case "backgroundImage":
			d.BackgroundImage = value
		case "backgroundColor":
			d.BackgroundColor = value
		case "color":
			d.Color = value
		case "header":
			d.Header = value
		case "footer":
			d.Footer = value
		}
	}
	return d
}

// removeDirectiveComments strips directive HTML comments from markdown.
func removeDirectiveComments(raw string) string {
	return directiveRe.ReplaceAllString(raw, "")
}
