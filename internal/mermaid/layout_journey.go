package mermaid

const (
	journeyTitleH   = emuPerInch * 2 / 5
	journeySectionH = emuPerInch / 3
	journeyTaskH    = emuPerInch / 3
	journeyTaskGap  = emuPerInch / 8
	journeyPadding  = emuPerInch / 8
)

func computeJourneyLayout(g Graph, maxW, maxH int) Layout {
	jd := g.Journey

	// Count total items for vertical sizing
	totalRows := 0
	if jd.Title != "" {
		totalRows++
	}
	for _, s := range jd.Sections {
		totalRows++ // section header
		totalRows += len(s.Tasks)
	}
	if totalRows == 0 {
		return Layout{}
	}

	// Compute vertical spacing
	availH := maxH
	rowH := journeyTaskH
	sectionHeaderH := journeySectionH
	titleH := journeyTitleH
	gap := journeyTaskGap

	neededH := 0
	if jd.Title != "" {
		neededH += titleH + gap
	}
	for _, s := range jd.Sections {
		neededH += sectionHeaderH + gap
		neededH += len(s.Tasks) * (rowH + gap)
	}

	scale := 1.0
	if neededH > availH {
		scale = float64(availH) / float64(neededH)
	}

	titleH = int(float64(titleH) * scale)
	sectionHeaderH = int(float64(sectionHeaderH) * scale)
	rowH = int(float64(rowH) * scale)
	gap = int(float64(gap) * scale)

	// Build layout
	jl := &JourneyLayout{Title: jd.Title}
	curY := 0
	taskAreaX := maxW / 4 // tasks start after label area
	taskAreaW := maxW - taskAreaX - journeyPadding

	if jd.Title != "" {
		jl.TitleX = 0
		jl.TitleY = curY
		jl.TitleW = maxW
		jl.TitleH = titleH
		curY += titleH + gap
	}

	for _, sec := range jd.Sections {
		sl := JourneySectionLayout{
			Name: sec.Name,
			X:    0,
			Y:    curY,
			W:    maxW,
			H:    sectionHeaderH,
		}
		curY += sectionHeaderH + gap

		for _, task := range sec.Tasks {
			barW := taskAreaW * task.Score / 5
			tl := JourneyTaskLayout{
				JourneyTask: task,
				X:           taskAreaX,
				Y:           curY,
				W:           taskAreaW,
				H:           rowH,
				BarW:        barW,
			}
			sl.Tasks = append(sl.Tasks, tl)
			curY += rowH + gap
		}

		jl.Sections = append(jl.Sections, sl)
	}

	return Layout{
		Type:    DiagramJourney,
		Journey: jl,
		W:       maxW,
		H:       maxH,
	}
}
