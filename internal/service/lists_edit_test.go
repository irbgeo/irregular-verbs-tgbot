package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToggleMyWordsCycles(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)

	// go is study. tap -> learned (draft only, words unchanged)
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, StatusLearned, u.State.List.Draft["go"])
	require.Equal(t, StatusStudy, u.Words["go"].Status, "words must not change before commit")
	// tap -> skipped
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ = repo.Get(ctx, 7)
	require.Equal(t, StatusSkipped, u.State.List.Draft["go"])
	// tap -> back to stored study -> draft entry removed
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ = repo.Get(ctx, 7)
	require.NotContains(t, u.State.List.Draft, "go", "after 3 taps draft should be cleared")
}

func TestToggleWordListStudy(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "all")

	// be is learned (effective != study) -> tap sets study
	_, _ = svc.ListToggle(ctx, 7, "be")
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, StatusStudy, u.State.List.Draft["be"])
	// "build" is new -> tap -> study; tap again -> new -> draft cleared
	_, _ = svc.ListToggle(ctx, 7, "build")
	u, _ = repo.Get(ctx, 7)
	require.Equal(t, StatusStudy, u.State.List.Draft["build"])
	_, _ = svc.ListToggle(ctx, 7, "build")
	u, _ = repo.Get(ctx, 7)
	require.NotContains(t, u.State.List.Draft, "build", "build draft should clear")
	// go is study -> tap study; tap again -> back to study (stored), draft cleared
	_, _ = svc.ListToggle(ctx, 7, "go")
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ = repo.Get(ctx, 7)
	require.NotContains(t, u.State.List.Draft, "go", "go draft should clear")
}

func TestToggleSetsSelectedInfoAndNavClears(t *testing.T) {
	ctx := context.Background()
	svc, _ := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)

	v, err := svc.ListToggle(ctx, 7, "go")
	require.NoError(t, err)
	require.NotNil(t, v.List)
	require.NotNil(t, v.List.Selected, "toggle must set Selected")
	s := v.List.Selected
	require.Equal(t, "go", s.Base)
	require.Equal(t, "went", s.Past)
	require.Equal(t, "gone", s.Participle)
	require.Equal(t, "идти", s.Translation)

	// navigation must clear the info block
	v2, err := svc.ListPage(ctx, 7, 0)
	require.NoError(t, err)
	require.Nil(t, v2.List.Selected, "page nav must clear Selected")
}

func TestToggleUnknownBaseNoSelected(t *testing.T) {
	ctx := context.Background()
	svc, _ := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)
	v, err := svc.ListToggle(ctx, 7, "nope")
	require.NoError(t, err)
	if v.List != nil {
		require.Nil(t, v.List.Selected, "unknown base must not set Selected")
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
	require.NoError(t, err)
	require.Equal(t, ScreenWordList, v.Screen, "stays on the list, not main_menu")
	u, _ := repo.Get(ctx, 7)
	require.NotNil(t, u.State.List)
	require.Empty(t, u.State.List.Draft, "draft must be cleared, list kept")
	require.Equal(t, StatusStudy, u.Words["build"].Status)
	require.NotContains(t, u.Words, "go", "go should be deleted")
	require.Zero(t, u.Words["build"].Box)
	require.Zero(t, u.Words["build"].Mode)
}

func TestCommitNewDeletes(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "all")
	_, _ = svc.ListToggle(ctx, 7, "go") // study -> new (toggle off)
	_, _ = svc.CommitList(ctx, 7)
	u, _ := repo.Get(ctx, 7)
	require.NotContains(t, u.Words, "go", "go should be deleted")
}

func TestCancelDiscards(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)
	_, _ = svc.ListToggle(ctx, 7, "go")
	v, _ := svc.CancelList(ctx, 7)
	require.Equal(t, ScreenMyWords, v.Screen)
	u, _ := repo.Get(ctx, 7)
	require.NotNil(t, u.State.List, "cancel should clear draft but keep list")
	require.Empty(t, u.State.List.Draft, "cancel should clear draft but keep list")
	require.Equal(t, StatusStudy, u.Words["go"].Status, "cancel must not change words")
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
	require.Equal(t, StatusStudy, u.State.List.Draft["build"])

	// tap again -> back to stored skipped, draft entry removed
	_, _ = svc.ListToggle(ctx, 7, "build")
	u, _ = repo.Get(ctx, 7)
	require.NotContains(t, u.State.List.Draft, "build", "build draft should be cleared")
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
	require.Equal(t, StatusSkipped, u.State.List.Draft["go"])

	// commit: apply skipped to words, stay on list
	_, err := svc.CommitList(ctx, 7)
	require.NoError(t, err)
	u, _ = repo.Get(ctx, 7)
	require.Equal(t, StatusSkipped, u.Words["go"].Status)
	require.NotNil(t, u.State.List)
	require.Empty(t, u.State.List.Draft, "draft must be cleared, list kept")
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
	require.Equal(t, StatusLearned, u.State.List.Draft["go"])

	_, err := svc.CommitList(ctx, 7)
	require.NoError(t, err)
	u, _ = repo.Get(ctx, 7)
	got := u.Words["go"]
	require.Equal(t, StatusLearned, got.Status)
	require.Equal(t, 2, got.Mode)
	require.Equal(t, BoxMax, got.Box)
}

func TestMyWordsSkipDraftStaysVisibleUntilCommit(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	_ = repo.Save(ctx, &User{
		ID:       7,
		Settings: Settings{Variant: "gb"},
		Words:    map[string]WordProgress{"go": {Status: StatusStudy}},
	})
	svc := New(repo, testCatalog())

	_, _ = svc.OpenMyWords(ctx, 7)
	// study -> learned -> skipped (draft)
	_, _ = svc.ListToggle(ctx, 7, "go")
	v, _ := svc.ListToggle(ctx, 7, "go")
	// still visible with the skipped icon, because membership uses stored status
	require.Len(t, v.List.Items, 1, "drafted-skip word must stay visible")
	require.Equal(t, "go", v.List.Items[0].Base)
	require.Equal(t, StatusSkipped, v.List.Items[0].Status)
	// after commit it leaves the list
	v, _ = svc.CommitList(ctx, 7)
	require.Empty(t, v.List.Items, "after commit skipped word must be gone")
}
