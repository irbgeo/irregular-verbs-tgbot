package service

import (
	"context"
	"sort"
)

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

// listView builds the current list View from State.List and syncs the clamped page.
func (s *Service) listView(u *User) View {
	ls := u.State.List
	if ls.Kind == KindWordList {
		lv := s.buildWordListView(u, ls.Page)
		ls.Page = lv.Page
		return View{Screen: ScreenWordList, List: &lv}
	}
	lv := s.buildMyWordsView(u, ls.Section, ls.Page)
	ls.Page = lv.Page
	return View{Screen: ScreenMyWords, List: &lv}
}

// OpenMyWords opens the «Мои слова» editor (Изучаю section).
func (s *Service) OpenMyWords(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{
		Screen: string(ScreenMyWords),
		List:   &ListState{Kind: KindMyWords, Section: StatusStudy, Page: 0, Draft: map[string]string{}},
	}
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// OpenWordList opens the «Список слов» editor.
func (s *Service) OpenWordList(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{
		Screen: string(ScreenWordList),
		List:   &ListState{Kind: KindWordList, Page: 0, Draft: map[string]string{}},
	}
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// ListSection switches the active «Мои слова» section.
func (s *Service) ListSection(ctx context.Context, userID int64, section string) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if u.State.List == nil || u.State.List.Kind != KindMyWords {
		return View{}, nil
	}
	if section != StatusStudy && section != StatusSkipped {
		return View{}, nil
	}
	u.State.List.Section = section
	u.State.List.Page = 0
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// ListPage changes the page of the current list.
func (s *Service) ListPage(ctx context.Context, userID int64, page int) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if u.State.List == nil {
		return View{}, nil
	}
	u.State.List.Page = page
	v := s.listView(u) // clamps and syncs ls.Page
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}
