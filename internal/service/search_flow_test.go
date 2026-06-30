package service

import (
	"context"
	"testing"
)

func searchSvc(t *testing.T) (*Service, *fakeUserRepo) {
	t.Helper()
	repo := newFakeUserRepo()
	_ = repo.Save(context.Background(), &User{ID: 7, Settings: Settings{Variant: "gb"}})
	return New(repo, searchCatalog()), repo
}

func TestOpenSearchShowsPrompt(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	v, err := svc.OpenSearch(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenSearch || v.List != nil {
		t.Fatalf("open search view = %+v (want screen=search, nil list)", v)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(ScreenSearch) || u.State.List != nil {
		t.Fatalf("state = %+v", u.State)
	}
}

func TestSearchPopulatesResults(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	v, err := svc.Search(ctx, 7, "go run")
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenSearch || v.List == nil || v.List.Kind != KindSearch {
		t.Fatalf("search view = %+v", v)
	}
	if len(v.List.Items) != 2 || v.List.Items[0].Base != "go" || v.List.Items[1].Base != "run" {
		t.Fatalf("items = %+v", v.List.Items)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List == nil || u.State.List.Query != "go run" {
		t.Fatalf("state list = %+v", u.State.List)
	}
}

func TestSearchTapAddsToStudyOnCommit(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	_, _ = svc.Search(ctx, 7, "go")
	// tap "go" -> draft study; words unchanged until commit
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusStudy {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}
	if _, ok := u.Words["go"]; ok {
		t.Fatal("must not write words before commit")
	}
	// commit -> go becomes study
	if _, err := svc.CommitList(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Status != StatusStudy {
		t.Fatalf("after commit go = %+v", u.Words["go"])
	}
}

func TestSearchBackToMenu(t *testing.T) {
	ctx := context.Background()
	svc, _ := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	_, _ = svc.Search(ctx, 7, "go")
	v, _ := svc.ListBack(ctx, 7)
	if v.Screen != ScreenMainMenu {
		t.Fatalf("back from search = %s", v.Screen)
	}
}

func TestOnTextRoutesToSearch(t *testing.T) {
	ctx := context.Background()
	svc, _ := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	v, err := svc.OnText(ctx, 7, "go")
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenSearch || v.List == nil || len(v.List.Items) != 1 || v.List.Items[0].Base != "go" {
		t.Fatalf("OnText on search screen must search; got %+v", v)
	}
}

func TestOnTextOffSearchDelegatesToAnswer(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	// not on the search screen: OnText must behave like Answer (no panic, no search list)
	_ = repo.Save(ctx, &User{ID: 7, Settings: Settings{Variant: "gb"}, State: State{Screen: string(ScreenMainMenu)}})
	v, err := svc.OnText(ctx, 7, "whatever")
	if err != nil {
		t.Fatal(err)
	}
	if v.List != nil && v.List.Kind == KindSearch {
		t.Fatalf("off-search OnText must not produce a search list; got %+v", v)
	}
}
