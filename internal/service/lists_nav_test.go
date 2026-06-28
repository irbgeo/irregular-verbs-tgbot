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

func TestOpenWordListInitsState(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	v, _ := svc.OpenWordList(ctx, 7)
	if v.Screen != ScreenWordList || v.List == nil || v.List.Kind != KindWordList {
		t.Fatalf("view = %+v", v)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List == nil || u.State.List.Kind != KindWordList {
		t.Fatalf("state = %+v", u.State.List)
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
