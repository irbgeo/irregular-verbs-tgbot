package service

import "testing"

func TestLoadVerbsParsesAll(t *testing.T) {
	vs, err := LoadVerbs("../../data/verbs.json")
	if err != nil {
		t.Fatalf("LoadVerbs: %v", err)
	}
	if len(vs) != 170 {
		t.Fatalf("got %d verbs, want 170", len(vs))
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
