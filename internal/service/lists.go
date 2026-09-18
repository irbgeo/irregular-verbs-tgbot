package service

import (
	"context"
	"sort"
	"strings"
)

const pageSize = 10

// OpenMyWords opens the «Мои слова» word list editor.
func (s *Service) OpenMyWords(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{
		Screen: string(ScreenMyWords),
		List: &ListState{
			Kind:  KindMyWords,
			Page:  0,
			Draft: map[string]string{},
		},
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
	return View{
		Screen: ScreenWordListLevels,
		Levels: Levels,
	}, nil
}

// ChooseLevel opens the word list for a level pool ("all" = every word).
func (s *Service) ChooseLevel(ctx context.Context, p ChooseLevelParams) (View, error) {
	if p.Level != "all" && !validLevel(p.Level) {
		return View{}, nil
	}
	u, err := s.load(ctx, p.UserID)
	if err != nil {
		return View{}, err
	}
	u.State = State{
		Screen: string(ScreenWordList),
		List: &ListState{
			Kind:  KindWordList,
			Level: p.Level,
			Page:  0,
			Draft: map[string]string{},
		},
	}
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// ListPage changes the page of the current list.
func (s *Service) ListPage(ctx context.Context, p ListPageParams) (View, error) {
	u, err := s.load(ctx, p.UserID)
	if err != nil {
		return View{}, err
	}
	if u.State.List == nil {
		return View{}, nil
	}
	u.State.List.Page = p.Page
	v := s.listView(u) // clamps and syncs ls.Page
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// ListToggle flips a word's draft target per the current list kind.
func (s *Service) ListToggle(ctx context.Context, p ListToggleParams) (View, error) {
	u, err := s.load(ctx, p.UserID)
	if err != nil {
		return View{}, err
	}
	ls := u.State.List
	if ls == nil {
		return View{}, nil
	}
	if _, ok := s.verb(p.Base); !ok {
		return s.listView(u), nil // unknown base: re-render, no change
	}

	eff := effectiveStatus(u, p.Base)
	stored := storedStatus(u, p.Base)

	var target string
	if ls.Kind == KindMyWords {
		target = nextMyWordsStatus(eff)
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
		delete(ls.Draft, p.Base)
	} else {
		ls.Draft[p.Base] = target
	}
	v := s.listView(u)
	// show the tapped word's full info (3 forms + translation) in the message
	// text; transient (not persisted), so any later navigation clears it.
	past, part, tr := s.itemForms(u, p.Base)
	v.List.Selected = &ListItem{
		Base:        p.Base,
		Past:        past,
		Participle:  part,
		Translation: tr,
	}
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
	if u.Words == nil {
		u.Words = map[string]WordProgress{}
	}
	for base, target := range ls.Draft {
		switch target {
		case StatusStudy:
			w := u.Words[base]
			w.Status = StatusStudy
			u.Words[base] = w
		case StatusSkipped:
			u.Words[base] = WordProgress{Status: StatusSkipped}
		case StatusLearned:
			u.Words[base] = WordProgress{
				Status: StatusLearned,
				Mode:   2,
				Box:    BoxMax,
			}
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
		return View{
			Screen: ScreenWordListLevels,
			Levels: Levels,
		}, nil
	}
	u.State = State{Screen: string(ScreenMainMenu)}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenMainMenu}, nil
}

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

// buildMyWordsView builds the «Мои слова» screen: a single list of the words
// the user is working on. Membership is by STORED status (study or learned),
// so a word drafted to skipped stays visible (with the ❌ icon) until commit.
func (s *Service) buildMyWordsView(u *User, page int) ListView {
	var bases []string
	for base, w := range u.Words {
		if w.Status == StatusStudy || w.Status == StatusLearned {
			bases = append(bases, base)
		}
	}
	sort.Strings(bases)

	start, end, pages, clamped := pageBounds(len(bases), page)
	items := make([]ListItem, 0, end-start)
	for _, b := range bases[start:end] {
		past, part, tr := s.itemForms(u, b)
		items = append(items, ListItem{
			Base:        b,
			Status:      effectiveStatus(u, b),
			Past:        past,
			Participle:  part,
			Translation: tr,
		})
	}
	return ListView{
		Kind:    KindMyWords,
		Page:    clamped,
		Pages:   pages,
		HasPrev: clamped > 0,
		HasNext: clamped < pages-1,
		Items:   items,
		Dirty:   draftDirty(u),
	}
}

// draftDirty reports whether the user has any pending draft changes.
func draftDirty(u *User) bool {
	return u.State.List != nil && len(u.State.List.Draft) > 0
}

// buildWordListView builds the «Список слов» screen page for the given level scope.
func (s *Service) buildWordListView(u *User, args buildWordListViewArgs) ListView {
	var bases []string
	if args.Level == "all" {
		bases = s.allBases
	} else {
		for _, v := range s.byLevel[args.Level] {
			bases = append(bases, v.Base)
		}
	}
	start, end, pages, clamped := pageBounds(len(bases), args.Page)
	items := make([]ListItem, 0, end-start)
	for _, b := range bases[start:end] {
		past, part, tr := s.itemForms(u, b)
		items = append(items, ListItem{
			Base:        b,
			Status:      effectiveStatus(u, b),
			Past:        past,
			Participle:  part,
			Translation: tr,
		})
	}
	return ListView{
		Kind:    KindWordList,
		Level:   args.Level,
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
	if ls.Kind == KindSearch {
		lv := s.buildSearchView(u, buildSearchViewArgs{
			Query: ls.Query,
			Page:  ls.Page,
		})
		ls.Page = lv.Page
		return View{
			Screen: ScreenSearch,
			List:   &lv,
		}
	}
	if ls.Kind == KindWordList {
		lv := s.buildWordListView(u, buildWordListViewArgs{
			Level: ls.Level,
			Page:  ls.Page,
		})
		ls.Page = lv.Page
		return View{
			Screen: ScreenWordList,
			List:   &lv,
		}
	}
	lv := s.buildMyWordsView(u, ls.Page)
	ls.Page = lv.Page
	return View{
		Screen: ScreenMyWords,
		List:   &lv,
	}
}

// nextMyWordsStatus is the next status in the «Мои слова» tap cycle:
// study → learned → skipped → study.
func nextMyWordsStatus(eff string) string {
	switch eff {
	case StatusStudy:
		return StatusLearned
	case StatusLearned:
		return StatusSkipped
	default: // skipped (or anything else) wraps back to study
		return StatusStudy
	}
}

// storedStatus is the persisted status ignoring the draft.
func storedStatus(u *User, base string) string {
	if w, ok := u.Words[base]; ok {
		return w.Status
	}
	return StatusNew
}
