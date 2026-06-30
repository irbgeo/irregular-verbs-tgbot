package service

import (
	"context"
	"sort"
	"strings"
)

const pageSize = 10

// itemForms returns the past, participle forms (user's variant) and the
// translations for the word.
func (s *Service) itemForms(u *User, base string) (past, participle, translation string) {
	v, ok := s.verb(base)
	if !ok {
		return "", "", ""
	}
	variant := u.Settings.Variant
	return strings.Join(v.Past[variant], "/"),
		strings.Join(v.Participle[variant], "/"),
		strings.Join(v.Translations, ", ")
}

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
		past, part, tr := s.itemForms(u, b)
		items = append(items, ListItem{Base: b, Status: seen[b], Past: past, Participle: part, Translation: tr})
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
		Dirty:        draftDirty(u),
	}
}

// draftDirty reports whether the user has any pending draft changes.
func draftDirty(u *User) bool {
	return u.State.List != nil && len(u.State.List.Draft) > 0
}

// buildWordListView builds the «Список слов» screen page for the given level scope.
func (s *Service) buildWordListView(u *User, level string, page int) ListView {
	var bases []string
	if level == "all" {
		bases = s.allBases
	} else {
		for _, v := range s.byLevel[level] {
			bases = append(bases, v.Base)
		}
	}
	start, end, pages, clamped := pageBounds(len(bases), page)
	items := make([]ListItem, 0, end-start)
	for _, b := range bases[start:end] {
		past, part, tr := s.itemForms(u, b)
		items = append(items, ListItem{Base: b, Status: effectiveStatus(u, b), Past: past, Participle: part, Translation: tr})
	}
	return ListView{
		Kind:    KindWordList,
		Level:   level,
		Page:    clamped,
		Pages:   pages,
		HasPrev: clamped > 0,
		HasNext: clamped < pages-1,
		Items:   items,
		Dirty:   draftDirty(u),
	}
}

// listView builds the current list View from State.List and syncs the clamped page.
func (s *Service) listView(u *User) View {
	ls := u.State.List
	if ls.Kind == KindWordList {
		lv := s.buildWordListView(u, ls.Level, ls.Page)
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

// OpenWordList opens the level picker for «Список слов».
func (s *Service) OpenWordList(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{Screen: string(ScreenWordListLevels)}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenWordListLevels, Levels: Levels}, nil
}

// ChooseLevel opens the word list for a level pool ("all" = every word).
func (s *Service) ChooseLevel(ctx context.Context, userID int64, level string) (View, error) {
	if level != "all" && !validLevel(level) {
		return View{}, nil
	}
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{
		Screen: string(ScreenWordList),
		List:   &ListState{Kind: KindWordList, Level: level, Page: 0, Draft: map[string]string{}},
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

// storedStatus is the persisted status ignoring the draft.
func storedStatus(u *User, base string) string {
	if w, ok := u.Words[base]; ok {
		return w.Status
	}
	return StatusNew
}

// ListToggle flips a word's draft target per the current list kind.
func (s *Service) ListToggle(ctx context.Context, userID int64, base string) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	ls := u.State.List
	if ls == nil {
		return View{}, nil
	}
	if _, ok := s.verb(base); !ok {
		return s.listView(u), nil // unknown base: re-render, no change
	}

	eff := effectiveStatus(u, base)
	stored := storedStatus(u, base)

	var target string
	if ls.Kind == KindMyWords {
		if eff == StatusSkipped {
			target = StatusStudy
		} else { // study or learned
			target = StatusSkipped
		}
	} else { // word_list: toggle study membership
		if eff == StatusStudy {
			if stored == StatusSkipped {
				target = StatusSkipped
			} else {
				target = StatusNew
			}
		} else {
			target = StatusStudy
		}
	}

	if target == stored {
		delete(ls.Draft, base)
	} else {
		ls.Draft[base] = target
	}
	v := s.listView(u)
	// show the tapped word's full info (3 forms + translation) in the message
	// text; transient (not persisted), so any later navigation clears it.
	past, part, tr := s.itemForms(u, base)
	v.List.Selected = &ListItem{Base: base, Past: past, Participle: part, Translation: tr}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// CommitList applies the draft to words and stays on the list.
func (s *Service) CommitList(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	ls := u.State.List
	if ls == nil {
		return View{}, nil
	}
	for base, target := range ls.Draft {
		switch target {
		case StatusStudy:
			if u.Words == nil {
				u.Words = map[string]WordProgress{}
			}
			w := u.Words[base]
			w.Status = StatusStudy
			u.Words[base] = w
		case StatusSkipped:
			if u.Words == nil {
				u.Words = map[string]WordProgress{}
			}
			u.Words[base] = WordProgress{Status: StatusSkipped}
		case StatusNew:
			delete(u.Words, base)
		}
	}
	ls.Draft = map[string]string{}
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// CancelList discards the draft and stays on the list.
func (s *Service) CancelList(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if u.State.List == nil {
		return View{}, nil
	}
	u.State.List.Draft = map[string]string{}
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// ListBack discards the draft and steps back one screen.
func (s *Service) ListBack(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	// word_list -> level picker; everything else (picker, my_words) -> menu.
	if u.State.List != nil && u.State.List.Kind == KindWordList {
		u.State = State{Screen: string(ScreenWordListLevels)}
		if err := s.save(ctx, u); err != nil {
			return View{}, err
		}
		return View{Screen: ScreenWordListLevels, Levels: Levels}, nil
	}
	u.State = State{Screen: string(ScreenMainMenu)}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenMainMenu}, nil
}
