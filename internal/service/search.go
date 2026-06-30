package service

import (
	"context"
	"sort"
	"strings"
)

// searchVerbs returns the bases of verbs matching the query, sorted
// alphabetically. The query is tokenized (whitespace / "/" / ","); a verb
// matches if ANY token matches it (union, de-duplicated). A token matches when
// it exactly equals a form (base/past/participle, both gb and us variants) or
// is a substring of a translation. Matching is case- and space-insensitive.
func (s *Service) searchVerbs(query string) []string {
	tokens := tokensOf(query)
	if len(tokens) == 0 {
		return nil
	}
	var out []string
	for _, base := range s.allBases {
		if verbMatchesAny(s.byBase[base], tokens) {
			out = append(out, base)
		}
	}
	sort.Strings(out)
	return out
}

// verbMatchesAny reports whether the verb matches at least one token.
func verbMatchesAny(v Verb, tokens []string) bool {
	forms := formSet(v)
	for _, t := range tokens {
		if forms[t] {
			return true
		}
		for _, tr := range v.Translations {
			if strings.Contains(norm(tr), t) {
				return true
			}
		}
	}
	return false
}

// formSet is the set of normalized forms (base + past + participle, both
// variants) used for exact matching.
func formSet(v Verb) map[string]bool {
	set := map[string]bool{norm(v.Base): true}
	for _, variant := range []string{"gb", "us"} {
		for _, f := range v.Past[variant] {
			set[norm(f)] = true
		}
		for _, f := range v.Participle[variant] {
			set[norm(f)] = true
		}
	}
	return set
}

// buildSearchView builds the «Поиск» results list for the query and page.
func (s *Service) buildSearchView(u *User, query string, page int) ListView {
	bases := s.searchVerbs(query)
	start, end, pages, clamped := pageBounds(len(bases), page)
	items := make([]ListItem, 0, end-start)
	for _, b := range bases[start:end] {
		past, part, tr := s.itemForms(u, b)
		items = append(items, ListItem{Base: b, Status: effectiveStatus(u, b), Past: past, Participle: part, Translation: tr})
	}
	return ListView{
		Kind:    KindSearch,
		Page:    clamped,
		Pages:   pages,
		HasPrev: clamped > 0,
		HasNext: clamped < pages-1,
		Items:   items,
		Dirty:   draftDirty(u),
	}
}

// OpenSearch shows the search prompt (no results yet).
func (s *Service) OpenSearch(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{Screen: string(ScreenSearch)}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenSearch}, nil
}

// Search runs the query and shows the results list (draft/commit reused).
func (s *Service) Search(ctx context.Context, userID int64, query string) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{
		Screen: string(ScreenSearch),
		List:   &ListState{Kind: KindSearch, Query: query, Page: 0, Draft: map[string]string{}},
	}
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// OnText is the single text-input entry point: on the search screen the text is
// a search query; otherwise it is a quiz answer.
func (s *Service) OnText(ctx context.Context, userID int64, text string) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if u.State.Screen == string(ScreenSearch) {
		return s.Search(ctx, userID, text)
	}
	return s.Answer(ctx, userID, text)
}
