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
