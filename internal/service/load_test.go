package service

import "testing"

func TestLoadVerbsParsesAll(t *testing.T) {
	vs, err := LoadVerbs("../../data/verbs.json")
	if err != nil {
		t.Fatalf("LoadVerbs: %v", err)
	}
	if len(vs) != 113 {
		t.Fatalf("got %d verbs, want 113", len(vs))
	}

	var be Verb
	for _, v := range vs {
		if v.Base == "be" {
			be = v
			break
		}
	}
	if be.Level != "elementary" {
		t.Errorf("be.Level = %q, want elementary", be.Level)
	}
	got := be.Past["gb"]
	if len(got) != 2 || got[0] != "was" || got[1] != "were" {
		t.Errorf("be.Past[gb] = %v, want [was were]", got)
	}
}

// TestVerbsDatasetInvariants guards the dataset against malformed entries: each
// verb base is unique, sits in a known level, and carries the forms, a
// translation, and distractors the bot needs to render quizzes.
func TestVerbsDatasetInvariants(t *testing.T) {
	vs, err := LoadVerbs("../../data/verbs.json")
	if err != nil {
		t.Fatalf("LoadVerbs: %v", err)
	}

	known := map[string]bool{}
	for _, l := range Levels {
		known[l] = true
	}

	seen := map[string]bool{}
	for _, v := range vs {
		if seen[v.Base] {
			t.Errorf("duplicate base %q", v.Base)
		}
		seen[v.Base] = true

		if !known[v.Level] {
			t.Errorf("%s: unknown level %q", v.Base, v.Level)
		}
		for _, variant := range []string{"gb", "us"} {
			if len(v.Past[variant]) == 0 {
				t.Errorf("%s: empty past[%s]", v.Base, variant)
			}
			if len(v.Participle[variant]) == 0 {
				t.Errorf("%s: empty participle[%s]", v.Base, variant)
			}
		}
		if len(v.Translations) == 0 {
			t.Errorf("%s: no translations", v.Base)
		}
		if len(v.CommonMistakes) < 2 {
			t.Errorf("%s: want >=2 common_mistakes, got %d", v.Base, len(v.CommonMistakes))
		}
	}
}

// TestCommonMistakesAreCleanDistractors guards that every verb's
// common_mistakes are usable choice distractors: at least 2 distinct,
// single latin words, none duplicated and none equal to a real form of the
// verb (a "mistake" that is actually a correct form is not a distractor).
func TestCommonMistakesAreCleanDistractors(t *testing.T) {
	vs, err := LoadVerbs("../../data/verbs.json")
	if err != nil {
		t.Fatalf("LoadVerbs: %v", err)
	}
	isWord := func(s string) bool {
		if s == "" {
			return false
		}
		for _, r := range s {
			if r < 'a' || r > 'z' {
				return false
			}
		}
		return true
	}
	for _, v := range vs {
		forms := map[string]bool{norm(v.Base): true}
		for _, variant := range []string{"gb", "us"} {
			for _, f := range v.Past[variant] {
				forms[norm(f)] = true
			}
			for _, f := range v.Participle[variant] {
				forms[norm(f)] = true
			}
		}
		seen := map[string]bool{}
		for _, m := range v.CommonMistakes {
			n := norm(m)
			switch {
			case !isWord(n):
				t.Errorf("%s: mistake %q must be a single latin word", v.Base, m)
			case forms[n]:
				t.Errorf("%s: mistake %q equals a real form", v.Base, m)
			case seen[n]:
				t.Errorf("%s: duplicate mistake %q", v.Base, m)
			}
			seen[n] = true
		}
		clean := 0
		for n := range seen {
			if isWord(n) && !forms[n] {
				clean++
			}
		}
		if clean < 2 {
			t.Errorf("%s: want >=2 clean distinct mistakes, got %d in %v", v.Base, clean, v.CommonMistakes)
		}
	}
}
