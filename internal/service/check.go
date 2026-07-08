package service

import (
	"strings"
	"unicode"
)

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// normBase normalizes a base-form answer, accepting an optional "to " prefix.
func normBase(s string) string { return strings.TrimPrefix(norm(s), "to ") }

// isFormSep treats any non-letter rune as a separator between forms, so users
// may divide answers with spaces, "/", ",", "-", "|", "." or anything else.
func isFormSep(r rune) bool { return !unicode.IsLetter(r) }

func anyEqual(input string, options []string) bool {
	in := norm(input)
	for _, o := range options {
		if in == norm(o) {
			return true
		}
	}
	return false
}

// matchForm reports whether input correctly answers a form. For a multi-variant
// form (e.g. was/were, burnt/burned) any single variant OR all of them together
// is accepted.
func matchForm(input string, forms []string) bool {
	return anyEqual(input, forms) || allFormsMatch(input, forms)
}

// allFormsMatch reports whether input lists exactly the set of options
// (every form present, none extra), splitting on spaces, "/" and ",".
func allFormsMatch(input string, options []string) bool {
	if len(options) == 0 {
		return false
	}
	got := map[string]bool{}
	for _, tok := range strings.FieldsFunc(input, isFormSep) {
		if n := norm(tok); n != "" {
			got[n] = true
		}
	}
	want := map[string]bool{}
	for _, o := range options {
		want[norm(o)] = true
	}
	if len(got) != len(want) {
		return false
	}
	for w := range want {
		if !got[w] {
			return false
		}
	}
	return true
}

// tokensOf splits input on any non-letter separator and
// normalizes each token.
func tokensOf(s string) []string {
	raw := strings.FieldsFunc(s, isFormSep)
	out := make([]string, 0, len(raw))
	for _, t := range raw {
		out = append(out, norm(t))
	}
	return out
}

// sameFormSet reports whether got and want hold the same forms (as a multiset,
// order within the group does not matter).
func sameFormSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	counts := map[string]int{}
	for _, w := range want {
		counts[norm(w)]++
	}
	for _, g := range got {
		if counts[g] == 0 {
			return false
		}
		counts[g]--
	}
	return true
}

// checkAllFormsOrdered reports whether input lists all three forms in order —
// base, then past, then participle — with an optional leading "to". Separators
// are flexible; the order of the three groups matters, order within a
// multi-variant group does not.
func (s *Service) checkAllFormsOrdered(v Verb, input, variant string) bool {
	toks := tokensOf(input)
	i := 0
	if i < len(toks) && toks[i] == "to" {
		i++ // optional infinitive marker
	}
	groups := [][]string{{v.Base}, v.Past[variant], v.Participle[variant]}
	return matchGroupsOrdered(groups, toks[i:])
}

// matchGroupsOrdered reports whether toks answers the ordered groups. Each
// multi-variant group accepts either one variant (consumes one token) or all of
// them together (consumes len(group) tokens). It backtracks so an ambiguous
// token — e.g. a past variant that also spells the participle — is tried both
// ways.
func matchGroupsOrdered(groups [][]string, toks []string) bool {
	if len(groups) == 0 {
		return len(toks) == 0
	}
	g := groups[0]
	if len(g) == 0 {
		return false
	}
	// one variant
	if len(toks) >= 1 && anyEqual(toks[0], g) && matchGroupsOrdered(groups[1:], toks[1:]) {
		return true
	}
	// all variants listed together
	if len(g) > 1 && len(toks) >= len(g) && sameFormSet(toks[:len(g)], g) && matchGroupsOrdered(groups[1:], toks[len(g):]) {
		return true
	}
	return false
}

// correctText is the human "correct answer" block for feedback: the three
// forms on the first line (separated by " - "), the translation on the next.
func (s *Service) correctText(v Verb, variant string) string {
	return v.Base + " - " +
		strings.Join(v.Past[variant], "/") + " - " +
		strings.Join(v.Participle[variant], "/") + "\n" +
		strings.Join(v.Translations, ", ")
}
