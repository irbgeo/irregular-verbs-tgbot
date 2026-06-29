package service

import (
	"strings"
	"unicode"
)

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// normBase normalizes a base-form answer, accepting an optional "to " prefix.
func normBase(s string) string { return strings.TrimPrefix(norm(s), "to ") }

func isFormSep(r rune) bool { return r == '/' || r == ',' || unicode.IsSpace(r) }

func anyEqual(input string, options []string) bool {
	in := norm(input)
	for _, o := range options {
		if in == norm(o) {
			return true
		}
	}
	return false
}

// requiresAllVariants reports whether all variants of a form must be entered.
// Only the verb "be" past (was/were) qualifies — they are grammatically
// distinct; other multi-variant forms are spelling alternatives where any one
// is accepted.
func requiresAllVariants(forms []string) bool {
	if len(forms) != 2 {
		return false
	}
	a, b := norm(forms[0]), norm(forms[1])
	return (a == "was" && b == "were") || (a == "were" && b == "was")
}

// matchForm reports whether input correctly answers a form: was/were needs all
// variants, any other form accepts a single valid variant.
func matchForm(input string, forms []string) bool {
	if requiresAllVariants(forms) {
		return allFormsMatch(input, forms)
	}
	return anyEqual(input, forms)
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

// tokensOf splits input on any separator (space, "/", ",", newline) and
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
	for _, g := range groups {
		if len(g) == 0 {
			return false
		}
		// was/were consumes both tokens; any other form consumes one.
		if requiresAllVariants(g) {
			if i+len(g) > len(toks) || !sameFormSet(toks[i:i+len(g)], g) {
				return false
			}
			i += len(g)
		} else {
			if i >= len(toks) || !anyEqual(toks[i], g) {
				return false
			}
			i++
		}
	}
	return i == len(toks)
}

// correctText is the human "correct answer" line for feedback: all three
// forms and the translation, separated by " - ".
func (s *Service) correctText(v Verb, variant string) string {
	return v.Base + " - " +
		strings.Join(v.Past[variant], "/") + " - " +
		strings.Join(v.Participle[variant], "/") + " - " +
		strings.Join(v.Translations, ", ")
}
