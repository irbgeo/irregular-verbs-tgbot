package service

import "strings"

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func anyEqual(input string, options []string) bool {
	in := norm(input)
	for _, o := range options {
		if in == norm(o) {
			return true
		}
	}
	return false
}

// effectiveVariant returns the variant, defaulting to "gb" if empty.
func effectiveVariant(variant string) string {
	if variant == "" {
		return "gb"
	}
	return variant
}

// checkAnswer reports whether input is correct for the given sub-question.
func (s *Service) checkAnswer(v Verb, step int, input, variant string) bool {
	vt := effectiveVariant(variant)
	switch step {
	case 0:
		return norm(input) == norm(v.Base)
	case 1:
		return anyEqual(input, v.Past[vt])
	case 2:
		return anyEqual(input, v.Participle[vt])
	default:
		return anyEqual(input, v.Translations)
	}
}

// correctText is the human "correct answer" line for feedback.
func (s *Service) correctText(v Verb, variant string) string {
	vt := effectiveVariant(variant)
	return v.Base + " — " +
		strings.Join(v.Past[vt], "/") + " — " +
		strings.Join(v.Participle[vt], "/") + " — " +
		strings.Join(v.Translations, ", ")
}
