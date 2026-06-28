package service

import (
	"context"
	"testing"
)

func navSvc(t *testing.T) (*Service, *fakeUserRepo) {
	t.Helper()
	repo := newFakeUserRepo()
	_ = repo.Save(context.Background(), &User{
		ID:       7,
		Settings: Settings{Variant: "gb"},
		Words: map[string]WordProgress{
			"go": {Status: StatusStudy}, "be": {Status: StatusLearned}, "do": {Status: StatusSkipped},
		},
	})
	return New(repo, testCatalog()), repo
}

func TestOpenMyWordsInitsState(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	v, err := svc.OpenMyWords(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenMyWords || v.List == nil || v.List.Section != StatusStudy {
		t.Fatalf("view = %+v", v)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List == nil || u.State.List.Kind != KindMyWords || u.State.List.Draft == nil {
		t.Fatalf("state = %+v", u.State.List)
	}
}

func TestOpenWordListShowsPicker(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	v, err := svc.OpenWordList(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenWordListLevels || len(v.Levels) != len(Levels) {
		t.Fatalf("view = %+v", v)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(ScreenWordListLevels) || u.State.List != nil {
		t.Fatalf("state = %+v", u.State)
	}
}

func TestChooseLevelOpensPool(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	v, err := svc.ChooseLevel(ctx, 7, "elementary")
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenWordList || v.List == nil || v.List.Kind != KindWordList {
		t.Fatalf("view = %+v", v)
	}
	// elementary pool = be, go (2 words), alpha
	if len(v.List.Items) != 2 || v.List.Items[0].Base != "be" {
		t.Fatalf("items = %+v", v.List.Items)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Level != "elementary" {
		t.Fatalf("level = %q", u.State.List.Level)
	}
}

func TestChooseLevelAll(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	v, _ := svc.ChooseLevel(ctx, 7, "all")
	if v.List == nil || len(v.List.Items) != 3 { // be, go, build
		t.Fatalf("all pool items = %+v", v.List)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Level != "all" {
		t.Fatalf("level = %q", u.State.List.Level)
	}
}

func TestListBackSteps(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	// word_list -> picker
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "elementary")
	v, _ := svc.ListBack(ctx, 7)
	if v.Screen != ScreenWordListLevels {
		t.Fatalf("back from list = %s", v.Screen)
	}
	// picker -> menu
	v, _ = svc.ListBack(ctx, 7)
	if v.Screen != ScreenMainMenu {
		t.Fatalf("back from picker = %s", v.Screen)
	}
	// my_words -> menu
	_, _ = svc.OpenMyWords(ctx, 7)
	v, _ = svc.ListBack(ctx, 7)
	if v.Screen != ScreenMainMenu {
		t.Fatalf("back from my_words = %s", v.Screen)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List != nil {
		t.Fatal("back must clear list state")
	}
}

func TestListSectionSwitches(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)
	v, err := svc.ListSection(ctx, 7, StatusSkipped)
	if err != nil {
		t.Fatal(err)
	}
	if v.List.Section != StatusSkipped || len(v.List.Items) != 1 || v.List.Items[0].Base != "do" {
		t.Fatalf("view = %+v", v.List)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Section != StatusSkipped || u.State.List.Page != 0 {
		t.Fatalf("state = %+v", u.State.List)
	}
}

func TestListPageClamps(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "all")
	v, _ := svc.ListPage(ctx, 7, 99) // only 1 page (3 words)
	if v.List.Page != 0 {
		t.Fatalf("page = %d, want clamped 0", v.List.Page)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Page != 0 {
		t.Fatalf("persisted page = %d", u.State.List.Page)
	}
}

func TestListNavNoStateIgnored(t *testing.T) {
	ctx := context.Background()
	svc, _ := navSvc(t)
	// no OpenMyWords/OpenWordList first -> List is nil
	if v, _ := svc.ListSection(ctx, 7, StatusSkipped); v.Screen != ScreenNone {
		t.Fatalf("expected empty view, got %+v", v)
	}
	if v, _ := svc.ListPage(ctx, 7, 1); v.Screen != ScreenNone {
		t.Fatalf("expected empty view, got %+v", v)
	}
}
