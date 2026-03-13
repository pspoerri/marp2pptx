package mermaid

import (
	"fmt"
	"regexp"
	"strings"
)

var classDefPattern = regexp.MustCompile(`^class\s+(\w+)\s*\{?\s*$`)
var classMemberInline = regexp.MustCompile(`^(\w+)\s*:\s*(.+)$`)

// classRelPattern matches class relationship lines like: A <|-- B : label
var classRelPattern = regexp.MustCompile(
	`^(\w+)\s+` +
		`(<\|--|\.\.\|>|<\|\.\.|--\|>|\*--|--\*|o--|--o|-->|<--|\.\.>|<\.\.|--|\.\.)` +
		`\s+(\w+)(?:\s*:\s*(.*))?$`,
)

func parseClassDiagram(lines []string) (Graph, error) {
	cd := &ClassDiagram{}
	classMap := make(map[string]*ClassDef)

	ensureClass := func(name string) {
		if _, ok := classMap[name]; !ok {
			cd.Classes = append(cd.Classes, ClassDef{Name: name})
			classMap[name] = &cd.Classes[len(cd.Classes)-1]
		}
	}

	inClass := ""
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		lower := strings.ToLower(line)
		if lower == "classdiagram" {
			continue
		}

		// End of class body
		if line == "}" {
			inClass = ""
			continue
		}

		// Inside class body - parse member
		if inClass != "" {
			if c, ok := classMap[inClass]; ok {
				if m, ok := parseClassMember(line); ok {
					c.Members = append(c.Members, m)
				}
			}
			continue
		}

		// Class definition: "class ClassName {" or "class ClassName"
		if m := classDefPattern.FindStringSubmatch(line); m != nil {
			name := m[1]
			ensureClass(name)
			if strings.HasSuffix(line, "{") {
				inClass = name
			}
			continue
		}

		// Inline member: "ClassName : +member"
		if m := classMemberInline.FindStringSubmatch(line); m != nil {
			// Only treat as inline member if m[1] is a known class or doesn't match relation
			if classRelPattern.FindStringSubmatch(line) == nil {
				name := m[1]
				ensureClass(name)
				if c, ok := classMap[name]; ok {
					if mem, ok := parseClassMember(m[2]); ok {
						c.Members = append(c.Members, mem)
					}
				}
				continue
			}
		}

		// Relationship
		if rel, ok := parseClassRelationLine(line); ok {
			ensureClass(rel.From)
			ensureClass(rel.To)
			cd.Relations = append(cd.Relations, rel)
			continue
		}
	}

	if len(cd.Classes) == 0 {
		return Graph{}, fmt.Errorf("no classes found in class diagram")
	}

	return Graph{
		Type:  DiagramClass,
		Class: cd,
	}, nil
}

func parseClassMember(line string) (ClassMember, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return ClassMember{}, false
	}

	m := ClassMember{}

	// Check for visibility prefix
	if len(line) > 0 {
		switch line[0] {
		case '+', '-', '#', '~':
			m.Visibility = string(line[0])
			line = line[1:]
		}
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return ClassMember{}, false
	}

	// Check if it's a method (contains parentheses)
	if idx := strings.Index(line, "("); idx >= 0 {
		m.IsMethod = true
		m.Name = strings.TrimSpace(line[:idx])
		// Extract return type after closing paren
		if ci := strings.Index(line, ")"); ci >= 0 && ci+1 < len(line) {
			m.Type = strings.TrimSpace(line[ci+1:])
		}
	} else {
		// Field: could be "Type name" or just "name"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			m.Type = parts[0]
			m.Name = parts[1]
		} else {
			m.Name = parts[0]
		}
	}

	return m, m.Name != ""
}

func parseClassRelationLine(line string) (ClassRelation, bool) {
	m := classRelPattern.FindStringSubmatch(line)
	if m == nil {
		return ClassRelation{}, false
	}

	from := m[1]
	op := m[2]
	to := m[3]
	label := ""
	if len(m) > 4 {
		label = strings.TrimSpace(m[4])
	}

	rel := ClassRelation{From: from, To: to, Label: label}

	switch op {
	case "<|--":
		rel.FromMarker = MarkerTriangle
		rel.Dashed = false
	case "--|>":
		rel.ToMarker = MarkerTriangle
		rel.Dashed = false
	case "<|..":
		rel.FromMarker = MarkerTriangle
		rel.Dashed = true
	case "..|>":
		rel.ToMarker = MarkerTriangle
		rel.Dashed = true
	case "*--":
		rel.FromMarker = MarkerDiamond
		rel.Dashed = false
	case "--*":
		rel.ToMarker = MarkerDiamond
		rel.Dashed = false
	case "o--":
		rel.FromMarker = MarkerCircle
		rel.Dashed = false
	case "--o":
		rel.ToMarker = MarkerCircle
		rel.Dashed = false
	case "-->":
		rel.ToMarker = MarkerArrow
		rel.Dashed = false
	case "<--":
		rel.FromMarker = MarkerArrow
		rel.Dashed = false
	case "..>":
		rel.ToMarker = MarkerArrow
		rel.Dashed = true
	case "<..":
		rel.FromMarker = MarkerArrow
		rel.Dashed = true
	case "--":
		rel.Dashed = false
	case "..":
		rel.Dashed = true
	default:
		return ClassRelation{}, false
	}

	return rel, true
}
