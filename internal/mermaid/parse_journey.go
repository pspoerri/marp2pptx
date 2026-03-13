package mermaid

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// journeyTaskPattern matches: Task name: score: actor1, actor2
var journeyTaskPattern = regexp.MustCompile(
	`^(.+?):\s*(\d+)(?:\s*:\s*(.+))?$`,
)

func parseJourneyDiagram(lines []string) (Graph, error) {
	jd := &JourneyDiagram{}
	var currentSection *JourneySection

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "%%") {
			continue
		}
		lower := strings.ToLower(line)
		if lower == "journey" {
			continue
		}

		// Title
		if strings.HasPrefix(lower, "title ") {
			jd.Title = strings.TrimSpace(line[6:])
			continue
		}

		// Section
		if strings.HasPrefix(lower, "section ") {
			jd.Sections = append(jd.Sections, JourneySection{
				Name: strings.TrimSpace(line[8:]),
			})
			currentSection = &jd.Sections[len(jd.Sections)-1]
			continue
		}

		// Task
		if m := journeyTaskPattern.FindStringSubmatch(line); m != nil {
			name := strings.TrimSpace(m[1])
			score, _ := strconv.Atoi(m[2])
			if score < 1 {
				score = 1
			}
			if score > 5 {
				score = 5
			}
			var actors []string
			if m[3] != "" {
				for _, a := range strings.Split(m[3], ",") {
					actors = append(actors, strings.TrimSpace(a))
				}
			}
			task := JourneyTask{Name: name, Score: score, Actors: actors}
			if currentSection != nil {
				currentSection.Tasks = append(currentSection.Tasks, task)
			} else {
				// Task without a section — create a default section
				jd.Sections = append(jd.Sections, JourneySection{
					Name:  "",
					Tasks: []JourneyTask{task},
				})
				currentSection = &jd.Sections[len(jd.Sections)-1]
			}
			continue
		}
	}

	totalTasks := 0
	for _, s := range jd.Sections {
		totalTasks += len(s.Tasks)
	}
	if totalTasks == 0 {
		return Graph{}, fmt.Errorf("no tasks found in journey diagram")
	}

	return Graph{
		Type:    DiagramJourney,
		Journey: jd,
	}, nil
}
