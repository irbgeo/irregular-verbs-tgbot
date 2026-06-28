package service

import "testing"

func listUser() *User {
	return &User{
		Words: map[string]WordProgress{
			"go": {Status: StatusStudy},
			"be": {Status: StatusLearned},
			"do": {Status: StatusSkipped},
		},
		State: State{List: &ListState{Kind: KindMyWords, Section: StatusStudy, Draft: map[string]string{}}},
	}
}

func TestBuildMyWordsView(t *testing.T) {
	s := New(nil, testCatalog())
	u := listUser()

	study := s.buildMyWordsView(u, StatusStudy, 0)
	if study.StudyCount != 2 || study.SkippedCount != 1 {
		t.Fatalf("counts: study=%d skipped=%d", study.StudyCount, study.SkippedCount)
	}
	// study section = go (study) + be (learned), sorted alpha
	if len(study.Items) != 2 || study.Items[0].Base != "be" || study.Items[1].Base != "go" {
		t.Fatalf("study items = %+v", study.Items)
	}
	if study.Items[0].Status != StatusLearned {
		t.Errorf("be status = %q", study.Items[0].Status)
	}

	skip := s.buildMyWordsView(u, StatusSkipped, 0)
	if len(skip.Items) != 1 || skip.Items[0].Base != "do" {
		t.Fatalf("skip items = %+v", skip.Items)
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
