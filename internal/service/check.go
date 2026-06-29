package service

import (
	"strings"
	"unicode"
)

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// BaseLabel renders an infinitive for display with the "to " marker.
func BaseLabel(base string) string { return "to " + base }

// normBase normalizes a base-form answer, accepting an optional "to " prefix.
func normBase(s string) string { return strings.TrimPrefix(norm(s), "to ") }

func isFormSep(r rune) bool { return r == '/' || r == ',' || unicode.IsSpace(r) }

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

// correctText is the human "correct answer" line for feedback.
func (s *Service) correctText(v Verb, variant string) string {
	return BaseLabel(v.Base) + " — " +
		strings.Join(v.Past[variant], "/") + " — " +
		strings.Join(v.Participle[variant], "/")
}
