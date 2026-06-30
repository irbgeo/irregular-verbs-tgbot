package service

import (
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
