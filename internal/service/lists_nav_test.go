package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	require.Equal(t, ScreenMyWords, v.Screen)
	require.NotNil(t, v.List)
	u, _ := repo.Get(ctx, 7)
	require.NotNil(t, u.State.List)
	require.Equal(t, KindMyWords, u.State.List.Kind)
	require.NotNil(t, u.State.List.Draft)
}

func TestOpenWordListShowsPicker(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	v, err := svc.OpenWordList(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, ScreenWordListLevels, v.Screen)
	require.Len(t, v.Levels, len(Levels))
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, string(ScreenWordListLevels), u.State.Screen)
	require.Nil(t, u.State.List)
}

func TestChooseLevelOpensPool(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	v, err := svc.ChooseLevel(ctx, 7, "elementary")
	require.NoError(t, err)
	require.Equal(t, ScreenWordList, v.Screen)
	require.NotNil(t, v.List)
	require.Equal(t, KindWordList, v.List.Kind)
	// elementary pool = be, go (2 words), alpha
	require.Len(t, v.List.Items, 2)
	require.Equal(t, "be", v.List.Items[0].Base)
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, "elementary", u.State.List.Level)
}

func TestChooseLevelAll(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	v, _ := svc.ChooseLevel(ctx, 7, "all")
	require.NotNil(t, v.List)
	require.Len(t, v.List.Items, 3) // be, go, build
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, "all", u.State.List.Level)
}

func TestListBackSteps(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	// word_list -> picker
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "elementary")
	v, _ := svc.ListBack(ctx, 7)
	require.Equal(t, ScreenWordListLevels, v.Screen)
	// picker -> menu
	v, _ = svc.ListBack(ctx, 7)
	require.Equal(t, ScreenMainMenu, v.Screen)
	// my_words -> menu
	_, _ = svc.OpenMyWords(ctx, 7)
	v, _ = svc.ListBack(ctx, 7)
	require.Equal(t, ScreenMainMenu, v.Screen)
	u, _ := repo.Get(ctx, 7)
	require.Nil(t, u.State.List, "back must clear list state")
}

func TestListPageClamps(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "all")
	v, _ := svc.ListPage(ctx, 7, 99) // only 1 page (3 words)
	require.Zero(t, v.List.Page, "want clamped 0")
	u, _ := repo.Get(ctx, 7)
	require.Zero(t, u.State.List.Page)
}

func TestListNavNoStateIgnored(t *testing.T) {
	ctx := context.Background()
	svc, _ := navSvc(t)
	// no OpenMyWords/OpenWordList first -> List is nil
	v, _ := svc.ListPage(ctx, 7, 1)
	require.Equal(t, ScreenNone, v.Screen, "expected empty view")
}
