package service

import (
	"context"
	"testing"
)

func TestToggleMyWordsCycles(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)

	// go is study. tap -> learned (draft only, words unchanged)
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusLearned {
		t.Fatalf("after 1 tap draft = %+v, want learned", u.State.List.Draft)
	}
	if u.Words["go"].Status != StatusStudy {
		t.Fatal("words must not change before commit")
	}
	// tap -> skipped
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ = repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusSkipped {
		t.Fatalf("after 2 taps draft = %+v, want skipped", u.State.List.Draft)
	}
	// tap -> back to stored study -> draft entry removed
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ = repo.Get(ctx, 7)
	if _, ok := u.State.List.Draft["go"]; ok {
		t.Fatalf("after 3 taps draft should be cleared, got %+v", u.State.List.Draft)
	}
}

func TestToggleWordListStudy(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "all")

	// be is learned (effective != study) -> tap sets study
	_, _ = svc.ListToggle(ctx, 7, "be")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["be"] != StatusStudy {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}
	// "build" is new -> tap -> study; tap again -> new -> draft cleared
	_, _ = svc.ListToggle(ctx, 7, "build")
	u, _ = repo.Get(ctx, 7)
	if u.State.List.Draft["build"] != StatusStudy {
		t.Fatalf("build draft = %+v", u.State.List.Draft)
	}
	_, _ = svc.ListToggle(ctx, 7, "build")
	u, _ = repo.Get(ctx, 7)
	if _, ok := u.State.List.Draft["build"]; ok {
		t.Fatalf("build draft should clear, got %+v", u.State.List.Draft)
	}
	// go is study -> tap study; tap again -> back to study (stored), draft cleared
	_, _ = svc.ListToggle(ctx, 7, "go")
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ = repo.Get(ctx, 7)
	if _, ok := u.State.List.Draft["go"]; ok {
		t.Fatalf("go draft should clear, got %+v", u.State.List.Draft)
	}
}

func TestToggleSetsSelectedInfoAndNavClears(t *testing.T) {
	ctx := context.Background()
	svc, _ := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)

	v, err := svc.ListToggle(ctx, 7, "go")
	if err != nil {
		t.Fatal(err)
	}
	if v.List == nil || v.List.Selected == nil {
		t.Fatalf("toggle must set Selected, got %+v", v.List)
	}
	s := v.List.Selected
	if s.Base != "go" || s.Past != "went" || s.Participle != "gone" || s.Translation != "идти" {
		t.Fatalf("selected = %+v", s)
	}

	// navigation must clear the info block
	v2, err := svc.ListPage(ctx, 7, 0)
	if err != nil {
		t.Fatal(err)
	}
	if v2.List.Selected != nil {
		t.Fatalf("page nav must clear Selected, got %+v", v2.List.Selected)
	}
}

func TestToggleUnknownBaseNoSelected(t *testing.T) {
	ctx := context.Background()
	svc, _ := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)
	v, err := svc.ListToggle(ctx, 7, "nope")
	if err != nil {
		t.Fatal(err)
	}
	if v.List != nil && v.List.Selected != nil {
		t.Fatalf("unknown base must not set Selected, got %+v", v.List.Selected)
	}
}

func TestCommitAppliesDraft(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "all")
	_, _ = svc.ListToggle(ctx, 7, "build") // new -> study
	_, _ = svc.ListToggle(ctx, 7, "go")    // study -> new

	v, err := svc.CommitList(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenWordList { // stays on the list, not main_menu
		t.Fatalf("screen = %s, want word_list", v.Screen)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List == nil || len(u.State.List.Draft) != 0 {
		t.Fatalf("after commit: list=%+v (draft must be cleared, list kept)", u.State.List)
	}
	if u.Words["build"].Status != StatusStudy {
		t.Fatalf("words = %+v", u.Words)
	}
	if _, ok := u.Words["go"]; ok {
		t.Fatalf("go should be deleted, words = %+v", u.Words)
	}
	if u.Words["build"].Box != 0 || u.Words["build"].Mode != 0 {
		t.Fatalf("build progress = %+v", u.Words["build"])
	}
}

func TestCommitNewDeletes(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "all")
	_, _ = svc.ListToggle(ctx, 7, "go") // study -> new (toggle off)
	_, _ = svc.CommitList(ctx, 7)
	u, _ := repo.Get(ctx, 7)
	if _, ok := u.Words["go"]; ok {
		t.Fatalf("go should be deleted, got %+v", u.Words["go"])
	}
}

func TestCancelDiscards(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)
	_, _ = svc.ListToggle(ctx, 7, "go")
	v, _ := svc.CancelList(ctx, 7)
	if v.Screen != ScreenMyWords {
		t.Fatalf("screen = %s, want my_words", v.Screen)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List == nil || len(u.State.List.Draft) != 0 {
		t.Fatalf("cancel should clear draft but keep list; got %+v", u.State.List)
	}
	if u.Words["go"].Status != StatusStudy {
		t.Fatalf("cancel must not change words; go=%+v", u.Words["go"])
	}
}

func TestToggleWordListSkippedRoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	_ = repo.Save(ctx, &User{
		ID:       7,
		Settings: Settings{Variant: "gb"},
		Words: map[string]WordProgress{
			"build": {Status: StatusSkipped},
		},
	})
	svc := New(repo, testCatalog())

	// open word list (catalog view)
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "all")

	// build is skipped -> tap -> study (draft)
	_, _ = svc.ListToggle(ctx, 7, "build")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["build"] != StatusStudy {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}

	// tap again -> back to stored skipped, draft entry removed
	_, _ = svc.ListToggle(ctx, 7, "build")
	u, _ = repo.Get(ctx, 7)
	if _, ok := u.State.List.Draft["build"]; ok {
		t.Fatalf("build draft should be cleared, got %+v", u.State.List.Draft)
	}
}

func TestCommitSkippedWritesSkipped(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	_ = repo.Save(ctx, &User{
		ID:       7,
		Settings: Settings{Variant: "gb"},
		Words: map[string]WordProgress{
			"go": {Status: StatusStudy},
		},
	})
	svc := New(repo, testCatalog())

	// open "Мои слова" (study section)
	_, _ = svc.OpenMyWords(ctx, 7)

	// go is study -> tap (learned) -> tap (skipped) in the draft
	_, _ = svc.ListToggle(ctx, 7, "go")
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusSkipped {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}

	// commit: apply skipped to words, stay on list
	_, err := svc.CommitList(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Status != StatusSkipped {
		t.Fatalf("words = %+v", u.Words)
	}
	if u.State.List == nil || len(u.State.List.Draft) != 0 {
		t.Fatalf("after commit: list=%+v (draft must be cleared, list kept)", u.State.List)
	}
}

func TestCommitLearnedWritesLearned(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	_ = repo.Save(ctx, &User{
		ID:       7,
		Settings: Settings{Variant: "gb"},
		Words:    map[string]WordProgress{"go": {Status: StatusStudy}},
	})
	svc := New(repo, testCatalog())

	_, _ = svc.OpenMyWords(ctx, 7)
	_, _ = svc.ListToggle(ctx, 7, "go") // study -> learned (draft)
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusLearned {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}

	if _, err := svc.CommitList(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ = repo.Get(ctx, 7)
	got := u.Words["go"]
	if got.Status != StatusLearned || got.Mode != 2 || got.Box != BoxMax {
		t.Fatalf("go = %+v, want {learned, mode 2, box 5}", got)
	}
}
