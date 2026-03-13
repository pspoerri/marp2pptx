package mermaid

import (
	"fmt"
	"regexp"
	"strings"
)

// stateTransPattern matches: State1 --> State2 or State1 --> State2 : label
var stateTransPattern = regexp.MustCompile(
	`^(\[\*\]|\w+)\s*-->\s*(\[\*\]|\w+)(?:\s*:\s*(.*))?$`,
)

// stateAliasPattern matches: state "Long name" as s1
var stateAliasPattern = regexp.MustCompile(
	`^state\s+"([^"]+)"\s+as\s+(\w+)$`,
)

func parseStateDiagram(lines []string) (Graph, error) {
	sd := &StateDiagram{}
	stateMap := make(map[string]*StateDef)
	starCounter := 0

	ensureState := func(id string) string {
		if id == "[*]" {
			starCounter++
			starID := fmt.Sprintf("__star_%d__", starCounter)
			sd.States = append(sd.States, StateDef{ID: starID, Label: "", Type: StateStar})
			stateMap[starID] = &sd.States[len(sd.States)-1]
			return starID
		}
		if _, ok := stateMap[id]; !ok {
			sd.States = append(sd.States, StateDef{ID: id, Label: id, Type: StateNormal})
			stateMap[id] = &sd.States[len(sd.States)-1]
		}
		return id
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		lower := strings.ToLower(line)
		if lower == "statediagram" || lower == "statediagram-v2" {
			continue
		}

		// Skip directives
		if isStateDirective(lower) {
			continue
		}

		// State alias: state "Label" as id
		if m := stateAliasPattern.FindStringSubmatch(line); m != nil {
			label := m[1]
			id := m[2]
			if _, ok := stateMap[id]; !ok {
				sd.States = append(sd.States, StateDef{ID: id, Label: label, Type: StateNormal})
				stateMap[id] = &sd.States[len(sd.States)-1]
			} else {
				stateMap[id].Label = label
			}
			continue
		}

		// Transition: State1 --> State2 : label
		if m := stateTransPattern.FindStringSubmatch(line); m != nil {
			fromRaw := m[1]
			toRaw := m[2]
			label := ""
			if len(m) > 3 {
				label = strings.TrimSpace(m[3])
			}
			fromID := ensureState(fromRaw)
			toID := ensureState(toRaw)
			sd.Transitions = append(sd.Transitions, StateTransition{
				From:  fromID,
				To:    toID,
				Label: label,
			})
			continue
		}
	}

	if len(sd.States) == 0 {
		return Graph{}, fmt.Errorf("no states found in state diagram")
	}

	return Graph{
		Type:  DiagramState,
		State: sd,
	}, nil
}

func isStateDirective(lower string) bool {
	for _, kw := range []string{
		"direction ", "note ", "state ", "hide empty description",
	} {
		if strings.HasPrefix(lower, kw) {
			return true
		}
	}
	return false
}
