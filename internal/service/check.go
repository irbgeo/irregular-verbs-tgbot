package service

import "strings"

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// BaseLabel renders an infinitive for display with the "to " marker.
func BaseLabel(base string) string { return "to " + base }

// normBase normalizes a base-form answer, accepting an optional "to " prefix.
func normBase(s string) string { return strings.TrimPrefix(norm(s), "to ") }

func anyEqual(input string, options []string) bool {
	in := norm(input)
	for _, o := range options {
		if in == norm(o) {
			return true
		}
	}
	return false
}

// checkAnswer reports whether input is correct for the given sub-question.
func (s *Service) checkAnswer(v Verb, step int, input, variant string) bool {
	switch step {
	case 0:
		return normBase(input) == norm(v.Base)
	case 1:
		return anyEqual(input, v.Past[variant])
	default:
		return anyEqual(input, v.Participle[variant])
	}
}

// correctText is the human "correct answer" line for feedback.
func (s *Service) correctText(v Verb, variant string) string {
	return BaseLabel(v.Base) + " — " +
		strings.Join(v.Past[variant], "/") + " — " +
		strings.Join(v.Participle[variant], "/") + " — " +
		strings.Join(v.Translations, ", ")
}
