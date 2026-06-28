package service

import "sort"

const pageSize = 10

// effectiveStatus is the word's status with the pending draft applied.
func effectiveStatus(u *User, base string) string {
	if u.State.List != nil {
		if t, ok := u.State.List.Draft[base]; ok {
			return t
		}
	}
	if w, ok := u.Words[base]; ok {
		return w.Status
	}
	return StatusNew
}

// pageBounds returns the [start,end) slice bounds for the given page, the total
// page count (min 1), and the clamped page index.
func pageBounds(n, page int) (start, end, pages, clamped int) {
	pages = (n + pageSize - 1) / pageSize
	if pages == 0 {
		pages = 1
	}
	if page < 0 {
		page = 0
	}
	if page >= pages {
		page = pages - 1
	}
	start = page * pageSize
	end = start + pageSize
	if end > n {
		end = n
	}
	return start, end, pages, page
}

// buildMyWordsView builds the «Мои слова» screen for the active section.
func (s *Service) buildMyWordsView(u *User, section string, page int) ListView {
	seen := map[string]string{}  // base -> effectiveStatus
	var study, skipped []string
	add := func(base string) {
		if _, exists := seen[base]; exists {
			return
		}
		status := effectiveStatus(u, base)
		seen[base] = status
		switch status {
		case StatusStudy, StatusLearned:
			study = append(study, base)
		case StatusSkipped:
			skipped = append(skipped, base)
		}
	}
	for base := range u.Words {
		add(base)
	}
	if u.State.List != nil {
		for base := range u.State.List.Draft {
			add(base)
		}
	}
	sort.Strings(study)
	sort.Strings(skipped)

	bases := study
	if section == StatusSkipped {
		bases = skipped
	}
	start, end, pages, clamped := pageBounds(len(bases), page)
	items := make([]ListItem, 0, end-start)
	for _, b := range bases[start:end] {
		items = append(items, ListItem{Base: b, Status: seen[b]})
	}
	return ListView{
		Kind:         KindMyWords,
		Section:      section,
		StudyCount:   len(study),
		SkippedCount: len(skipped),
		Page:         clamped,
		Pages:        pages,
		HasPrev:      clamped > 0,
		HasNext:      clamped < pages-1,
		Items:        items,
	}
}

// buildWordListView builds the «Список слов» screen page.
func (s *Service) buildWordListView(u *User, page int) ListView {
	start, end, pages, clamped := pageBounds(len(s.allBases), page)
	items := make([]ListItem, 0, end-start)
	for _, b := range s.allBases[start:end] {
		items = append(items, ListItem{Base: b, Status: effectiveStatus(u, b)})
	}
	level := ""
	if len(items) > 0 {
		level = s.byBase[items[0].Base].Level
	}
	return ListView{
		Kind:    KindWordList,
		Level:   level,
		Page:    clamped,
		Pages:   pages,
		HasPrev: clamped > 0,
		HasNext: clamped < pages-1,
		Items:   items,
	}
}
