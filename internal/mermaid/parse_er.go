package mermaid

import (
	"fmt"
	"regexp"
	"strings"
)

var erEntityPattern = regexp.MustCompile(`^(\w[\w-]*)\s*\{\s*$`)

// erRelPattern matches: ENTITY1 ||--o{ ENTITY2 : label
var erRelPattern = regexp.MustCompile(
	`^(\w[\w-]*)\s+` +
		`(\|\||o\||\}\||\}o)` + // left cardinality
		`(--|\.\.)` + // line type
		`(\|\||\|o|\|\{|o\{)` + // right cardinality
		`\s+(\w[\w-]*)(?:\s*:\s*(.*))?$`,
)

func parseERDiagram(lines []string) (Graph, error) {
	er := &ERDiagram{}
	entityMap := make(map[string]*EREntity)
	inEntity := ""

	ensureEntity := func(name string) {
		if _, ok := entityMap[name]; !ok {
			er.Entities = append(er.Entities, EREntity{Name: name})
			entityMap[name] = &er.Entities[len(er.Entities)-1]
		}
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		lower := strings.ToLower(line)
		if lower == "erdiagram" {
			continue
		}

		// End of entity body
		if line == "}" {
			inEntity = ""
			continue
		}

		// Inside entity body - parse attribute
		if inEntity != "" {
			if e, ok := entityMap[inEntity]; ok {
				if attr, ok := parseERAttribute(line); ok {
					e.Attributes = append(e.Attributes, attr)
				}
			}
			continue
		}

		// Entity definition with body: "ENTITY {"
		if m := erEntityPattern.FindStringSubmatch(line); m != nil {
			ensureEntity(m[1])
			inEntity = m[1]
			continue
		}

		// Relationship
		if rel, ok := parseERRelationshipLine(line); ok {
			ensureEntity(rel.EntityA)
			ensureEntity(rel.EntityB)
			er.Relationships = append(er.Relationships, rel)
			continue
		}
	}

	if len(er.Entities) == 0 {
		return Graph{}, fmt.Errorf("no entities found in ER diagram")
	}

	return Graph{
		Type: DiagramER,
		ER:   er,
	}, nil
}

func parseERAttribute(line string) (ERAttribute, bool) {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return ERAttribute{}, false
	}
	attr := ERAttribute{
		Type: parts[0],
		Name: parts[1],
	}
	for _, p := range parts[2:] {
		upper := strings.ToUpper(p)
		if upper == "PK" || upper == "FK" || upper == "UK" {
			attr.Keys = append(attr.Keys, upper)
		}
	}
	return attr, true
}

func parseERRelationshipLine(line string) (ERRelationship, bool) {
	m := erRelPattern.FindStringSubmatch(line)
	if m == nil {
		return ERRelationship{}, false
	}

	return ERRelationship{
		EntityA:      m[1],
		CardinalityA: parseERCardinality(m[2]),
		Identifying:  m[3] == "--",
		CardinalityB: parseERCardinality(m[4]),
		EntityB:      m[5],
		Label:        strings.TrimSpace(m[6]),
	}, true
}

func parseERCardinality(s string) ERCardinality {
	switch s {
	case "||":
		return CardExactlyOne
	case "o|", "|o":
		return CardZeroOrOne
	case "}|", "|{":
		return CardOneOrMore
	case "}o", "o{":
		return CardZeroOrMore
	default:
		return CardExactlyOne
	}
}
