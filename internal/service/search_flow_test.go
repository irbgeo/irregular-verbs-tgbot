package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	require.Equal(t, ScreenSearch, v.Screen, "open search view = %+v (want screen=search, nil list)", v)
	require.Nil(t, v.List, "open search view = %+v (want screen=search, nil list)", v)
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, string(ScreenSearch), u.State.Screen, "state = %+v", u.State)
	require.Nil(t, u.State.List, "state = %+v", u.State)
}

func TestSearchPopulatesResults(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	v, err := svc.Search(ctx, 7, "go run")
	require.NoError(t, err)
	require.Equal(t, ScreenSearch, v.Screen, "search view = %+v", v)
	require.NotNil(t, v.List, "search view = %+v", v)
	require.Equal(t, KindSearch, v.List.Kind, "search view = %+v", v)
	require.Len(t, v.List.Items, 2, "items = %+v", v.List.Items)
	require.Equal(t, "go", v.List.Items[0].Base, "items = %+v", v.List.Items)
	require.Equal(t, "run", v.List.Items[1].Base, "items = %+v", v.List.Items)
	u, _ := repo.Get(ctx, 7)
	require.NotNil(t, u.State.List, "state list = %+v", u.State.List)
	require.Equal(t, "go run", u.State.List.Query, "state list = %+v", u.State.List)
}

func TestSearchTapAddsToStudyOnCommit(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	_, _ = svc.Search(ctx, 7, "go")
	// tap "go" -> draft study; words unchanged until commit
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, StatusStudy, u.State.List.Draft["go"], "draft = %+v", u.State.List.Draft)
	_, ok := u.Words["go"]
	require.False(t, ok, "must not write words before commit")
	// commit -> go becomes study
	_, err := svc.CommitList(ctx, 7)
	require.NoError(t, err)
	u, _ = repo.Get(ctx, 7)
	require.Equal(t, StatusStudy, u.Words["go"].Status, "after commit go = %+v", u.Words["go"])
}

func TestSearchBackToMenu(t *testing.T) {
	ctx := context.Background()
	svc, _ := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	_, _ = svc.Search(ctx, 7, "go")
	v, _ := svc.ListBack(ctx, 7)
	require.Equal(t, ScreenMainMenu, v.Screen, "back from search = %s", v.Screen)
}

func TestOnTextRoutesToSearch(t *testing.T) {
	ctx := context.Background()
	svc, _ := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	v, err := svc.OnText(ctx, 7, "go")
	require.NoError(t, err)
	require.Equal(t, ScreenSearch, v.Screen, "OnText on search screen must search; got %+v", v)
	require.NotNil(t, v.List, "OnText on search screen must search; got %+v", v)
	require.Len(t, v.List.Items, 1, "OnText on search screen must search; got %+v", v)
	require.Equal(t, "go", v.List.Items[0].Base, "OnText on search screen must search; got %+v", v)
}

func TestOnTextOffSearchDelegatesToAnswer(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	// not on the search screen: OnText must behave like Answer (no panic, no search list)
	_ = repo.Save(ctx, &User{ID: 7, Settings: Settings{Variant: "gb"}, State: State{Screen: string(ScreenMainMenu)}})
	v, err := svc.OnText(ctx, 7, "whatever")
	require.NoError(t, err)
	require.False(t, v.List != nil && v.List.Kind == KindSearch, "off-search OnText must not produce a search list; got %+v", v)
}
