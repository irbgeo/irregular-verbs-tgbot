package service

import "testing"

func listUser() *User {
	return &User{
		Words: map[string]WordProgress{
			"go": {Status: StatusStudy},
			"be": {Status: StatusLearned},
			"do": {Status: StatusSkipped},
		},
		State: State{List: &ListState{Kind: KindMyWords, Draft: map[string]string{}}},
	}
}

func TestBuildMyWordsView(t *testing.T) {
	s := New(nil, testCatalog())
	u := listUser()

	v := s.buildMyWordsView(u, 0)
	// stored study + learned only, alpha: be (learned), go (study); "do" (skipped) hidden
	if len(v.Items) != 2 || v.Items[0].Base != "be" || v.Items[1].Base != "go" {
		t.Fatalf("items = %+v", v.Items)
	}
	if v.Items[0].Status != StatusLearned || v.Items[1].Status != StatusStudy {
		t.Fatalf("statuses = %+v", v.Items)
	}
}

func TestBuildWordListView(t *testing.T) {
	s := New(nil, testCatalog()) // be, go (elementary); build (pre-intermediate)
	u := &User{Words: map[string]WordProgress{"go": {Status: StatusStudy}}}

	// "all" level: all 3 words in catalog order
	v := s.buildWordListView(u, "all", 0)
	if v.Pages != 1 {
		t.Fatalf("pages = %d, want 1 (3 words)", v.Pages)
	}
	// order: elementary(be, go) then pre-intermediate(build)
	if len(v.Items) != 3 || v.Items[0].Base != "be" || v.Items[2].Base != "build" {
		t.Fatalf("items = %+v", v.Items)
	}
	if v.Items[1].Base != "go" || v.Items[1].Status != StatusStudy {
		t.Errorf("go item = %+v", v.Items[1])
	}

	// elementary level: only be, go
	el := s.buildWordListView(u, "elementary", 0)
	if len(el.Items) != 2 || el.Items[0].Base != "be" || el.Items[1].Base != "go" {
		t.Fatalf("elementary items = %+v", el.Items)
	}
	if el.Level != "elementary" {
		t.Errorf("level = %q", el.Level)
	}
}
